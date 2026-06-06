// Package api is the HTTP/WebSocket layer of the desktop server. It wires the queue, player,
// auth, and presence hub behind a small REST + WS surface that is byte-compatible with the
// Android server, so the same React remote drives either backend.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"github.com/DiyRex/LetzPlay/desktop/internal/youtube"
)

// Server holds the HTTP dependency graph. Everything is injected — nothing global — so handlers
// stay testable.
type Server struct {
	queue    *domain.Queue
	player   domain.Player
	auth     *auth.Service
	sessions *auth.SessionManager
	meta     *youtube.MetadataClient
	hub      *Hub
	assets   fs.FS
}

// NewServer builds the server. `assets` is the SPA filesystem (the embedded web/dist).
func NewServer(
	queue *domain.Queue,
	player domain.Player,
	authService *auth.Service,
	sessions *auth.SessionManager,
	hub *Hub,
	assets fs.FS,
) *Server {
	return &Server{
		queue:    queue,
		player:   player,
		auth:     authService,
		sessions: sessions,
		meta:     youtube.NewMetadataClient(),
		hub:      hub,
		assets:   assets,
	}
}

// Handler builds the route table. API and WS routes are registered explicitly; everything else
// falls through to the SPA host.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)

	mux.HandleFunc("GET /api/queue", s.protected(s.handleGetQueue))
	mux.HandleFunc("POST /api/queue", s.protected(s.handleAddSong))
	mux.HandleFunc("DELETE /api/queue/{id}", s.protected(s.handleRemoveSong))
	mux.HandleFunc("POST /api/queue/reorder", s.protected(s.handleReorder))

	// Transport + volume are available to any logged-in user (shared "party remote" model).
	// Role still gates managing *other people's* songs (remove/reorder).
	mux.HandleFunc("POST /api/player/play", s.protected(s.handlePlay))
	mux.HandleFunc("POST /api/player/pause", s.protected(s.handlePause))
	mux.HandleFunc("POST /api/player/skip", s.protected(s.handleSkip))
	mux.HandleFunc("POST /api/player/volume", s.protected(s.handleVolume))

	mux.HandleFunc("GET /ws", s.protected(s.handleWS))

	mux.HandleFunc("GET /", s.handleStatic)
	return mux
}

// --- auth handlers ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	role, ok := s.auth.Authenticate(req.Password)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Incorrect password")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		writeErr(w, http.StatusBadRequest, "A name is required")
		return
	}
	s.sessions.Write(w, auth.Session{Username: username, Role: role})
	writeJSON(w, http.StatusOK, sessionResponse{Username: username, Role: role})
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	s.sessions.Clear(w)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessions.Read(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "Not logged in")
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{Username: session.Username, Role: session.Role})
}

// --- queue handlers ---

func (s *Server) handleGetQueue(w http.ResponseWriter, _ *http.Request, _ auth.Session) {
	writeJSON(w, http.StatusOK, s.queue.Snapshot())
}

// maxPlaylistEntries caps how many tracks a single playlist add can enqueue.
const maxPlaylistEntries = 100

func (s *Server) handleAddSong(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var req addSongRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}

	// Playlist link → expand into many songs (desktop has yt-dlp). A plain video → one song.
	if youtube.IsPlaylistURL(req.URL) {
		s.addPlaylist(w, r, session, req.URL)
		return
	}

	videoID, ok := youtube.ExtractVideoID(req.URL)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Not a valid YouTube link")
		return
	}
	title := videoID
	var thumb string
	if meta, ok := s.meta.Fetch(r.Context(), videoID); ok {
		title = meta.Title
		thumb = meta.ThumbnailURL
	}
	song := s.makeSong(videoID, title, thumb, session.Username)
	s.queue.Add(song)
	writeJSON(w, http.StatusCreated, addResult{Added: 1, Song: song})
}

