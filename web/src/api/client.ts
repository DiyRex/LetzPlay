import type {
  AddResult,
  JukeboxSnapshot,
  Playlist,
  PlaylistSummary,
  RepeatMode,
  Session,
} from "./types"

/**
 * Thin typed wrapper over the server REST API. Every call sends the session cookie
 * (`credentials: "include"`) and surfaces server error messages as thrown `Error`s, so callers
 * can simply `try/catch` and show a toast.
 */

class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message)
    this.name = "ApiError"
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "include",
    headers: init?.body ? { "Content-Type": "application/json" } : undefined,
    ...init,
  })
  if (!res.ok) {
    const message = await res
      .json()
      .then((b) => (b as { error?: string }).error)
      .catch(() => undefined)
    throw new ApiError(message ?? `Request failed (${res.status})`, res.status)
  }
  // Some endpoints return an empty 200/201; guard the JSON parse.
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

export const api = {
  login: (username: string, password: string) =>
    request<Session>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),

  logout: () => request<void>("/api/auth/logout", { method: "POST" }),

  me: () => request<Session>("/api/auth/me"),

  getQueue: () => request<JukeboxSnapshot>("/api/queue"),

  addSong: (url: string) =>
    request<AddResult>("/api/queue", { method: "POST", body: JSON.stringify({ url }) }),

  removeSong: (id: string) => request<void>(`/api/queue/${id}`, { method: "DELETE" }),

  reorder: (songId: string, targetIndex: number) =>
    request<void>("/api/queue/reorder", {
      method: "POST",
      body: JSON.stringify({ songId, targetIndex }),
    }),

  // Tap-to-play: jump the cursor straight to a song already in the list.
  playNow: (id: string) => request<void>(`/api/queue/${id}/play`, { method: "POST" }),

  play: () => request<void>("/api/player/play", { method: "POST" }),
  pause: () => request<void>("/api/player/pause", { method: "POST" }),
  skip: () => request<void>("/api/player/skip", { method: "POST" }),
  previous: () => request<void>("/api/player/previous", { method: "POST" }),
  seek: (seconds: number) =>
    request<void>("/api/player/seek", { method: "POST", body: JSON.stringify({ seconds }) }),
  setShuffle: (shuffle: boolean) =>
    request<void>("/api/player/shuffle", { method: "POST", body: JSON.stringify({ shuffle }) }),
  setRepeat: (repeat: RepeatMode) =>
    request<void>("/api/player/repeat", { method: "POST", body: JSON.stringify({ repeat }) }),
  clearQueue: () => request<void>("/api/player/clear", { method: "POST" }),

  // Playlists
  listPlaylists: () => request<PlaylistSummary[]>("/api/playlists"),
  createPlaylist: (name: string) =>
    request<Playlist>("/api/playlists", { method: "POST", body: JSON.stringify({ name }) }),
  saveQueueAsPlaylist: (name: string) =>
    request<Playlist>("/api/playlists/save-queue", { method: "POST", body: JSON.stringify({ name }) }),
  getPlaylist: (id: string) => request<Playlist>(`/api/playlists/${id}`),
  deletePlaylist: (id: string) => request<void>(`/api/playlists/${id}`, { method: "DELETE" }),
  addPlaylistSong: (id: string, url: string) =>
    request<Playlist>(`/api/playlists/${id}/songs`, { method: "POST", body: JSON.stringify({ url }) }),
  removePlaylistSong: (id: string, videoId: string) =>
    request<void>(`/api/playlists/${id}/songs/${videoId}`, { method: "DELETE" }),
  enqueuePlaylist: (id: string) =>
    request<AddResult>(`/api/playlists/${id}/enqueue`, { method: "POST" }),
  setVolume: (volume: number) =>
    request<void>("/api/player/volume", { method: "POST", body: JSON.stringify({ volume }) }),
}

export { ApiError }
