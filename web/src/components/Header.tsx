import { LogOut, Music4, Palette } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ShareButton } from "@/components/ShareButton"
import { useAccent } from "@/hooks/useAccent"
import type { Session } from "@/api/types"

interface HeaderProps {
  session: Session
  onLogout: () => void
}

export function Header({ session, onLogout }: HeaderProps) {
  const { cycle } = useAccent()
  return (
    <header className="flex items-center justify-between gap-3 py-2">
      <div className="flex items-center gap-2.5">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/15 text-primary">
          <Music4 className="size-5" />
        </div>
        <div className="leading-tight">
          <h1 className="text-base font-semibold">LetzPlay Musix</h1>
          <p className="text-xs text-muted-foreground">{session.username}</p>
        </div>
      </div>
      <div className="flex items-center gap-1">
        <Badge variant={session.role === "ADMIN" ? "default" : "secondary"} className="mr-1">
          {session.role === "ADMIN" ? "Admin" : "Guest"}
        </Badge>
        <ShareButton />
        <Button variant="ghost" size="icon" onClick={cycle} aria-label="Change accent color">
          <Palette className="size-4" />
        </Button>
        <Button variant="ghost" size="icon" onClick={onLogout} aria-label="Log out">
          <LogOut className="size-4" />
        </Button>
      </div>
    </header>
  )
}
