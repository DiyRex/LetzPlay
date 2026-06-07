// Command letzplay is the desktop (Linux/macOS) jukebox server: a laptop wired to a speaker runs
// this, picks an audio output in the TUI, and guests on the LAN queue YouTube music from the web
// remote. It mirrors the Android TV app's behaviour using the same web remote and wire protocol.
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DiyRex/LetzPlay/desktop/internal/api"
	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/config"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"github.com/DiyRex/LetzPlay/desktop/internal/player"
	"github.com/DiyRex/LetzPlay/desktop/internal/playlist"
	"github.com/DiyRex/LetzPlay/desktop/internal/tui"
	"github.com/DiyRex/LetzPlay/desktop/internal/webui"
	"github.com/DiyRex/LetzPlay/desktop/internal/youtube"
)

func main() {
	// Load optional .env (real environment always wins over the file). Flags, parsed below,
	// override everything — so precedence is: flag > environment > .env > default.
	config.LoadDotEnv("desktop/.env", ".env")

	port := flag.Int("port", config.Int(config.EnvPort, 8090),
		"HTTP port for the web remote (env "+config.EnvPort+")")
	adminPassword := flag.String("admin-password", config.String(config.EnvAdminPassword, "admin"),
		"password that grants admin (env "+config.EnvAdminPassword+")")
	guestPassword := flag.String("guest-password", config.String(config.EnvGuestPassword, "party"),
		"password guests use to join (env "+config.EnvGuestPassword+")")
	open := flag.Bool("open", config.Bool(config.EnvOpen, false),
		"allow guests to join with any password (env "+config.EnvOpen+")")
	headless := flag.Bool("headless", config.Bool(config.EnvHeadless, false),
		"run without the interactive TUI, for servers with no terminal (env "+config.EnvHeadless+")")
	maxPerUser := flag.Int("max-per-user", config.Int(config.EnvMaxPerUser, 0),
		"max queued songs per guest, 0 = unlimited (env "+config.EnvMaxPerUser+")")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// --- core wiring (mirrors the Android ServiceLocator) ---
	queue := domain.NewQueue()

	mpv := player.NewMpv(queue.OnStatus, queue.OnProgress, queue.Advance)
	if err := mpv.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "✗ Could not start the audio player.")
		fmt.Fprintln(os.Stderr, "  LetzPlay needs `mpv` and `yt-dlp` installed and on your PATH:")
		fmt.Fprintln(os.Stderr, "    macOS:  brew install mpv yt-dlp")
		fmt.Fprintln(os.Stderr, "    Linux:  sudo apt install mpv && pipx install yt-dlp")
		fmt.Fprintf(os.Stderr, "  (%v)\n", err)
		os.Exit(1)
	}
	defer mpv.Stop()
	warnIfYtdlpStale()
	prefetcher := player.NewPrefetcher()
	player.StartCoordinator(ctx, queue, mpv, prefetcher)

	authService, err := auth.NewService(*adminPassword, *guestPassword, !*open)
	if err != nil {
		log.Fatalf("auth init: %v", err)
	}
	sessions := auth.NewSessionManager(randomSecret())

	hub := api.NewHub(queue)
	go hub.Run(ctx)

	assets, err := webui.Assets()
	if err != nil {
		log.Fatalf("web assets: %v", err)
	}

	playlists := playlist.NewStore(playlist.DefaultPath())

	// Persist the queue across restarts and run autoplay radio when it empties (desktop-only).
	startQueuePersistence(ctx, queue)
	startRadio(ctx, queue)

	stats := api.NewStats(ctx, queue)
	server := api.NewServer(queue, mpv, authService, sessions, hub, assets, playlists, stats, *maxPerUser)
	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", *port), Handler: server.Handler()}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	url := lanURL(*port)

	if *headless {
		// No terminal (server box / CI): just announce the URL and run until interrupted.
		fmt.Printf("LetzPlay Musix running — open %s   (Ctrl+C to stop)\n", url)
		<-ctx.Done()
	} else {
		// --- foreground TUI ---
		program := tea.NewProgram(tui.New(ctx, queue, mpv, hub, url), tea.WithAltScreen())
		go func() {
			<-ctx.Done()
			program.Quit()
		}()
		if _, err := program.Run(); err != nil {
			log.Printf("tui: %v", err)
		}
	}

	// Tear everything down (TUI quit, or signal in headless mode).
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

