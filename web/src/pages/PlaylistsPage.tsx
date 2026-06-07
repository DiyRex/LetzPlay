import { useCallback, useEffect, useState } from "react"
import { ChevronDown, ChevronRight, ListPlus, Music2, Play, Plus, Trash2, X } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { api } from "@/api/client"
import type { Playlist, PlaylistSummary } from "@/api/types"

/** Create, browse, and manage saved playlists; load one into the queue. Data is fetched on demand
 * (playlists aren't part of the live websocket state). */
export function PlaylistsPage() {
  const [playlists, setPlaylists] = useState<PlaylistSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [newName, setNewName] = useState("")
  const [openId, setOpenId] = useState<string | null>(null)
  const [openPlaylist, setOpenPlaylist] = useState<Playlist | null>(null)

  const refresh = useCallback(() => {
    api
      .listPlaylists()
      .then(setPlaylists)
      .catch(() => undefined)
      .finally(() => setLoading(false))
  }, [])

  useEffect(refresh, [refresh])

  const fail = (err: unknown, fallback: string) =>
    toast.error(err instanceof Error ? err.message : fallback)

  const create = async (e: React.FormEvent) => {
    e.preventDefault()
    const name = newName.trim()
    if (!name) return
    try {
      await api.createPlaylist(name)
      setNewName("")
      refresh()
    } catch (err) {
      fail(err, "Could not create playlist")
    }
  }

  const open = async (id: string) => {
    if (openId === id) {
      setOpenId(null)
      setOpenPlaylist(null)
      return
    }
    setOpenId(id)
    setOpenPlaylist(null)
    try {
      setOpenPlaylist(await api.getPlaylist(id))
    } catch (err) {
      fail(err, "Could not open playlist")
    }
  }

  const reloadOpen = async (id: string) => {
    try {
      setOpenPlaylist(await api.getPlaylist(id))
    } catch {
      /* ignore */
    }
    refresh()
  }

  const enqueue = async (id: string, name: string) => {
    try {
      const res = await api.enqueuePlaylist(id)
      toast.success(`Added ${res.added} song${res.added === 1 ? "" : "s"} from "${name}"`)
    } catch (err) {
      fail(err, "Could not load playlist")
    }
  }

  const remove = async (id: string) => {
    if (!confirm("Delete this playlist?")) return
    try {
      await api.deletePlaylist(id)
      if (openId === id) {
        setOpenId(null)
        setOpenPlaylist(null)
      }
      refresh()
    } catch (err) {
      fail(err, "Could not delete playlist")
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <form onSubmit={create} className="flex gap-2">
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="New playlist name…"
          aria-label="New playlist name"
        />
        <Button type="submit" disabled={!newName.trim()} className="shrink-0">
          <Plus className="size-4" />
          Create
        </Button>
      </form>

      {loading ? (
        <p className="py-8 text-center text-sm text-muted-foreground">Loading…</p>
      ) : playlists.length === 0 ? (
        <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-10 text-center text-muted-foreground">
          <ListPlus className="size-7" />
          <p className="text-sm">No playlists yet — create one, or save the queue from the Songs tab.</p>
        </div>
      ) : (
        <ul className="flex flex-col gap-2">
          {playlists.map((p) => (
            <li key={p.id} className="rounded-lg border bg-card">
              <div className="flex items-center gap-2 p-2.5">
                <button
                  type="button"
                  onClick={() => open(p.id)}
                  className="flex min-w-0 flex-1 items-center gap-2 text-left"
                >
                  {openId === p.id ? (
                    <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
                  )}
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{p.name}</span>
                    <span className="block text-xs text-muted-foreground">
                      {p.count} song{p.count === 1 ? "" : "s"}
                    </span>
                  </span>
                </button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => enqueue(p.id, p.name)}
                  disabled={p.count === 0}
                >
                  <Play className="size-4" />
                  Play
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-muted-foreground hover:text-destructive"
                  onClick={() => remove(p.id)}
                  aria-label="Delete playlist"
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>

              {openId === p.id && (
                <PlaylistDetail
                  playlist={openPlaylist}
                  onChanged={() => reloadOpen(p.id)}
                  onFail={fail}
                />
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function PlaylistDetail({
  playlist,
  onChanged,
  onFail,
}: {
  playlist: Playlist | null
  onChanged: () => void
  onFail: (err: unknown, fallback: string) => void
}) {
  const [url, setUrl] = useState("")

  if (!playlist) {
    return <p className="px-3 pb-3 text-xs text-muted-foreground">Loading…</p>
  }

  const addSong = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = url.trim()
    if (!trimmed) return
    try {
      await api.addPlaylistSong(playlist.id, trimmed)
      setUrl("")
      onChanged()
    } catch (err) {
      onFail(err, "Could not add song")
    }
  }

  const removeSong = async (videoId: string) => {
    try {
      await api.removePlaylistSong(playlist.id, videoId)
      onChanged()
    } catch (err) {
      onFail(err, "Could not remove song")
    }
  }

  return (
    <Card className="m-2 mt-0 border-dashed bg-background">
      <CardContent className="flex flex-col gap-2 p-2.5">
        <form onSubmit={addSong} className="flex gap-2">
          <Input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="Add a YouTube link…"
            inputMode="url"
            aria-label="Add song to playlist"
          />
          <Button type="submit" size="sm" disabled={!url.trim()}>
            <Plus className="size-4" />
          </Button>
        </form>

        {playlist.songs.length === 0 ? (
          <p className="py-2 text-center text-xs text-muted-foreground">Empty — add songs above.</p>
        ) : (
          <ul className="flex flex-col">
            {playlist.songs.map((song) => (
              <li key={song.videoId} className="flex items-center gap-2 py-1.5">
                <span className="size-9 shrink-0 overflow-hidden rounded bg-secondary">
                  {song.thumbnailUrl ? (
                    <img src={song.thumbnailUrl} alt="" className="size-full object-cover" />
                  ) : (
                    <span className="flex size-full items-center justify-center text-muted-foreground">
                      <Music2 className="size-4" />
                    </span>
                  )}
                </span>
                <span className="min-w-0 flex-1 truncate text-sm">{song.title}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-muted-foreground hover:text-destructive"
                  onClick={() => removeSong(song.videoId)}
                  aria-label="Remove from playlist"
                >
                  <X className="size-4" />
                </Button>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
