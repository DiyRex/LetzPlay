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

type sessionResponse struct {
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// ConnectedUser is one present remote, shown in the web "who's here" panel.
type ConnectedUser struct {
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
}

// LiveState is the websocket payload: the jukebox snapshot plus live presence. Presence is a
// transport concern, so it is merged in here at broadcast time rather than stored in the queue.
type LiveState struct {
	Snapshot domain.Snapshot `json:"snapshot"`
	Users    []ConnectedUser `json:"users"`
}
