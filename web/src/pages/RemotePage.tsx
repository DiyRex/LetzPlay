import { AddSong } from "@/components/AddSong"
import { AdminControls } from "@/components/AdminControls"
import { ConnectedUsers } from "@/components/ConnectedUsers"
import { Header } from "@/components/Header"
import { NowPlaying } from "@/components/NowPlaying"
import { QueueList } from "@/components/QueueList"
import { Separator } from "@/components/ui/separator"
import { useJukebox } from "@/hooks/useJukebox"
import type { Session } from "@/api/types"

interface RemotePageProps {
  session: Session
  onLogout: () => void
}

/** The main remote: live now-playing, transport (admins), add-a-song, presence, and the queue. */
export function RemotePage({ session, onLogout }: RemotePageProps) {
  const { snapshot, users } = useJukebox(true)

  return (
    <div className="mx-auto flex min-h-dvh max-w-md flex-col gap-4 px-4 pb-8">
      <div className="sticky top-0 z-10 -mx-4 bg-background/85 px-4 backdrop-blur">
        <Header session={session} onLogout={onLogout} />
        <Separator />
      </div>

      <NowPlaying snapshot={snapshot} />

      {session.role === "ADMIN" && <AdminControls snapshot={snapshot} />}

      <AddSong />

      <ConnectedUsers users={users} me={session.username} />

      <QueueList snapshot={snapshot} session={session} />
    </div>
  )
}
