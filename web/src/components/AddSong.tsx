import { useState } from "react"
import { Plus } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/api/client"

/** Paste-a-link form. Self-contained: owns its input state and reports outcomes via toast. */
export function AddSong() {
  const [url, setUrl] = useState("")
  const [submitting, setSubmitting] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = url.trim()
    if (!trimmed || submitting) return
    setSubmitting(true)
    try {
      const result = await api.addSong(trimmed)
      toast.success(
        result.added > 1 ? `Added ${result.added} songs to the queue` : `Added "${result.song.title}"`,
      )
      setUrl("")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not add song")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={submit} className="flex gap-2">
      <Input
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        placeholder="Paste a YouTube link or playlist…"
        inputMode="url"
        autoComplete="off"
        aria-label="YouTube link"
      />
      <Button type="submit" disabled={submitting || !url.trim()} className="shrink-0">
        <Plus className="size-4" />
        Add
      </Button>
    </form>
  )
}
