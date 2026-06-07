package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
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

// handleRadioFromSong seeds YouTube's "mix" from a chosen song and appends related tracks — an
// on-demand version of autoplay radio.
func (s *Server) handleRadioFromSong(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := r.PathValue("id")
	var seed string
	for _, t := range s.queue.Snapshot().Tracks {
		if t.ID == id {
			seed = t.VideoID
			break
		}
	}
	if seed == "" {
		writeErr(w, http.StatusNotFound, "Song not in the list")
		return
	}
	mix, err := youtube.RadioMix(r.Context(), seed, 8)
	if err != nil || len(mix) == 0 {
		writeErr(w, http.StatusBadGateway, "Couldn't build a radio mix for that song")
		return
	}
	existing := map[string]bool{}
	for _, t := range s.queue.Snapshot().Tracks {
		existing[t.VideoID] = true
	}
	added := 0
	for _, e := range mix {
		if existing[e.VideoID] {
			continue
		}
		s.queue.Add(s.makeSong(e.VideoID, e.Title, "https://i.ytimg.com/vi/"+e.VideoID+"/hqdefault.jpg", session.Username))
		added++
	}
	writeJSON(w, http.StatusOK, addResult{Added: added, Song: domain.Song{}})
}

// --- audio: normalize / equalizer / speed / fair queue ---

// audioFilter builds an mpv/ffmpeg audio-filter chain from the EQ preset + normalization toggle.
func audioFilter(eq string, normalize bool) string {
	parts := make([]string, 0, 2)
	switch eq {
	case "bass":
		parts = append(parts, "bass=g=10")
	case "treble":
		parts = append(parts, "treble=g=8")
	case "vocal":
		parts = append(parts, "equalizer=f=2500:t=q:w=1.5:g=5")
	case "loud":
		parts = append(parts, "bass=g=6", "treble=g=4")
	}
	if normalize {
		parts = append(parts, "dynaudnorm=g=15") // smooth dynamic loudness normalization
	}
	return strings.Join(parts, ",")
}

func (s *Server) applyAudio() {
	snap := s.queue.Snapshot()
	s.player.SetAudioFilters(audioFilter(snap.Eq, snap.Normalize))
}

func (s *Server) handleNormalize(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req normalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.queue.SetNormalize(req.Normalize)
	s.applyAudio()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleEq(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req eqRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.queue.SetEqualizer(req.Eq)
	s.applyAudio()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSpeed(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req speedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.queue.SetSpeed(req.Speed)
	s.player.SetSpeed(s.queue.Snapshot().Speed)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleFairQueue(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	var req fairQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	s.queue.SetFairQueue(req.FairQueue)
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
