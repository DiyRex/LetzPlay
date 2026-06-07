import { Gauge, SlidersHorizontal, Volume2 } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"
import { api } from "@/api/client"
import type { JukeboxSnapshot } from "@/api/types"

interface AudioControlsProps {
  snapshot: JukeboxSnapshot
}

const EQ_PRESETS = [
  { key: "flat", label: "Flat" },
  { key: "bass", label: "Bass" },
  { key: "treble", label: "Treble" },
  { key: "vocal", label: "Vocal" },
  { key: "loud", label: "Loud" },
]
const SPEEDS = [0.75, 1, 1.25, 1.5, 2]

/** Sound settings shared across the room: loudness normalization, equalizer preset, and speed. */
export function AudioControls({ snapshot }: AudioControlsProps) {
  const guard = (fn: () => Promise<unknown>) => async () => {
    try {
      await fn()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed")
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3 p-4">
        <div className="flex items-center gap-2">
          <Volume2 className="size-4 shrink-0 text-muted-foreground" />
          <span className="flex-1 text-sm">Normalize volume</span>
          <Button
            variant={snapshot.normalize ? "default" : "outline"}
            size="sm"
            className="h-8"
            onClick={guard(() => api.setNormalize(!snapshot.normalize))}
            aria-pressed={snapshot.normalize}
          >
            {snapshot.normalize ? "On" : "Off"}
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <SlidersHorizontal className="size-4 shrink-0 text-muted-foreground" />
          <div className="flex flex-1 flex-wrap justify-end gap-1.5">
            {EQ_PRESETS.map((p) => (
              <Button
                key={p.key}
                variant={snapshot.eq === p.key ? "default" : "secondary"}
                size="sm"
                className="h-8 px-2.5"
                onClick={guard(() => api.setEq(p.key))}
              >
                {p.label}
              </Button>
            ))}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Gauge className="size-4 shrink-0 text-muted-foreground" />
          <div className="flex flex-1 flex-wrap justify-end gap-1.5">
            {SPEEDS.map((s) => (
              <Button
                key={s}
                variant={Math.abs(snapshot.speed - s) < 0.01 ? "default" : "secondary"}
                size="sm"
                className={cn("h-8 px-2.5 tabular-nums")}
                onClick={guard(() => api.setSpeed(s))}
              >
                {s}×
              </Button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
