package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"github.com/DiyRex/LetzPlay/desktop/internal/playlist"
	"github.com/DiyRex/LetzPlay/desktop/internal/youtube"
)

type namedRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleListPlaylists(w http.ResponseWriter, _ *http.Request, _ auth.Session) {
	writeJSON(w, http.StatusOK, s.playlists.List())
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	name, ok := decodeName(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusCreated, s.playlists.Create(name, []playlist.Song{}))
}

// handleSaveQueueAsPlaylist snapshots the current queue into a new playlist.
func (s *Server) handleSaveQueueAsPlaylist(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	name, ok := decodeName(w, r)
	if !ok {
		return
	}
	tracks := s.queue.Snapshot().Tracks
	songs := make([]playlist.Song, 0, len(tracks))
	for _, t := range tracks {
		songs = append(songs, playlist.Song{VideoID: t.VideoID, Title: t.Title, ThumbnailURL: t.ThumbnailURL})
	}
	if len(songs) == 0 {
		writeErr(w, http.StatusBadRequest, "The queue is empty")
		return
	}
	writeJSON(w, http.StatusCreated, s.playlists.Create(name, songs))
}

func (s *Server) handleGetPlaylist(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	p, ok := s.playlists.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "Playlist not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	if !s.playlists.Delete(r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "Playlist not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleAddPlaylistSong resolves a YouTube link and adds it to the playlist.
func (s *Server) handleAddPlaylistSong(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	id := r.PathValue("id")
	var req addSongRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return
	}
	videoID, ok := youtube.ExtractVideoID(req.URL)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Not a valid YouTube link")
		return
	}
	title := videoID
	thumb := "https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg"
	if meta, ok := s.meta.Fetch(r.Context(), videoID); ok {
		title = meta.Title
		if meta.ThumbnailURL != "" {
			thumb = meta.ThumbnailURL
		}
	}
	if !s.playlists.AddSong(id, playlist.Song{VideoID: videoID, Title: title, ThumbnailURL: thumb}) {
		writeErr(w, http.StatusNotFound, "Playlist not found")
		return
	}
	p, _ := s.playlists.Get(id)
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleRemovePlaylistSong(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	if !s.playlists.RemoveSong(r.PathValue("id"), r.PathValue("videoId")) {
		writeErr(w, http.StatusNotFound, "Playlist not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleEnqueuePlaylist appends every song of a playlist to the live queue.
func (s *Server) handleEnqueuePlaylist(w http.ResponseWriter, r *http.Request, session auth.Session) {
	p, ok := s.playlists.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "Playlist not found")
		return
	}
	for _, song := range p.Songs {
		s.queue.Add(domain.Song{
			ID:             newID(),
			VideoID:        song.VideoID,
			Title:          song.Title,
			ThumbnailURL:   song.ThumbnailURL,
			AddedBy:        session.Username,
			AddedAtEpochMs: time.Now().UnixMilli(),
		})
	}
	writeJSON(w, http.StatusOK, addResult{Added: len(p.Songs), Song: domain.Song{}})
}

func decodeName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req namedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Malformed request")
		return "", false
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "A name is required")
		return "", false
	}
	return name, true
}
