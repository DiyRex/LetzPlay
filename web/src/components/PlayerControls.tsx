import { Pause, Play, SkipBack, SkipForward, Volume2 } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Slider } from "@/components/ui/slider"
import { api } from "@/api/client"
import type { JukeboxSnapshot } from "@/api/types"

interface PlayerControlsProps {
  snapshot: JukeboxSnapshot
}

/** Transport + volume, available to every connected user. The server authorizes any session. */
export function PlayerControls({ snapshot }: PlayerControlsProps) {
  const isPlaying = snapshot.status === "PLAYING" || snapshot.status === "BUFFERING"

  const guard = (fn: () => Promise<void>) => async () => {
    try {
      await fn()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed")
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-4 p-4">
        <div className="flex items-center justify-center gap-3">
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
        </div>

        <div className="flex items-center gap-3">
          <Volume2 className="size-4 shrink-0 text-muted-foreground" />
          <Slider
            value={[snapshot.volume]}
            min={0}
            max={100}
            step={1}
            onValueCommit={([v]) => void guard(() => api.setVolume(v))()}
            aria-label="Volume"
          />
          <span className="w-8 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
            {snapshot.volume}
          </span>
        </div>
      </CardContent>
    </Card>
  )
}
