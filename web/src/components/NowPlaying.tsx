import { Music2 } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { formatTime } from "@/lib/format"
import type { JukeboxSnapshot } from "@/api/types"

interface NowPlayingProps {
  snapshot: JukeboxSnapshot
}

export function NowPlaying({ snapshot }: NowPlayingProps) {
  const song = snapshot.nowPlaying
  const progress =
    snapshot.durationSeconds > 0
      ? Math.min(100, (snapshot.positionSeconds / snapshot.durationSeconds) * 100)
      : 0

  return (
    <Card className="overflow-hidden">
      <CardContent className="flex items-center gap-4 p-4">
        <div className="size-16 shrink-0 overflow-hidden rounded-md bg-secondary">
          {song?.thumbnailUrl ? (
            <img src={song.thumbnailUrl} alt="" className="size-full object-cover" />
          ) : (
            <div className="flex size-full items-center justify-center text-muted-foreground">
              <Music2 className="size-6" />
            </div>
          )}
        </div>

        <div className="min-w-0 flex-1">
          <p className="text-xs font-semibold uppercase tracking-wider text-primary">Now Playing</p>
          <p className="truncate text-base font-semibold">{song?.title ?? "Nothing playing"}</p>
          {song && (
            <p className="truncate text-sm text-muted-foreground">added by {song.addedBy}</p>
          )}

          <div className="mt-3 flex items-center gap-2">
            <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-secondary">
              <div
                className="h-full rounded-full bg-primary transition-[width] duration-1000 ease-linear"
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
              {formatTime(snapshot.positionSeconds)} / {formatTime(snapshot.durationSeconds)}
            </span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
