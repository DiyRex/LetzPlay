import { useEffect, useRef, useState } from "react"
import { api } from "@/api/client"
import type { ConnectedUser, JukeboxSnapshot } from "@/api/types"

const EMPTY_SNAPSHOT: JukeboxSnapshot = {
  tracks: [],
  currentIndex: -1,
  status: "IDLE",
  positionSeconds: 0,
  durationSeconds: 0,
  volume: 100,
  shuffle: false,
  repeat: "OFF",
  locked: false,
  autoplay: false,
  normalize: false,
  eq: "flat",
  speed: 1,
  fairQueue: false,
}

export interface LiveState {
  snapshot: JukeboxSnapshot
  users: ConnectedUser[]
  skipVotes: number
  skipNeeded: number
  sleepAtMs: number
}

const EMPTY_LIVE: LiveState = {
  snapshot: EMPTY_SNAPSHOT,
  users: [],
  skipVotes: 0,
  skipNeeded: 1,
  sleepAtMs: 0,
}

/**
 * Live jukebox state over a WebSocket. The server pushes a full LiveState (snapshot + presence +
 * skip votes + sleep timer) on every change; the client just renders the latest. Auto-reconnects
 * with backoff; a REST fetch seeds the first snapshot in case the socket is slow.
 */
export function useJukebox(enabled: boolean): LiveState {
  const [live, setLive] = useState<LiveState>(EMPTY_LIVE)
  const socketRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!enabled) return
    let closed = false
    let retry = 0
    let reconnectTimer: ReturnType<typeof setTimeout>

    api
      .getQueue()
      .then((snapshot) => setLive((prev) => ({ ...prev, snapshot })))
      .catch(() => undefined)

    const connect = () => {
      if (closed) return
      const scheme = location.protocol === "https:" ? "wss" : "ws"
      const ws = new WebSocket(`${scheme}://${location.host}/ws`)
      socketRef.current = ws

      ws.onmessage = (event) => {
        try {
          setLive(JSON.parse(event.data) as LiveState)
        } catch {
          /* ignore malformed frame */
        }
      }
      ws.onopen = () => {
        retry = 0
      }
      ws.onclose = () => {
        if (closed) return
        retry = Math.min(retry + 1, 6)
        reconnectTimer = setTimeout(connect, 500 * retry)
      }
      ws.onerror = () => ws.close()
    }

    connect()
    return () => {
      closed = true
      clearTimeout(reconnectTimer)
      socketRef.current?.close()
    }
  }, [enabled])

  return live
}
