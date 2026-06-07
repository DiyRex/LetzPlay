import { useEffect, useRef, useState } from "react"
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
  const seekable = !!song && duration > 0

  // Three position sources, in priority order:
  //  - `scrub`:   live value while the user drags (follows the finger exactly)
  //  - `pending`: value just released; held until the server's reported position catches up, so the
  //               thumb doesn't snap back to the old spot during the ~1s seek round-trip
  //  - otherwise: the live server position
  const [scrub, setScrub] = useState<number | null>(null)
  const [pending, setPending] = useState<number | null>(null)
  const pendingTimer = useRef<ReturnType<typeof setTimeout>>()

  const position = scrub ?? pending ?? snapshot.positionSeconds

  // Release the held value once the server reports a position near where we seeked.
  useEffect(() => {
    if (pending !== null && Math.abs(snapshot.positionSeconds - pending) < 2.5) {
      setPending(null)
    }
  }, [snapshot.positionSeconds, pending])

  const commitSeek = (value: number) => {
    setScrub(null)
    setPending(value)
    clearTimeout(pendingTimer.current)
    pendingTimer.current = setTimeout(() => setPending(null), 4000) // safety release
    void api.seek(value).catch(() => undefined)
  }

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

        <div className="flex flex-col gap-1.5 pt-1">
          <Slider
            value={[Math.min(Math.max(position, 0), duration || 1)]}
            min={0}
            max={Math.max(duration, 1)}
            step={1}
            disabled={!seekable}
            onValueChange={([v]) => setScrub(v)}
            onValueCommit={([v]) => (seekable ? commitSeek(v) : setScrub(null))}
            aria-label="Seek"
          />
          <div className="flex justify-between text-xs tabular-nums text-muted-foreground">
            <span className={scrub !== null ? "font-semibold text-primary" : undefined}>
              {formatTime(position)}
            </span>
            <span>{formatTime(duration)}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
