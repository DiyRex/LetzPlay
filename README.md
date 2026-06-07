# LetzPlay Musix

A self-hosted **YouTube jukebox** for a room. One device — an **Android TV box** or a
**laptop/desktop** — connects to a speaker and runs a server. Everyone else on the same Wi-Fi opens
a web page on their phone, logs in with the party password, and queues YouTube songs. The host
device plays them in order; the queue updates live on every phone, and everyone can see who's
connected and who added each track.

Two backends, **one shared web remote**:

| Host | Plays via | Best for |
|------|-----------|----------|
| **Android TV app** (`android/`) | YouTube IFrame player in a WebView (ToS-compliant), audio out the box's AUX/HDMI | a dedicated TV box left plugged into the speakers |
| **Desktop binary** (`desktop/`) | `mpv` + `yt-dlp`, with a navigable terminal UI to pick the audio output device | a laptop you connect to the speaker for the night |

Both serve the same **React + shadcn/ui** remote (`web/`) and speak an identical JSON/WebSocket
protocol, so a phone doesn't know or care which one it's talking to.

---

## Repository layout

```
LetzPlayMusix/
├── web/        Shared React + shadcn/ui remote (Vite + Tailwind). Built once, used by both servers.
├── android/    Android TV app (Kotlin, Ktor embedded server, android-youtube-player).
├── desktop/    Go binary (Linux/macOS): mpv playback + Bubble Tea TUI + embedded web server.
├── scripts/    build-web.sh (build remote → both servers), build-desktop.sh (remote + Go binary).
└── CLAUDE.md   Architecture & conventions for contributors and AI assistants.
```

The remote is built in `web/` and copied into `android/app/src/main/assets/web` and embedded into
the Go binary at `desktop/internal/webui/dist`. **Always run `scripts/build-web.sh` after changing
anything in `web/`.**

---

## Features

- 🎵 Paste a YouTube link (watch / `youtu.be` / shorts / embed / live / bare id) or a **playlist** → it queues.
- 📜 Persistent song list (played, current, upcoming all stay): add, remove, reorder, **tap any song to play it**.
- ⏯ Full transport: **previous / play-pause / next**, an interactive **seek bar**, **volume**, **shuffle**, and **repeat** (off / all / one) — all live across every remote.
- 💾 **Playlists**: create, add songs, save the current queue as a playlist, and load a playlist back into the queue (persisted on desktop).
- 📑 Tabbed remote: **Now Playing · Songs · Playlists**.
- 👮 **Login with roles.** A shared *party password* (guest) and an *admin password* (full control).
  Guests manage only songs they added; admins control playback and any song.
- 🧑‍🤝‍🧑 **Live presence.** See who's connected, and who added each track.
- 🔊 **Desktop:** navigable TUI to choose the audio output device, with live now-playing and controls.
- 📺 **Android TV:** full-screen "now playing" plus a QR code so phones can connect in one scan.
- ⚡ Everything updates in real time over WebSockets — no refreshing.

---

## Quick start

### Desktop (laptop → speaker)

Prerequisites: **Go 1.22+**, **Node 18+**, and **mpv + yt-dlp** on your PATH.

```bash
# macOS
brew install go node mpv yt-dlp
# Debian/Ubuntu
sudo apt install golang nodejs npm mpv && pipx install yt-dlp

# Configure (port + passwords) via env file, then build + run
cp desktop/.env.example desktop/.env   # edit it: port, admin/guest passwords
./scripts/build-desktop.sh
./desktop/bin/letzplay                  # reads desktop/.env
```

The TUI opens: pick your audio output with ↑/↓ + Enter, and the printed `http://<your-ip>:8090`
is what guests open.

**Configuration** comes from `desktop/.env` (or real env vars), and flags override either.
Precedence: **flag > environment > `.env` > default**.

| Flag | Env var | Default | Meaning |
|------|---------|---------|---------|
| `--port` | `LETZPLAY_PORT` | `8090` | web remote port |
| `--admin-password` | `LETZPLAY_ADMIN_PASSWORD` | `admin` | grants full control |
| `--guest-password` | `LETZPLAY_GUEST_PASSWORD` | `party` | lets guests join |
| `--open` | `LETZPLAY_OPEN` | `false` | accept *any* password as a guest |
| `--headless` | `LETZPLAY_HEADLESS` | `false` | run without the TUI (server box / CI) |

**TUI keys:** `↑/↓` select device · `enter` apply · `space` play/pause · `n` skip · `+/-` volume ·
`r` refresh devices · `q` quit.

### Android TV box

Prerequisites: **Android Studio** (or the Android SDK + JDK 17) and **Node 18+**.

```bash
./scripts/build-web.sh                 # builds the remote into the app assets
cd android && ./gradlew installDebug   # or open the android/ folder in Android Studio
```

Launch **LetzPlay Musix** from the TV home screen. The screen shows a QR code and URL; scan it
from a phone on the same Wi-Fi. Audio follows the box's normal output (AUX/HDMI).

> First-run defaults are `admin` / `party`. Change them — see CLAUDE.md for where credentials live.

---

## Development

```bash
# Web remote with hot reload (proxies /api and /ws to a running server on :8080)
cd web && npm install && npm run dev

# Go tests / Android unit tests
cd desktop && go test ./...
cd android && ./gradlew test
```

---

## How it fits together

```
        Phones (React + shadcn remote)
                  │  REST + WebSocket  ({snapshot, users})
                  ▼
   ┌───────────────────────────────┐
   │  Server (Android Ktor  OR  Go) │
   │  ┌──────────┐   ┌───────────┐  │
   │  │  Queue   │──▶│  Player   │──┼──▶  Speaker
   │  │ (state)  │   │ (IFrame / │  │
   │  └──────────┘   │   mpv)    │  │
   │   auth + presence└──────────┘  │
   └───────────────────────────────┘
```

The **Queue** is the single source of truth; the **Player** is an interface (`PlaybackController`
in Kotlin, `domain.Player` in Go) so playback logic stays testable and swappable. See
[CLAUDE.md](./CLAUDE.md) for the full architecture and conventions.

## License & terms

For personal/home use. The Android app uses YouTube's official IFrame player (ToS-compliant). The
desktop binary uses `mpv`/`yt-dlp` to play audio — review YouTube's Terms of Service for your use.
