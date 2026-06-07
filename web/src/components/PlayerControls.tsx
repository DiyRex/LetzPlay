import { useEffect, useState } from "react"
import { Pause, Play, Repeat, Repeat1, Shuffle, SkipBack, SkipForward, Volume2 } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Slider } from "@/components/ui/slider"
import { cn } from "@/lib/utils"
import { api } from "@/api/client"
import type { JukeboxSnapshot, RepeatMode } from "@/api/types"

interface PlayerControlsProps {
  snapshot: JukeboxSnapshot
}

const NEXT_REPEAT: Record<RepeatMode, RepeatMode> = { OFF: "ALL", ALL: "ONE", ONE: "OFF" }

/** Transport, shuffle/repeat, and volume — available to every connected user. */
export function PlayerControls({ snapshot }: PlayerControlsProps) {
  const isPlaying = snapshot.status === "PLAYING" || snapshot.status === "BUFFERING"

  // Local volume so live snapshots (which arrive ~1/s) don't fight the slider mid-drag.
  const [volume, setVolume] = useState(snapshot.volume)
  const [dragging, setDragging] = useState(false)
  useEffect(() => {
    if (!dragging) setVolume(snapshot.volume)
  }, [snapshot.volume, dragging])

  const guard = (fn: () => Promise<unknown>) => async () => {
    try {
      await fn()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed")
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-4 p-4">
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            className={cn(snapshot.shuffle && "text-primary")}
            onClick={guard(() => api.setShuffle(!snapshot.shuffle))}
            aria-label="Shuffle"
            aria-pressed={snapshot.shuffle}
          >
            <Shuffle className="size-5" />
          </Button>

          <Button variant="secondary" size="icon" onClick={guard(api.previous)} aria-label="Previous">
            <SkipBack className="size-5" />
          </Button>
          <Button
            size="icon"
            className="h-12 w-12"
            onClick={guard(() => (isPlaying ? api.pause() : api.play()))}
            aria-label={isPlaying ? "Pause" : "Play"}
          >
            {isPlaying ? <Pause className="size-6" /> : <Play className="size-6" />}
          </Button>
          <Button variant="secondary" size="icon" onClick={guard(api.skip)} aria-label="Next">
            <SkipForward className="size-5" />
          </Button>

          <Button
            variant="ghost"
            size="icon"
            className={cn(snapshot.repeat !== "OFF" && "text-primary")}
            onClick={guard(() => api.setRepeat(NEXT_REPEAT[snapshot.repeat]))}
            aria-label={`Repeat: ${snapshot.repeat.toLowerCase()}`}
          >
            {snapshot.repeat === "ONE" ? <Repeat1 className="size-5" /> : <Repeat className="size-5" />}
          </Button>
        </div>

        <div className="flex items-center gap-3">
          <Volume2 className="size-4 shrink-0 text-muted-foreground" />
          <Slider
            value={[volume]}
            min={0}
            max={100}
            step={1}
            onValueChange={([v]) => {
              setDragging(true)
              setVolume(v)
            }}
            onValueCommit={([v]) => {
              setDragging(false)
              void guard(() => api.setVolume(v))()
            }}
            aria-label="Volume"
          />
          <span className="w-8 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
            {volume}
          </span>
        </div>
      </CardContent>
    </Card>
  )
}
