# CLAUDE.md — Architecture & Conventions

Guidance for anyone (human or AI) working in this repo. Read this before making changes.

## What this is

A self-hosted YouTube jukebox with **two interchangeable servers** and **one shared web remote**:

- `web/` — React + shadcn/ui + Tailwind remote (Vite). The single UI for phones.
- `android/` — Android TV app (Kotlin, Ktor embedded server, android-youtube-player).
- `desktop/` — Go binary (Linux/macOS): mpv playback + Bubble Tea TUI + embedded server.

Phones never know which backend they're talking to: both expose the **same wire protocol**.

## The golden rule: the wire protocol is a contract

Three files describe the **same** JSON. Change one, change all three:

| Concept | TypeScript (`web`) | Kotlin (`android`) | Go (`desktop`) |
|---|---|---|---|
| Song / Snapshot | `src/api/types.ts` | `domain/model/*.kt` | `internal/domain/model.go` |
| Live WS payload | `LiveState {snapshot, users}` | `dto/Dtos.kt` `LiveState` | `internal/api/dto.go` `LiveState` |
| REST DTOs | `src/api/client.ts` | `dto/Dtos.kt` | `internal/api/dto.go` |

Endpoints (identical on both servers):

```
POST   /api/auth/login     {username, password} -> {username, role}   (sets session cookie)
POST   /api/auth/logout
GET    /api/auth/me        -> {username, role}
GET    /api/queue          -> JukeboxSnapshot
POST   /api/queue          {url} -> Song            (any logged-in user)
DELETE /api/queue/{id}                              (admin, or the user who added it)
POST   /api/queue/reorder  {songId, targetIndex}    (admin, or owner)
POST   /api/queue/{id}/play                          (tap-to-play; jump cursor; any user)
POST   /api/player/play|pause|skip|previous          (any logged-in user)
POST   /api/player/seek    {seconds}                 (any user)
POST   /api/player/volume  {volume}                  (any user)
POST   /api/player/shuffle {shuffle}                 (any user)
POST   /api/player/repeat  {repeat: OFF|ALL|ONE}     (any user)
POST   /api/player/clear                             (any user)
POST   /api/player/voteskip                          (majority of connected users -> skip)
POST   /api/player/sleep   {minutes}                 (0 cancels; server auto-pauses)
POST   /api/player/autoplay {autoplay}               (radio: auto-add related when empty; desktop)
GET    /api/search?q=                                -> [SearchResult]  (yt-dlp; empty on Android)
GET    /api/lyrics?videoId=                          -> {found,synced[],plain}  (lrclib, keyless)
POST   /api/admin/lock     {locked}                  (admin; guests can't add while locked)
POST   /api/admin/password {admin?,guest?}           (admin; runtime, in-memory)
GET    /api/playlists                                -> [{id,name,count}]
POST   /api/playlists      {name}                    -> Playlist
POST   /api/playlists/save-queue {name}              -> Playlist (snapshot of the queue)
GET    /api/playlists/{id}                           -> Playlist (with songs)
DELETE /api/playlists/{id}
POST   /api/playlists/{id}/songs {url}               -> Playlist
DELETE /api/playlists/{id}/songs/{videoId}
POST   /api/playlists/{id}/enqueue                   -> AddResult (load into queue)
GET    /ws                 -> pushes LiveState on every change   (auth required)
```

Model note: the queue is a **persistent playlist with a cursor** — `Snapshot{tracks[], currentIndex, status, position, duration, volume, shuffle, repeat}`. Songs are never auto-removed; advancing/previous/jump move the cursor. Repeat-one is realized by the player looping the current file (Go: mpv `loop-file`; Android: re-load on ENDED) so the queue doesn't advance. Playlists persist to a JSON file on desktop (`playlist.Store`), in-memory on Android.

WebSocket pushes `LiveState = { snapshot, users }`. REST `/api/queue` returns the bare snapshot
(the socket fills in presence). Keep these consistent across backends.

## Architecture (same shape in both servers)

```
Queue (pure state machine)  ──observes──▶  Coordinator  ──▶  Player (interface)
   ▲                                                              │
   │ mutations                                       status/progress/onEnded
   │                                                              ▼
Routes (auth + authz)                              WS hub/broadcaster (+ presence)
```

- **Queue** (`MusicQueue.kt` / `domain.Queue`) is the single source of truth. It is *pure*: no
  HTTP, no player, no OS calls. All mutations go through one lock and publish an immutable
  snapshot. This is the most important class — keep it that way. It is unit-tested on both sides.
- **Player** is an interface (`PlaybackController` / `domain.Player`). Concrete players
  (`YouTubePlayerController`, `player.Mpv`) are the *only* code that touches the playback SDK.
- **Coordinator** is the *only* place that decides "what should be playing" — it watches the
  now-playing track and tells the player to load it. Queue ↔ player stay decoupled.
- **Presence** lives in the WS layer (hub/broadcaster), not the queue — it's a transport concern.
- **Auth**: there are no per-user accounts. The *password* selects the role (admin vs guest). The
  *username* is a display identity entered at login, shown on each queued song and in presence.

## Conventions (avoid spaghetti)

- **No god objects.** Activities/`main` wire dependencies and render; they don't hold business logic.
- **Dependency injection by construction.** Pass collaborators in (Android: `ServiceLocator`;
  Go: explicit `New*` constructors). Nothing reaches for globals.
- **Keep the SDK at the edge.** Player libraries, WebView, mpv IPC — confined to one file each.
- **Immutable snapshots, single writer.** Don't mutate shared state from multiple places; route
  every change through the Queue.
- **Authorization in the route layer**, mechanics in the domain. The Queue never checks roles.
- **Route registration order matters:** API + `/ws` first, the SPA catch-all last.
- Match the surrounding style; comments explain *why*, not *what*.

## Where credentials live

- **Desktop:** configured via `desktop/.env` or env vars (`LETZPLAY_PORT`,
  `LETZPLAY_ADMIN_PASSWORD`, `LETZPLAY_GUEST_PASSWORD`, `LETZPLAY_OPEN`, `LETZPLAY_HEADLESS`) loaded
  in `internal/config`, with flags overriding (precedence: flag > env > .env > default; default
  port 8090). Passwords are bcrypt-hashed at startup (`internal/auth`); session cookies are
  HMAC-signed with a per-launch secret. `.env` is git-ignored; `.env.example` is the template.
- **Android:** stored in `SharedPreferences` via `AppSettings`, hashed with PBKDF2
  (`PasswordHasher`). Defaults `admin` / `party` seed on first run — change them in `AppSettings`
  (or add a settings screen). Session secret is random per launch.

## Build / run

```bash
scripts/build-web.sh        # build the React remote → android assets AND desktop embed dir
scripts/build-desktop.sh    # build-web.sh + `go build` the binary
cd web && npm run dev       # remote with hot reload (proxies to a server on :8080)
cd desktop && go test ./... # Go tests
cd android && ./gradlew test
```

After **any** change in `web/`, run `scripts/build-web.sh` or the servers serve a stale bundle.
`desktop/internal/webui/dist/index.html` is a committed placeholder so the Go `embed` compiles;
the real bundle (git-ignored `assets/`) is produced locally by the build script.

## Gotchas

- The desktop binary needs `mpv` and `yt-dlp` on PATH or it exits with an install hint.
- Android: the IFrame player must be visible to be ToS-compliant — keep it on the always-on
  Activity, never headless. The foreground service holds wake + Wi-Fi locks so the box doesn't nap.
- Don't commit built web assets (only the placeholder `index.html`).
