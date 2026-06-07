// Wire types — kept in sync with the server DTOs (Kotlin `server/dto` and Go `internal/api`).
// Both backends serialize identical JSON so this one file describes both.

export type Role = "GUEST" | "ADMIN"

export type PlaybackStatus = "IDLE" | "BUFFERING" | "PLAYING" | "PAUSED" | "ENDED"

export type RepeatMode = "OFF" | "ALL" | "ONE"

export interface Song {
  id: string
  videoId: string
  title: string
  thumbnailUrl?: string | null
  addedBy: string
  addedAtEpochMs: number
}

/**
 * Persistent playlist with a moving cursor (not a consumed queue): `tracks` holds every added song
 * — played, current, and upcoming — and `currentIndex` points at the one playing (-1 when none).
 */
export interface JukeboxSnapshot {
  tracks: Song[]
  currentIndex: number
  status: PlaybackStatus
  positionSeconds: number
  durationSeconds: number
  volume: number
  shuffle: boolean
  repeat: RepeatMode
  locked: boolean
  autoplay: boolean
}

/** A YouTube search hit (desktop server, via yt-dlp). */
export interface SearchResult {
  videoId: string
  title: string
  channel: string
  thumbnailUrl: string
}

export interface LyricsLine {
  timeMs: number
  text: string
}

export interface Lyrics {
  found: boolean
  synced: LyricsLine[]
  plain: string
}

/** A song stored in a saved playlist (lighter than a queue Song). */
export interface PlaylistSong {
  videoId: string
  title: string
  thumbnailUrl?: string
}

/** List-view form of a playlist. */
export interface PlaylistSummary {
  id: string
  name: string
  count: number
}

/** A full saved playlist with its songs. */
export interface Playlist {
  id: string
  name: string
  songs: PlaylistSong[]
  updatedAtMs: number
}

export interface Session {
  username: string
  role: Role
}

/** Response to adding a song/playlist: how many were queued, plus the representative track. */
export interface AddResult {
  added: number
  song: Song
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
