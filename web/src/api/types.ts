// Wire types — kept in sync with the server DTOs (Kotlin `server/dto` and Go `internal/api`).
// Both backends serialize identical JSON so this one file describes both.

export type Role = "GUEST" | "ADMIN"

export type PlaybackStatus = "IDLE" | "BUFFERING" | "PLAYING" | "PAUSED" | "ENDED"

export interface Song {
  id: string
  videoId: string
  title: string
  thumbnailUrl?: string | null
  addedBy: string
  addedAtEpochMs: number
}

export interface JukeboxSnapshot {
  nowPlaying: Song | null
  queue: Song[]
  status: PlaybackStatus
  positionSeconds: number
  durationSeconds: number
  volume: number
}

export interface Session {
  username: string
  role: Role
}

/** A connected remote, shown in the "who's here" panel. */
export interface ConnectedUser {
  username: string
  role: Role
}

/** WebSocket payload: jukebox snapshot plus live presence (matches both backends). */
export interface LiveState {
  snapshot: JukeboxSnapshot
  users: ConnectedUser[]
}
