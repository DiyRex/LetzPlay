import { ChevronDown, ChevronUp, ListMusic, Music2, X } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { api } from "@/api/client"
import type { JukeboxSnapshot, Session } from "@/api/types"

interface QueueListProps {
  snapshot: JukeboxSnapshot
  session: Session
}

export function QueueList({ snapshot, session }: QueueListProps) {
  const { queue } = snapshot
  const canManage = (addedBy: string) => session.role === "ADMIN" || addedBy === session.username

  const remove = async (id: string) => {
    try {
      await api.removeSong(id)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not remove")
    }
  }

  const move = async (id: string, to: number) => {
    try {
      await api.reorder(id, to)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not reorder")
    }
  }

  if (queue.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-10 text-center text-muted-foreground">
        <ListMusic className="size-7" />
        <p className="text-sm">Queue is empty — add the first song.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between px-1">
        <h2 className="text-sm font-semibold text-muted-foreground">Up Next</h2>
        <span className="text-xs text-muted-foreground">{queue.length}</span>
      </div>

      <ul className="flex flex-col gap-2">
        {queue.map((song, index) => (
          <li
            key={song.id}
            className="flex items-center gap-3 rounded-lg border bg-card p-2.5"
          >
            <span className="w-5 shrink-0 text-center text-sm tabular-nums text-muted-foreground">
              {index + 1}
            </span>

            <div className="size-11 shrink-0 overflow-hidden rounded bg-secondary">
              {song.thumbnailUrl ? (
                <img src={song.thumbnailUrl} alt="" className="size-full object-cover" />
              ) : (
                <div className="flex size-full items-center justify-center text-muted-foreground">
                  <Music2 className="size-4" />
                </div>
              )}
            </div>

            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">{song.title}</p>
              <p className="truncate text-xs text-muted-foreground">{song.addedBy}</p>
            </div>

            {canManage(song.addedBy) && (
              <div className="flex shrink-0 items-center">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  disabled={index === 0}
                  onClick={() => move(song.id, index - 1)}
                  aria-label="Move up"
                >
                  <ChevronUp className="size-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  disabled={index === queue.length - 1}
                  onClick={() => move(song.id, index + 1)}
                  aria-label="Move down"
                >
                  <ChevronDown className="size-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-muted-foreground hover:text-destructive"
                  onClick={() => remove(song.id)}
                  aria-label="Remove"
                >
                  <X className="size-4" />
                </Button>
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
