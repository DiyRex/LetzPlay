import { useEffect, useState } from "react"
import { BarChart3, ChevronDown, ChevronUp } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { api } from "@/api/client"
import type { Stats } from "@/api/types"

/** Collapsible session stats: most-played songs and top requesters. */
export function StatsCard() {
  const [open, setOpen] = useState(false)
  const [stats, setStats] = useState<Stats | null>(null)

  useEffect(() => {
    if (!open) return
    let active = true
    api.stats().then((s) => active && setStats(s)).catch(() => undefined)
    return () => {
      active = false
    }
  }, [open])

  const list = (title: string, rows: Stats["mostPlayed"]) => (
    <div className="flex flex-col gap-1">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</p>
      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground">Nothing yet.</p>
      ) : (
        rows.map((r, i) => (
          <div key={i} className="flex items-center gap-2 text-sm">
            <span className="w-4 shrink-0 tabular-nums text-muted-foreground">{i + 1}</span>
            <span className="min-w-0 flex-1 truncate">{r.label}</span>
            <span className="shrink-0 tabular-nums text-muted-foreground">{r.count}</span>
          </div>
        ))
      )}
    </div>
  )

  return (
    <Card>
      <CardContent className="p-0">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex w-full items-center gap-2 p-3 text-left text-sm font-medium"
        >
          <BarChart3 className="size-4 text-primary" />
          <span className="flex-1">Party stats</span>
          {open ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
        </button>
        {open && (
          <div className="flex flex-col gap-4 px-4 pb-4">
            {list("Most played", stats?.mostPlayed ?? [])}
            {list("Top requesters", stats?.topRequesters ?? [])}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
