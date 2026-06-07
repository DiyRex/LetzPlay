import { useState } from "react"
import { Music2 } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Slider } from "@/components/ui/slider"
import { formatTime } from "@/lib/format"
import { api } from "@/api/client"
import type { JukeboxSnapshot } from "@/api/types"

interface NowPlayingProps {
  snapshot: JukeboxSnapshot
}

export function NowPlaying({ snapshot }: NowPlayingProps) {
  const song = snapshot.tracks[snapshot.currentIndex] ?? null
  const duration = snapshot.durationSeconds

  // While the user drags the seek bar, hold a local value so incoming progress frames don't yank
  // the thumb back. Released on commit (seek) — then live updates resume.
  const [scrubbing, setScrubbing] = useState<number | null>(null)
  const position = scrubbing ?? snapshot.positionSeconds

  return (
    <Card className="overflow-hidden">
      <CardContent className="flex flex-col gap-3 p-4">
        <div className="flex items-center gap-4">
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
            {song && <p className="truncate text-sm text-muted-foreground">added by {song.addedBy}</p>}
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <Slider
            value={[Math.min(position, duration || 0)]}
            min={0}
            max={Math.max(duration, 1)}
            step={1}
            disabled={!song || duration <= 0}
            onValueChange={([v]) => setScrubbing(v)}
            onValueCommit={([v]) => {
              setScrubbing(null)
              if (song && duration > 0) void api.seek(v).catch(() => undefined)
            }}
            aria-label="Seek"
          />
          <div className="flex justify-between text-xs tabular-nums text-muted-foreground">
            <span>{formatTime(position)}</span>
            <span>{formatTime(duration)}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
