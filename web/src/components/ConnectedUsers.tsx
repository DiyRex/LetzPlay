import { Users } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import type { ConnectedUser } from "@/api/types"

interface ConnectedUsersProps {
  users: ConnectedUser[]
  me: string
}

/** "Who's here" — the live list of connected remotes, with the current user highlighted. */
export function ConnectedUsers({ users, me }: ConnectedUsersProps) {
  if (users.length === 0) return null

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-1.5 px-1 text-sm font-semibold text-muted-foreground">
        <Users className="size-4" />
        <span>Connected</span>
        <span className="text-xs font-normal">({users.length})</span>
      </div>
      <div className="flex flex-wrap gap-2">
        {users.map((user) => (
          <Badge
            key={user.username}
            variant={user.role === "ADMIN" ? "default" : "secondary"}
            className="gap-1"
          >
            {user.username}
            {user.username === me && " (you)"}
          </Badge>
        ))}
      </div>
    </div>
  )
}
