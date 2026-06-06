import type { JukeboxSnapshot, Session, Song } from "./types"

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
    request<Song>("/api/queue", { method: "POST", body: JSON.stringify({ url }) }),

  removeSong: (id: string) => request<void>(`/api/queue/${id}`, { method: "DELETE" }),

  reorder: (songId: string, targetIndex: number) =>
    request<void>("/api/queue/reorder", {
      method: "POST",
      body: JSON.stringify({ songId, targetIndex }),
    }),

  play: () => request<void>("/api/player/play", { method: "POST" }),
  pause: () => request<void>("/api/player/pause", { method: "POST" }),
  skip: () => request<void>("/api/player/skip", { method: "POST" }),
  setVolume: (volume: number) =>
    request<void>("/api/player/volume", { method: "POST", body: JSON.stringify({ volume }) }),
}

export { ApiError }
