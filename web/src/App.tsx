import { Loader2 } from "lucide-react"
import { Toaster } from "sonner"
import { LoginPage } from "@/pages/LoginPage"
import { RemotePage } from "@/pages/RemotePage"
import { useAuth } from "@/hooks/useAuth"

export default function App() {
  const { session, loading, login, logout } = useAuth()

  return (
    <>
      {loading ? (
        <div className="flex min-h-dvh items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : session ? (
        <RemotePage session={session} onLogout={logout} />
      ) : (
        <LoginPage onLogin={login} />
      )}
      <Toaster theme="dark" position="top-center" richColors />
    </>
  )
}
