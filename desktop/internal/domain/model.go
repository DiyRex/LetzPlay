// Package domain holds the core jukebox types and queue logic. It is pure: no HTTP, no player,
// no OS calls — which keeps it trivially testable and free of the coupling that breeds bugs.
//
// The JSON tags here intentionally match the Android (Kotlin) server and the React client's
// TypeScript types byte-for-byte, so one web remote talks to either backend unchanged.
package domain

// PlaybackStatus mirrors the coarse states reported by the player.
type PlaybackStatus string

const (
	StatusIdle      PlaybackStatus = "IDLE"
	StatusBuffering PlaybackStatus = "BUFFERING"
	StatusPlaying   PlaybackStatus = "PLAYING"
	StatusPaused    PlaybackStatus = "PAUSED"
	StatusEnded     PlaybackStatus = "ENDED"
)

// RepeatMode controls what happens at the end of a track.
type RepeatMode string

const (
	RepeatOff RepeatMode = "OFF" // stop at the end of the list
	RepeatAll RepeatMode = "ALL" // wrap back to the first track
	RepeatOne RepeatMode = "ONE" // loop the current track
)

// Role is the authorization tier resolved at login.
type Role string

const (
	RoleGuest Role = "GUEST"
	RoleAdmin Role = "ADMIN"
)

// IsAdmin reports whether the role may control playback and manage any song.
func (r Role) IsAdmin() bool { return r == RoleAdmin }

// Song is one immutable queue entry.
type Song struct {
	ID             string `json:"id"`
	VideoID        string `json:"videoId"`
	Title          string `json:"title"`
	ThumbnailURL   string `json:"thumbnailUrl,omitempty"`
	AddedBy        string `json:"addedBy"`
	AddedAtEpochMs int64  `json:"addedAtEpochMs"`
}

// Snapshot is the full, serializable jukebox state broadcast to every remote.
//
// The model is a persistent playlist with a moving cursor — NOT a consumed queue. `Tracks` holds
// every added song in order (played, current, and upcoming all stay in the list); `CurrentIndex`
// points at the one playing (or -1 when nothing is selected). Advancing/Previous/jumping just move
// the cursor; songs are only removed by an explicit delete.
type Snapshot struct {
	Tracks          []Song         `json:"tracks"`
	CurrentIndex    int            `json:"currentIndex"`
	Status          PlaybackStatus `json:"status"`
	PositionSeconds float64        `json:"positionSeconds"`
	DurationSeconds float64        `json:"durationSeconds"`
	Volume          int            `json:"volume"`
	Shuffle         bool           `json:"shuffle"`
	Repeat          RepeatMode     `json:"repeat"`
	Locked          bool           `json:"locked"`   // admin queue lock: only admins may add
	Autoplay        bool           `json:"autoplay"` // radio: auto-add a related track when empty
}

// Current returns the playing track, or nil when CurrentIndex is out of range.
func (s Snapshot) Current() *Song {
	if s.CurrentIndex < 0 || s.CurrentIndex >= len(s.Tracks) {
		return nil
	}
	t := s.Tracks[s.CurrentIndex]
	return &t
}

// Player is the abstraction over the actual audio player. The server depends on this, never on
// the concrete mpv implementation, so playback can be swapped or faked in tests.
type Player interface {
	Load(videoID string)
	Play()
	Pause()
	Seek(seconds float64)
	SetVolume(percent int)
	SetLoop(loop bool) // loop the current file (used for RepeatOne)
}
