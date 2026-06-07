import { useState } from "react"
import { AdminPanel } from "@/components/AdminPanel"
import { AudioControls } from "@/components/AudioControls"
import { BottomNav, type Tab } from "@/components/BottomNav"
import { ConnectedUsers } from "@/components/ConnectedUsers"
import { ExtraControls } from "@/components/ExtraControls"
import { Header } from "@/components/Header"
import { LyricsPanel } from "@/components/LyricsPanel"
import { NowPlaying } from "@/components/NowPlaying"
import { PlayerControls } from "@/components/PlayerControls"
import { Separator } from "@/components/ui/separator"
import { PlaylistsPage } from "@/pages/PlaylistsPage"
import { QueuePage } from "@/pages/QueuePage"
import { useJukebox } from "@/hooks/useJukebox"
import { useMediaSession } from "@/hooks/useMediaSession"
import type { Session } from "@/api/types"

interface RemotePageProps {
  session: Session
  onLogout: () => void
}

/** App shell: sticky header, the active tab's content, and a bottom nav (player / songs / playlists). */
export function RemotePage({ session, onLogout }: RemotePageProps) {
  const { snapshot, users, skipVotes, skipNeeded, sleepAtMs } = useJukebox(true)
  const [tab, setTab] = useState<Tab>("play")
  const current = snapshot.tracks[snapshot.currentIndex] ?? null
  useMediaSession(snapshot)

  return (
    <div className="mx-auto flex min-h-dvh max-w-md flex-col gap-4 px-4 pb-24 pt-1">
      <div className="sticky top-0 z-10 -mx-4 bg-background/85 px-4 backdrop-blur">
        <Header session={session} onLogout={onLogout} />
        <Separator />
      </div>

      {tab === "play" && (
        <>
          <NowPlaying snapshot={snapshot} />
          <PlayerControls snapshot={snapshot} />
          <ExtraControls
            snapshot={snapshot}
            skipVotes={skipVotes}
            skipNeeded={skipNeeded}
            sleepAtMs={sleepAtMs}
          />
          <AudioControls snapshot={snapshot} />
          <LyricsPanel videoId={current?.videoId ?? null} positionSeconds={snapshot.positionSeconds} />
          {session.role === "ADMIN" && <AdminPanel snapshot={snapshot} />}
          <ConnectedUsers users={users} me={session.username} />
        </>
      )}

      {tab === "queue" && <QueuePage snapshot={snapshot} session={session} />}

      {tab === "playlists" && <PlaylistsPage />}

      <BottomNav tab={tab} onChange={setTab} queueCount={snapshot.tracks.length} />
    </div>
  )
}