// addPlaylist enqueues every entry of a playlist URL. The first added song auto-plays via the
// queue's promotion rule; the rest follow. Titles come from yt-dlp's flat listing (no per-video
// network call), and thumbnails are derived from the video id.
func (s *Server) addPlaylist(w http.ResponseWriter, r *http.Request, session auth.Session, url string) {
	entries, err := youtube.ExpandPlaylist(r.Context(), url, maxPlaylistEntries)
	if err != nil || len(entries) == 0 {
		writeErr(w, http.StatusBadRequest, "Could not read that playlist")
		return
	}
	var first domain.Song
	for i, e := range entries {
		title := e.Title
		if title == "" {
			title = e.VideoID
		}
		thumb := "https://i.ytimg.com/vi/" + e.VideoID + "/hqdefault.jpg"
		song := s.makeSong(e.VideoID, title, thumb, session.Username)
		if i == 0 {
			first = song
		}
		s.queue.Add(song)
	}
	writeJSON(w, http.StatusCreated, addResult{Added: len(entries), Song: first})
}

func (s *Server) makeSong(videoID, title, thumb, addedBy string) domain.Song {
	return domain.Song{
		ID:             newID(),
		VideoID:        videoID,
		Title:          title,
		ThumbnailURL:   thumb,
		AddedBy:        addedBy,
		AddedAtEpochMs: time.Now().UnixMilli(),
	}
}

func (s *Server) handleRemoveSong(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := r.PathValue("id")
	owner := s.queue.OwnerOf(id)
	if owner == "" {
		writeErr(w, http.StatusNotFound, "Song not in queue")
		return
	}
	if !session.Role.IsAdmin() && owner != session.Username {
		writeErr(w, http.StatusForbidden, "You can only remove songs you added")
		return
	}
	s.queue.Remove(id)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReorder(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var req reorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	owner := s.queue.OwnerOf(req.SongID)
	if owner == "" {
		writeErr(w, http.StatusNotFound, "Song not in queue")
		return
	}
	if !session.Role.IsAdmin() && owner != session.Username {
		writeErr(w, http.StatusForbidden, "You can only reorder songs you added")
		return
	}
	s.queue.Reorder(req.SongID, req.TargetIndex)
	w.WriteHeader(http.StatusOK)
}

// --- player handlers (admin only) ---

func (s *Server) handlePlay(w http.ResponseWriter, _ *http.Request, _ auth.Session) {
	s.player.Play()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePause(w http.ResponseWriter, _ *http.Request, _ auth.Session) {
	s.player.Pause()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSkip(w http.ResponseWriter, _ *http.Request, _ auth.Session) {
	s.queue.Advance()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleVolume(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req volumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.player.SetVolume(req.Volume)
	s.queue.SetVolume(req.Volume)
	w.WriteHeader(http.StatusOK)
}

// --- websocket ---

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request, session auth.Session) {
	s.hub.ServeWS(w, r, ConnectedUser{Username: session.Username, Role: session.Role})
}

// --- static SPA host ---

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	data, err := fs.ReadFile(s.assets, path)
	if err != nil {
		// SPA fallback: unknown route -> index.html so client routing survives a refresh.
		data, err = fs.ReadFile(s.assets, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		path = "index.html"
	}
	w.Header().Set("Content-Type", contentType(path))
	_, _ = w.Write(data)
}

// --- middleware + helpers ---

type sessionHandler func(http.ResponseWriter, *http.Request, auth.Session)

func (s *Server) protected(h sessionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessions.Read(r)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "Login required")
			return
		}
		h(w, r, session)
	}
}

func (s *Server) adminOnly(h sessionHandler) http.HandlerFunc {
	return s.protected(func(w http.ResponseWriter, r *http.Request, session auth.Session) {
		if !session.Role.IsAdmin() {
			writeErr(w, http.StatusForbidden, "Admin only")
			return
		}
		h(w, r, session)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".mjs"):
		return "application/javascript"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}

// ErrNoAssets is returned by embedded asset loaders when the web bundle is missing.
var ErrNoAssets = errors.New("web assets not built; run scripts/build-web.sh")
