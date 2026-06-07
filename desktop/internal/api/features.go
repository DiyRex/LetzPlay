package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/youtube"
)

// --- search ---

type searchResultDTO struct {
	VideoID      string `json:"videoId"`
	Title        string `json:"title"`
	Channel      string `json:"channel"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, []searchResultDTO{})
		return
	}
	results, err := youtube.Search(r.Context(), query, 15)
	if err != nil {
		// Most likely yt-dlp isn't available (e.g. Android server) — return empty, not an error.
		writeJSON(w, http.StatusOK, []searchResultDTO{})
		return
	}
	out := make([]searchResultDTO, 0, len(results))
	for _, res := range results {
		out = append(out, searchResultDTO{
			VideoID:      res.VideoID,
			Title:        res.Title,
			Channel:      res.Channel,
			ThumbnailURL: "https://i.ytimg.com/vi/" + res.VideoID + "/mqdefault.jpg",
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// --- lyrics ---

func (s *Server) handleLyrics(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	videoID := r.URL.Query().Get("videoId")
	title := ""
	for _, t := range s.queue.Snapshot().Tracks {
		if t.VideoID == videoID {
			title = t.Title
			break
		}
	}
	if title == "" {
		title = r.URL.Query().Get("title") // fallback if the track isn't in the queue
	}
	if strings.TrimSpace(title) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"found": false})
		return
	}
	writeJSON(w, http.StatusOK, s.lyrics.Get(r.Context(), title))
}

// --- vote to skip ---

func (s *Server) handleVoteSkip(w http.ResponseWriter, _ *http.Request, session auth.Session) {
	current := s.queue.Snapshot().Current()
	if current == nil {
		writeErr(w, http.StatusConflict, "Nothing is playing")
		return
	}
	_, _, reached := s.hub.VoteSkip(current.VideoID, session.Username)
	if reached {
		s.queue.Advance()
		s.hub.ResetVotes()
	}
	w.WriteHeader(http.StatusOK)
}

// --- sleep timer ---

type sleepRequest struct {
	Minutes int `json:"minutes"` // 0 cancels
}

func (s *Server) handleSleep(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req sleepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.setSleep(req.Minutes)
	w.WriteHeader(http.StatusOK)
}

// setSleep schedules (or cancels) the auto-pause. minutes<=0 cancels.
func (s *Server) setSleep(minutes int) {
	s.sleepMu.Lock()
	defer s.sleepMu.Unlock()
	if s.sleepTimer != nil {
		s.sleepTimer.Stop()
		s.sleepTimer = nil
	}
	if minutes <= 0 {
		s.hub.SetSleepAt(0)
		return
	}
	d := time.Duration(minutes) * time.Minute
	s.hub.SetSleepAt(time.Now().Add(d).UnixMilli())
	s.sleepTimer = time.AfterFunc(d, func() {
		s.player.Pause()
		s.hub.SetSleepAt(0)
	})
}

// --- autoplay (radio) toggle ---

type autoplayRequest struct {
	Autoplay bool `json:"autoplay"`
}

func (s *Server) handleAutoplay(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req autoplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.queue.SetAutoplay(req.Autoplay)
	w.WriteHeader(http.StatusOK)
}

// --- admin ---

type lockRequest struct {
	Locked bool `json:"locked"`
}

func (s *Server) handleLock(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req lockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.queue.SetLocked(req.Locked)
	w.WriteHeader(http.StatusOK)
}

type passwordRequest struct {
	Admin string `json:"admin"`
	Guest string `json:"guest"`
}

func (s *Server) handlePassword(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req passwordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	if strings.TrimSpace(req.Admin) != "" {
		if err := s.auth.SetAdminPassword(req.Admin); err != nil {
			writeErr(w, http.StatusInternalServerError, "Could not update admin password")
			return
		}
	}
	if strings.TrimSpace(req.Guest) != "" {
		if err := s.auth.SetGuestPassword(req.Guest); err != nil {
			writeErr(w, http.StatusInternalServerError, "Could not update guest password")
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
