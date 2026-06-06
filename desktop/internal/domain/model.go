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
type Snapshot struct {
	NowPlaying      *Song          `json:"nowPlaying"`
	Queue           []Song         `json:"queue"`
	Status          PlaybackStatus `json:"status"`
	PositionSeconds float64        `json:"positionSeconds"`
	DurationSeconds float64        `json:"durationSeconds"`
	Volume          int            `json:"volume"`
}

// Player is the abstraction over the actual audio player. The server depends on this, never on
// the concrete mpv implementation, so playback can be swapped or faked in tests.
type Player interface {
	Load(videoID string)
	Play()
	Pause()
	SetVolume(percent int)
}
