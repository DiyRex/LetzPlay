import { ChevronDown, ChevronUp, ListMusic, Music2, Play, Radio, Volume2, X } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { api } from "@/api/client"
import type { JukeboxSnapshot, Session } from "@/api/types"

interface QueueListProps {
  snapshot: JukeboxSnapshot
  session: Session
}

/**
 * The full song list (played, current, upcoming). The current track is highlighted, already-played
 * ones are dimmed, and tapping any row jumps playback to it. Songs are never auto-removed — only
 * via the explicit remove button (admins, or the person who added it).
 */
export function QueueList({ snapshot, session }: QueueListProps) {
  const { tracks, currentIndex } = snapshot
  const canManage = (addedBy: string) => session.role === "ADMIN" || addedBy === session.username

  const act = (fn: () => Promise<unknown>, failMsg: string) => async () => {
    try {
      await fn()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : failMsg)
    }
  }

  if (tracks.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-10 text-center text-muted-foreground">
        <ListMusic className="size-7" />
        <p className="text-sm">No songs yet — paste a link above to start the party.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between px-1">
        <h2 className="text-sm font-semibold text-muted-foreground">Queue</h2>
        <span className="text-xs text-muted-foreground">{tracks.length}</span>
      </div>

      <ul className="flex flex-col gap-2">
        {tracks.map((song, index) => {
          const isCurrent = index === currentIndex
          const isPlayed = index < currentIndex
          return (
            <li
              key={song.id}
              className={cn(
                "flex items-center gap-2 rounded-lg border bg-card p-2 transition-colors",
                isCurrent && "border-primary/60 bg-primary/10",
                isPlayed && "opacity-55",
              )}
            >
              {/* Tappable area → jump playback to this song */}
              <button
                type="button"
                onClick={act(() => api.playNow(song.id), "Could not play that")}
                className="flex min-w-0 flex-1 items-center gap-3 text-left"
                aria-label={`Play ${song.title}`}
              >
                <span className="flex w-5 shrink-0 justify-center text-sm tabular-nums text-muted-foreground">
                  {isCurrent ? <Volume2 className="size-4 text-primary" /> : index + 1}
                </span>
                <span className="relative size-11 shrink-0 overflow-hidden rounded bg-secondary">
                  {song.thumbnailUrl ? (
                    <img src={song.thumbnailUrl} alt="" className="size-full object-cover" />
                  ) : (
                    <span className="flex size-full items-center justify-center text-muted-foreground">
                      <Music2 className="size-4" />
                    </span>
                  )}
                  {!isCurrent && (
                    <span className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition-opacity hover:opacity-100">
                      <Play className="size-4 text-white" />
                    </span>
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className={cn("block truncate text-sm font-medium", isCurrent && "text-primary")}>
                    {song.title}
                  </span>
                  <span className="block truncate text-xs text-muted-foreground">{song.addedBy}</span>
                </span>
              </button>

              <div className="flex shrink-0 items-center">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-muted-foreground hover:text-primary"
                  onClick={act(() => api.radioFromSong(song.id), "Could not start radio")}
                  aria-label="Start radio from this song"
                  title="Start radio from this song"
                >
                  <Radio className="size-4" />
                </Button>
              </div>

              {canManage(song.addedBy) && (
                <div className="flex shrink-0 items-center">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    disabled={index === 0}
                    onClick={act(() => api.reorder(song.id, index - 1), "Could not reorder")}
                    aria-label="Move up"
                  >
                    <ChevronUp className="size-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    disabled={index === tracks.length - 1}
                    onClick={act(() => api.reorder(song.id, index + 1), "Could not reorder")}
                    aria-label="Move down"
                  >
                    <ChevronDown className="size-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-muted-foreground hover:text-destructive"
                    onClick={act(() => api.removeSong(song.id), "Could not remove")}
                    aria-label="Remove"
                  >
                    <X className="size-4" />
                  </Button>
                </div>
              )}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
