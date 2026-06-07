package api

import "github.com/DiyRex/LetzPlay/desktop/internal/domain"

// Wire contracts shared with the React remote and kept identical to the Android server's DTOs.

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type addSongRequest struct {
	URL string `json:"url"`
}

type reorderRequest struct {
	SongID      string `json:"songId"`
	TargetIndex int    `json:"targetIndex"`
}

type volumeRequest struct {
	Volume int `json:"volume"`
}

type seekRequest struct {
	Seconds float64 `json:"seconds"`
}

type shuffleRequest struct {
	Shuffle bool `json:"shuffle"`
}

type repeatRequest struct {
	Repeat domain.RepeatMode `json:"repeat"`
}

type normalizeRequest struct {
	Normalize bool `json:"normalize"`
}

type eqRequest struct {
	Eq string `json:"eq"`
}

type speedRequest struct {
	Speed float64 `json:"speed"`
}

type fairQueueRequest struct {
	FairQueue bool `json:"fairQueue"`
}

type sessionResponse struct {
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
}

// addResult is the response to POST /api/queue. `added` is how many tracks were queued (>1 for a
// playlist); `song` is the first/representative track. Same shape on the Android server.
type addResult struct {
	Added int         `json:"added"`
	Song  domain.Song `json:"song"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// ConnectedUser is one present remote, shown in the web "who's here" panel.
type ConnectedUser struct {
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
}

// LiveState is the websocket payload: the jukebox snapshot plus live, presence-derived state
// (who's connected, skip votes, sleep timer). These are transport concerns merged in at broadcast
// time rather than stored in the queue.
type LiveState struct {
	Snapshot   domain.Snapshot `json:"snapshot"`
	Users      []ConnectedUser `json:"users"`
	SkipVotes  int             `json:"skipVotes"`  // votes to skip the current track
	SkipNeeded int             `json:"skipNeeded"` // votes required (majority of connected users)
	SleepAtMs  int64           `json:"sleepAtMs"`  // epoch ms when playback auto-pauses (0 = off)
}
