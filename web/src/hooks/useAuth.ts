import { useCallback, useEffect, useState } from "react"
import { api } from "@/api/client"
import type { Session } from "@/api/types"

interface AuthState {
  session: Session | null
  /** True until the initial "am I already logged in?" check resolves. */
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

/** Owns the session lifecycle: restores an existing cookie on load, exposes login/logout. */
export function useAuth(): AuthState {
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    api
      .me()
      .then((s) => active && setSession(s))
      .catch(() => active && setSession(null))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    setSession(await api.login(username, password))
  }, [])

  const logout = useCallback(async () => {
    await api.logout()
    setSession(null)
  }, [])

  return { session, loading, login, logout }
}
