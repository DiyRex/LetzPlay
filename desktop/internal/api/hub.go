package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync/atomic"
	"time"

	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
)

// Hub fans live state out to all connected remotes and tracks presence (who is connected). It
// runs a single goroutine that owns the client set, so no per-client locking is needed.
type Hub struct {
	queue      *domain.Queue
	register   chan *client
	unregister chan *client
	clients    map[*client]bool
	upgrader   websocket.Upgrader
	users      atomic.Value // []ConnectedUser, for the TUI to read lock-free
}

type client struct {
	conn *websocket.Conn
	send chan []byte
	user ConnectedUser
}

// NewHub creates a hub bound to the queue.
func NewHub(queue *domain.Queue) *Hub {
	h := &Hub{
		queue:      queue,
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[*client]bool),
		// Same-origin only: the page is served by this very server.
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
	}
	h.users.Store([]ConnectedUser{})
	return h
}

// Run owns the hub state until ctx is cancelled. Broadcasts happen on queue changes AND on
// presence changes, so the "who's here" list stays as live as the music.
func (h *Hub) Run(ctx context.Context) {
	updates, unsubscribe := h.queue.Subscribe()
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.clients[c] = true
			h.refreshPresence()
			h.broadcast(h.queue.Snapshot())
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.refreshPresence()
			h.broadcast(h.queue.Snapshot())
		case snap := <-updates:
			h.broadcast(snap)
		}
	}
}

// Users returns the current presence list (safe to call from the TUI goroutine).
func (h *Hub) Users() []ConnectedUser {
	return h.users.Load().([]ConnectedUser)
}

// ServeWS upgrades the connection and registers the authenticated user as present.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, user ConnectedUser) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 16), user: user}
	h.register <- c
	go h.writePump(c)
	go h.readPump(c)
}

func (h *Hub) broadcast(snap domain.Snapshot) {
	payload, err := json.Marshal(LiveState{Snapshot: snap, Users: h.Users()})
	if err != nil {
		return
	}
	for c := range h.clients {
		select {
		case c.send <- payload:
		default:
			// Slow client: drop it rather than block the whole hub.
			delete(h.clients, c)
			close(c.send)
		}
	}
}

// refreshPresence recomputes the deduplicated user list (one entry per username, admin wins).
func (h *Hub) refreshPresence() {
	byName := make(map[string]domain.Role)
	for c := range h.clients {
		if existing, ok := byName[c.user.Username]; !ok || c.user.Role.IsAdmin() && !existing.IsAdmin() {
			byName[c.user.Username] = c.user.Role
		}
	}
	users := make([]ConnectedUser, 0, len(byName))
	for name, role := range byName {
		users = append(users, ConnectedUser{Username: name, Role: role})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })
	h.users.Store(users)
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
