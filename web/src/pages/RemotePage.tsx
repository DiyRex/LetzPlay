import { useState } from "react"
import { BottomNav, type Tab } from "@/components/BottomNav"
import { ConnectedUsers } from "@/components/ConnectedUsers"
import { Header } from "@/components/Header"
import { NowPlaying } from "@/components/NowPlaying"
import { PlayerControls } from "@/components/PlayerControls"
import { Separator } from "@/components/ui/separator"
import { PlaylistsPage } from "@/pages/PlaylistsPage"
import { QueuePage } from "@/pages/QueuePage"
import { useJukebox } from "@/hooks/useJukebox"
import type { Session } from "@/api/types"

interface RemotePageProps {
  session: Session
  onLogout: () => void
}

/** App shell: a sticky header, the active tab's content, and a bottom nav (player / songs / playlists). */
export function RemotePage({ session, onLogout }: RemotePageProps) {
  const { snapshot, users } = useJukebox(true)
  const [tab, setTab] = useState<Tab>("play")

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
          <ConnectedUsers users={users} me={session.username} />
        </>
      )}

      {tab === "queue" && <QueuePage snapshot={snapshot} session={session} />}

      {tab === "playlists" && <PlaylistsPage />}

      <BottomNav tab={tab} onChange={setTab} queueCount={snapshot.tracks.length} />
    </div>
  )
}