// warnIfYtdlpStale prints the yt-dlp version and a gentle update hint. An outdated yt-dlp is the
// most common reason videos fail to play (YouTube changes frequently break old versions), which
// shows up as songs stuck buffering or the queue skipping through tracks.
func warnIfYtdlpStale() {
	out, err := exec.Command("yt-dlp", "--version").Output()
	if err != nil {
		return
	}
	version := strings.TrimSpace(string(out))
	fmt.Printf("yt-dlp %s — if songs won't play or the queue keeps skipping, update it:\n", version)
	fmt.Println("    yt-dlp -U     (or: brew upgrade yt-dlp / pipx upgrade yt-dlp)")
}

// randomSecret returns a 32-byte per-launch HMAC key for signing session cookies.
func randomSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("cannot generate session secret: %v", err)
	}
	return b
}

// lanURL returns the http URL guests should open, using the first site-local IPv4 address.
func lanURL(port int) string {
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ip4 := ipnet.IP.To4(); ip4 != nil && ip4.IsPrivate() {
					return fmt.Sprintf("http://%s:%d", ip4.String(), port)
				}
			}
		}
	}
	return fmt.Sprintf("http://localhost:%d", port)
}

func queuePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "letzplay", "queue.json")
}

// startQueuePersistence restores the saved track list on boot and saves it whenever it changes
// (only on list changes, not on every progress tick).
func startQueuePersistence(ctx context.Context, q *domain.Queue) {
	path := queuePath()
	if data, err := os.ReadFile(path); err == nil {
		var tracks []domain.Song
		if json.Unmarshal(data, &tracks) == nil && len(tracks) > 0 {
			q.Restore(tracks)
		}
	}

	updates, unsubscribe := q.Subscribe()
	go func() {
		defer unsubscribe()
		lastKey := ""
		for {
			select {
			case <-ctx.Done():
				return
			case snap, ok := <-updates:
				if !ok {
					return
				}
				key := trackKey(snap.Tracks)
				if key == lastKey {
					continue // list unchanged (e.g. just a progress update) — skip the write
				}
				lastKey = key
				if data, err := json.Marshal(snap.Tracks); err == nil {
					_ = os.MkdirAll(filepath.Dir(path), 0o755)
					_ = os.WriteFile(path, data, 0o644)
				}
			}
		}
	}()
}

func trackKey(tracks []domain.Song) string {
	ids := make([]string, len(tracks))
	for i, t := range tracks {
		ids[i] = t.ID
	}
	return strings.Join(ids, ",")
}

// startRadio watches for the queue running dry while autoplay is on, then appends related tracks
// (YouTube's "mix") so music keeps going. Desktop-only (uses yt-dlp).
func startRadio(ctx context.Context, q *domain.Queue) {
	updates, unsubscribe := q.Subscribe()
	go func() {
		defer unsubscribe()
		wasIdle := true
		fetching := false
		for {
			select {
			case <-ctx.Done():
				return
			case snap, ok := <-updates:
				if !ok {
					return
				}
				idle := snap.Status == domain.StatusIdle
				justEnded := idle && !wasIdle
				wasIdle = idle

				seed := snap.Current()
				if !justEnded || !snap.Autoplay || fetching || seed == nil {
					continue
				}
				fetching = true
				seedID := seed.VideoID
				existing := map[string]bool{}
				for _, t := range snap.Tracks {
					existing[t.VideoID] = true
				}
				go func() {
					defer func() { fetching = false }()
					mix, err := youtube.RadioMix(ctx, seedID, 5)
					if err != nil {
						return
					}
					for _, e := range mix {
						if existing[e.VideoID] {
							continue
						}
						q.Add(domain.Song{
							ID:             radioID(),
							VideoID:        e.VideoID,
							Title:          e.Title,
							ThumbnailURL:   "https://i.ytimg.com/vi/" + e.VideoID + "/hqdefault.jpg",
							AddedBy:        "Radio",
							AddedAtEpochMs: time.Now().UnixMilli(),
						})
					}
				}()
			}
		}
	}()
}

func radioID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("radio-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}
