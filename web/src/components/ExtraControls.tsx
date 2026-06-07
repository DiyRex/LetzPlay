import { useEffect, useState } from "react"
import { Moon, Radio, SkipForward } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"
import { formatTime } from "@/lib/format"
import { api } from "@/api/client"
import type { JukeboxSnapshot } from "@/api/types"

interface ExtraControlsProps {
  snapshot: JukeboxSnapshot
  skipVotes: number
  skipNeeded: number
  sleepAtMs: number
}

const SLEEP_OPTIONS = [15, 30, 60]

/** Party extras: vote-to-skip, radio (autoplay) toggle, and a sleep timer. */
export function ExtraControls({ snapshot, skipVotes, skipNeeded, sleepAtMs }: ExtraControlsProps) {
  const hasCurrent = snapshot.currentIndex >= 0

  // Local ticker so the sleep countdown updates smoothly without server traffic.
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (sleepAtMs <= 0) return
    const id = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(id)
  }, [sleepAtMs])
  const sleepRemaining = sleepAtMs > 0 ? Math.max(0, Math.round((sleepAtMs - now) / 1000)) : 0

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
          <Button
            variant="outline"
            size="sm"
            className="flex-1"
            disabled={!hasCurrent}
            onClick={guard(api.voteSkip)}
          >
            <SkipForward className="size-4" />
            Vote skip {skipNeeded > 1 ? `(${skipVotes}/${skipNeeded})` : ""}
          </Button>
          <Button
            variant={snapshot.autoplay ? "default" : "outline"}
            size="sm"
            className="flex-1"
            onClick={guard(() => api.setAutoplay(!snapshot.autoplay))}
            aria-pressed={snapshot.autoplay}
          >
            <Radio className="size-4" />
            Radio {snapshot.autoplay ? "on" : "off"}
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Moon className="size-4" />
            <span className="tabular-nums">
              {sleepRemaining > 0 ? formatTime(sleepRemaining) : "Sleep"}
            </span>
          </div>
          <div className="flex flex-1 justify-end gap-1.5">
            {SLEEP_OPTIONS.map((m) => (
              <Button
                key={m}
                variant="secondary"
                size="sm"
                className="h-8 px-2.5"
                onClick={guard(() => api.setSleep(m))}
              >
                {m}m
              </Button>
            ))}
            <Button
              variant="ghost"
              size="sm"
              className={cn("h-8 px-2.5", sleepRemaining === 0 && "text-muted-foreground")}
              disabled={sleepRemaining === 0}
              onClick={guard(() => api.setSleep(0))}
            >
              Off
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
