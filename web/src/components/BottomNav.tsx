import { ListMusic, Music4, Radio } from "lucide-react"
import { cn } from "@/lib/utils"

export type Tab = "play" | "queue" | "playlists"

interface BottomNavProps {
  tab: Tab
  onChange: (tab: Tab) => void
  queueCount: number
}

const ITEMS: { key: Tab; label: string; icon: typeof Radio }[] = [
  { key: "play", label: "Now Playing", icon: Radio },
  { key: "queue", label: "Songs", icon: ListMusic },
  { key: "playlists", label: "Playlists", icon: Music4 },
]

/** Fixed bottom tab bar — the primary navigation between the player, the song list, and playlists. */
export function BottomNav({ tab, onChange, queueCount }: BottomNavProps) {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-20 border-t border-border bg-background/95 backdrop-blur">
      <div
        className="mx-auto flex max-w-md items-stretch justify-around"
        style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      >
        {ITEMS.map(({ key, label, icon: Icon }) => {
          const active = tab === key
          return (
            <button
              key={key}
              type="button"
              onClick={() => onChange(key)}
              className={cn(
                "relative flex flex-1 flex-col items-center gap-0.5 py-2.5 text-xs transition-colors",
                active ? "text-primary" : "text-muted-foreground hover:text-foreground",
              )}
              aria-current={active}
            >
              <Icon className="size-5" />
              <span>{label}</span>
              {key === "queue" && queueCount > 0 && (
                <span className="absolute right-1/2 top-1.5 translate-x-3 rounded-full bg-primary px-1.5 text-[10px] font-semibold leading-4 text-primary-foreground">
                  {queueCount}
                </span>
              )}
            </button>
          )
        })}
      </div>
    </nav>
  )
}
