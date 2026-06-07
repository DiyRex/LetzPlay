import { useEffect, useRef, useState } from "react"
import { api } from "@/api/client"
import type { ConnectedUser, JukeboxSnapshot, LiveState } from "@/api/types"

const EMPTY_SNAPSHOT: JukeboxSnapshot = {
  tracks: [],
  currentIndex: -1,
  status: "IDLE",
  positionSeconds: 0,
  durationSeconds: 0,
  volume: 100,
  shuffle: false,
  repeat: "OFF",
}

/**
 * Live jukebox state over a WebSocket. The server pushes a full {@link LiveState} (snapshot +
 * presence) on every change — queue change *or* someone connecting/disconnecting — so the client
 * never reconciles diffs; it renders whatever arrived last. Drops are handled by auto-reconnect
 * with backoff; a REST fetch seeds the first render in case the socket is slow.
 *
 * @param enabled gate the connection on auth — no point opening a socket before login.
 */
export function useJukebox(enabled: boolean): { snapshot: JukeboxSnapshot; users: ConnectedUser[] } {
  const [snapshot, setSnapshot] = useState<JukeboxSnapshot>(EMPTY_SNAPSHOT)
  const [users, setUsers] = useState<ConnectedUser[]>([])
  const socketRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!enabled) return
    let closed = false
    let retry = 0
    let reconnectTimer: ReturnType<typeof setTimeout>

    // Seed snapshot immediately so the UI isn't blank while the socket handshakes.
    api.getQueue().then(setSnapshot).catch(() => undefined)

    const connect = () => {
      if (closed) return
      const scheme = location.protocol === "https:" ? "wss" : "ws"
      const ws = new WebSocket(`${scheme}://${location.host}/ws`)
      socketRef.current = ws

      ws.onmessage = (event) => {
        try {
          const state = JSON.parse(event.data) as LiveState
          setSnapshot(state.snapshot)
          setUsers(state.users)
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
        reconnectTimer = setTimeout(connect, 500 * retry) // linear backoff, capped at 3s
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

  return { snapshot, users }
}
