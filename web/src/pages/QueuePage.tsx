import { useState } from "react"
import { Save, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { AddSong } from "@/components/AddSong"
import { QueueList } from "@/components/QueueList"
import { StatsCard } from "@/components/StatsCard"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/api/client"
import type { JukeboxSnapshot, Session } from "@/api/types"

interface QueuePageProps {
  snapshot: JukeboxSnapshot
  session: Session
}

/** The full song list with management: add, clear, save-as-playlist, reorder/remove, tap-to-play. */
export function QueuePage({ snapshot, session }: QueuePageProps) {
  const [saving, setSaving] = useState(false)
  const [name, setName] = useState("")

  const clear = async () => {
    if (!confirm("Clear the entire queue?")) return
    try {
      await api.clearQueue()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not clear")
    }
  }

  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    try {
      await api.saveQueueAsPlaylist(trimmed)
      toast.success(`Saved "${trimmed}" playlist`)
      setName("")
      setSaving(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save")
    }
  }

  const existingVideoIds = new Set(snapshot.tracks.map((t) => t.videoId))

  return (
    <div className="flex flex-col gap-4">
      <AddSong existingVideoIds={existingVideoIds} />

      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" className="flex-1" onClick={() => setSaving((v) => !v)}>
          <Save className="size-4" />
          Save as playlist
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="flex-1 text-muted-foreground hover:text-destructive"
          onClick={clear}
          disabled={snapshot.tracks.length === 0}
        >
          <Trash2 className="size-4" />
          Clear
        </Button>
      </div>

      {saving && (
        <form onSubmit={save} className="flex gap-2">
          <Input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Playlist name…"
            aria-label="Playlist name"
          />
          <Button type="submit" disabled={!name.trim()}>
            Save
          </Button>
        </form>
      )}

      <QueueList snapshot={snapshot} session={session} />

      <StatsCard />
    </div>
  )
}
