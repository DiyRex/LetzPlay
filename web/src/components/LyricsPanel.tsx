import { useEffect, useRef, useState } from "react"
import { ChevronDown, ChevronUp, Mic2 } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"
import { api } from "@/api/client"
import type { Lyrics } from "@/api/types"

interface LyricsPanelProps {
  videoId: string | null
  positionSeconds: number
}

/** Collapsible lyrics for the current track. Shows time-synced lines (auto-scrolling, current line
 * highlighted) when available, else plain text. Fetched from lrclib via the server. */
export function LyricsPanel({ videoId, positionSeconds }: LyricsPanelProps) {
  const [open, setOpen] = useState(false)
  const [lyrics, setLyrics] = useState<Lyrics | null>(null)
  const [loading, setLoading] = useState(false)
  const activeRef = useRef<HTMLParagraphElement | null>(null)

  useEffect(() => {
    if (!videoId || !open) return
    let active = true
    setLoading(true)
    setLyrics(null)
    api
      .lyrics(videoId)
      .then((l) => active && setLyrics(l))
      .catch(() => active && setLyrics({ found: false, synced: [], plain: "" }))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [videoId, open])

  const posMs = positionSeconds * 1000
  const synced = lyrics?.synced ?? []
  let activeIndex = -1
  for (let i = 0; i < synced.length; i++) {
    if (synced[i].timeMs <= posMs) activeIndex = i
    else break
  }

  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: "center", behavior: "smooth" })
  }, [activeIndex])

  return (
    <Card>
      <CardContent className="p-0">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex w-full items-center gap-2 p-3 text-left text-sm font-medium"
        >
          <Mic2 className="size-4 text-primary" />
          <span className="flex-1">Lyrics</span>
          {open ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
        </button>

        {open && (
          <div className="max-h-72 overflow-y-auto px-4 pb-4">
            {loading && <p className="py-4 text-center text-sm text-muted-foreground">Looking up lyrics…</p>}
            {!loading && !lyrics?.found && (
              <p className="py-4 text-center text-sm text-muted-foreground">No lyrics found for this track.</p>
            )}
            {!loading && lyrics?.found && synced.length > 0 && (
              <div className="flex flex-col gap-2 py-1 text-center">
                {synced.map((line, i) => (
                  <p
                    key={i}
                    ref={i === activeIndex ? activeRef : null}
                    className={cn(
                      "text-sm transition-colors",
                      i === activeIndex ? "font-semibold text-primary" : "text-muted-foreground",
                    )}
                  >
                    {line.text || "♪"}
                  </p>
                ))}
              </div>
            )}
            {!loading && lyrics?.found && synced.length === 0 && lyrics.plain && (
              <pre className="whitespace-pre-wrap py-1 text-center font-sans text-sm text-muted-foreground">
                {lyrics.plain}
              </pre>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
