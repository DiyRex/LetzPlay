import { useEffect } from "react"
import { api } from "@/api/client"
import type { JukeboxSnapshot } from "@/api/types"

/**
 * Wires the browser's Media Session API so the phone lock screen, notification, and headphone
 * buttons control the jukebox: shows the current track + artwork and maps play/pause/next/prev/seek
 * to the server. No-ops where the API is unsupported.
 */
export function useMediaSession(snapshot: JukeboxSnapshot) {
  const current = snapshot.tracks[snapshot.currentIndex] ?? null

  // Metadata + playback state follow the current track.
  useEffect(() => {
    if (!("mediaSession" in navigator)) return
    if (current) {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: current.title,
        artist: `added by ${current.addedBy}`,
        artwork: current.thumbnailUrl
          ? [{ src: current.thumbnailUrl, sizes: "480x360", type: "image/jpeg" }]
          : [],
      })
    }
    navigator.mediaSession.playbackState =
      snapshot.status === "PLAYING" || snapshot.status === "BUFFERING" ? "playing" : "paused"
  }, [current, snapshot.status])

  // Position (for the lock-screen scrubber). Guarded — some browsers throw on bad values.
  useEffect(() => {
    if (!("mediaSession" in navigator) || !("setPositionState" in navigator.mediaSession)) return
    if (snapshot.durationSeconds > 0) {
      try {
        navigator.mediaSession.setPositionState({
          duration: snapshot.durationSeconds,
          position: Math.min(snapshot.positionSeconds, snapshot.durationSeconds),
          playbackRate: snapshot.speed || 1,
        })
      } catch {
        /* ignore */
      }
    }
  }, [snapshot.positionSeconds, snapshot.durationSeconds, snapshot.speed])

  // Action handlers (set once).
  useEffect(() => {
    if (!("mediaSession" in navigator)) return
    const ignore = () => undefined
    const set = (action: MediaSessionAction, handler: MediaSessionActionHandler | null) => {
      try {
        navigator.mediaSession.setActionHandler(action, handler)
      } catch {
        /* unsupported action */
      }
    }
    set("play", () => void api.play().catch(ignore))
    set("pause", () => void api.pause().catch(ignore))
    set("nexttrack", () => void api.skip().catch(ignore))
    set("previoustrack", () => void api.previous().catch(ignore))
    set("seekto", (details) => {
      if (typeof details.seekTime === "number") void api.seek(details.seekTime).catch(ignore)
    })
    return () => {
      ;(["play", "pause", "nexttrack", "previoustrack", "seekto"] as MediaSessionAction[]).forEach((a) =>
        set(a, null),
      )
    }
  }, [])
}
