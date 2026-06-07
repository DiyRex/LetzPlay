package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
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
	notify     chan struct{} // request an immediate re-broadcast (e.g. after a vote)
	upgrader   websocket.Upgrader
	users      atomic.Value // []ConnectedUser, for the TUI to read lock-free

	votesMu sync.Mutex
	voteKey string          // videoId the current votes apply to
	voters  map[string]bool // usernames that voted to skip
	sleepAt atomic.Int64    // epoch ms to auto-pause (0 = off)
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
		notify:     make(chan struct{}, 1),
		// Same-origin only: the page is served by this very server.
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
		voters:   make(map[string]bool),
	}
	h.users.Store([]ConnectedUser{})
	return h
}

// VoteSkip records a vote to skip videoID by user. Votes reset automatically when the track
// changes. Returns the current vote count, the number needed (majority of connected users), and
// whether the threshold was reached.
func (h *Hub) VoteSkip(videoID, user string) (votes, needed int, reached bool) {
	h.votesMu.Lock()
	if h.voteKey != videoID {
		h.voteKey = videoID
		h.voters = make(map[string]bool)
	}
	h.voters[user] = true
	votes = len(h.voters)
	h.votesMu.Unlock()

	needed = skipThreshold(len(h.Users()))
	return votes, needed, votes >= needed
}

// ResetVotes clears skip votes (called after a skip).
func (h *Hub) ResetVotes() {
	h.votesMu.Lock()
	h.voteKey = ""
	h.voters = make(map[string]bool)
	h.votesMu.Unlock()
}

// SetSleepAt records when playback should auto-pause (epoch ms; 0 = off) for broadcast.
func (h *Hub) SetSleepAt(ms int64) { h.sleepAt.Store(ms) }

func (h *Hub) votesFor(videoID string) int {
	h.votesMu.Lock()
	defer h.votesMu.Unlock()
	if h.voteKey != videoID {
		return 0
	}
	return len(h.voters)
}

// skipThreshold is a simple majority of connected users (at least 1).
func skipThreshold(users int) int {
	if users < 1 {
		return 1
	}
	return users/2 + 1
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
		case <-h.notify:
			h.broadcast(h.queue.Snapshot())
		}
	}
}

// Broadcast requests an immediate re-broadcast of current state (safe to call from any goroutine).
// Used after a vote so all clients see the new count instantly instead of waiting for a tick.
func (h *Hub) Broadcast() {
	select {
	case h.notify <- struct{}{}:
	default: // a broadcast is already pending
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
	currentVideoID := ""
	if c := snap.Current(); c != nil {
		currentVideoID = c.VideoID
	}
	payload, err := json.Marshal(LiveState{
		Snapshot:   snap,
		Users:      h.Users(),
		SkipVotes:  h.votesFor(currentVideoID),
		SkipNeeded: skipThreshold(len(h.Users())),
		SleepAtMs:  h.sleepAt.Load(),
	})
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
