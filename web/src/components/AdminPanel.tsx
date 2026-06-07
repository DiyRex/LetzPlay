import { useState } from "react"
import { Lock, LockOpen, Shield } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api } from "@/api/client"
import type { JukeboxSnapshot } from "@/api/types"

interface AdminPanelProps {
  snapshot: JukeboxSnapshot
}

/** Admin-only controls: lock the queue (guests can't add) and change the party/admin passwords. */
export function AdminPanel({ snapshot }: AdminPanelProps) {
  const [open, setOpen] = useState(false)
  const [admin, setAdmin] = useState("")
  const [guest, setGuest] = useState("")

  const guard = (fn: () => Promise<unknown>, ok?: string) => async () => {
    try {
      await fn()
      if (ok) toast.success(ok)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed")
    }
  }

  const savePasswords = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!admin.trim() && !guest.trim()) return
    await guard(() => api.setPasswords(admin.trim(), guest.trim()), "Passwords updated")()
    setAdmin("")
    setGuest("")
  }

  return (
    <Card className="border-primary/30">
      <CardContent className="flex flex-col gap-3 p-4">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex items-center gap-2 text-left text-sm font-semibold"
        >
          <Shield className="size-4 text-primary" />
          Admin
        </button>

        <Button
          variant={snapshot.locked ? "default" : "outline"}
          size="sm"
          onClick={guard(() => api.setLock(!snapshot.locked))}
        >
          {snapshot.locked ? <Lock className="size-4" /> : <LockOpen className="size-4" />}
          {snapshot.locked ? "Queue locked (only admins can add)" : "Lock queue"}
        </Button>

        {open && (
          <form onSubmit={savePasswords} className="flex flex-col gap-2 border-t border-border pt-3">
            <div className="space-y-1">
              <Label htmlFor="adminPw">New admin password</Label>
              <Input
                id="adminPw"
                type="password"
                value={admin}
                onChange={(e) => setAdmin(e.target.value)}
                placeholder="leave blank to keep"
                autoComplete="new-password"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="guestPw">New party (guest) password</Label>
              <Input
                id="guestPw"
                type="password"
                value={guest}
                onChange={(e) => setGuest(e.target.value)}
                placeholder="leave blank to keep"
                autoComplete="new-password"
              />
            </div>
            <Button type="submit" size="sm" disabled={!admin.trim() && !guest.trim()}>
              Update passwords
            </Button>
            <p className="text-xs text-muted-foreground">
              Changes apply immediately. On desktop they reset when the server restarts (set them in
              the .env to persist).
            </p>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
