import { useState } from "react"
import { Loader2, Music2, Plus, Search } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/api/client"
import type { SearchResult } from "@/api/types"

const LOOKS_LIKE_LINK = /youtu\.?be|youtube\.com|^https?:\/\//i
const BARE_ID = /^[A-Za-z0-9_-]{11}$/

/**
 * One box for both: paste a YouTube link/playlist (queues it directly), or type words to search
 * (results appear below; tap one to queue it). Keeps the remote to a single, obvious input.
 */
export function AddSong() {
  const [text, setText] = useState("")
  const [busy, setBusy] = useState(false)
  const [results, setResults] = useState<SearchResult[]>([])
  const [searched, setSearched] = useState(false)

  const isLink = (s: string) => LOOKS_LIKE_LINK.test(s) || BARE_ID.test(s)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    const q = text.trim()
    if (!q || busy) return
    setBusy(true)
    try {
      if (isLink(q)) {
        const res = await api.addSong(q)
        toast.success(res.added > 1 ? `Added ${res.added} songs` : `Added "${res.song.title}"`)
        setText("")
        setResults([])
        setSearched(false)
      } else {
        const found = await api.search(q)
        setResults(found)
        setSearched(true)
        if (found.length === 0) toast.message("No results — try different words, or paste a link")
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Something went wrong")
    } finally {
      setBusy(false)
    }
  }

  const queueResult = async (r: SearchResult) => {
    try {
      await api.addSong(r.videoId)
      toast.success(`Added "${r.title}"`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not add")
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <form onSubmit={submit} className="flex gap-2">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Search or paste a YouTube link/playlist…"
            inputMode="search"
            autoComplete="off"
            aria-label="Search or paste a link"
            className="pl-9"
          />
        </div>
        <Button type="submit" disabled={busy || !text.trim()} className="shrink-0">
          {busy ? <Loader2 className="size-4 animate-spin" /> : isLink(text) ? <Plus className="size-4" /> : <Search className="size-4" />}
          {isLink(text) ? "Add" : "Search"}
        </Button>
      </form>

      {searched && results.length > 0 && (
        <ul className="flex max-h-80 flex-col gap-1 overflow-y-auto rounded-lg border bg-card p-1">
          {results.map((r) => (
            <li key={r.videoId}>
              <button
                type="button"
                onClick={() => queueResult(r)}
                className="flex w-full items-center gap-3 rounded-md p-1.5 text-left hover:bg-secondary"
              >
                <span className="size-10 shrink-0 overflow-hidden rounded bg-secondary">
                  {r.thumbnailUrl ? (
                    <img src={r.thumbnailUrl} alt="" className="size-full object-cover" />
                  ) : (
                    <span className="flex size-full items-center justify-center text-muted-foreground">
                      <Music2 className="size-4" />
                    </span>
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">{r.title}</span>
                  {r.channel && <span className="block truncate text-xs text-muted-foreground">{r.channel}</span>}
                </span>
                <Plus className="size-4 shrink-0 text-muted-foreground" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
