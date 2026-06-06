// Command letzplay is the desktop (Linux/macOS) jukebox server: a laptop wired to a speaker runs
// this, picks an audio output in the TUI, and guests on the LAN queue YouTube music from the web
// remote. It mirrors the Android TV app's behaviour using the same web remote and wire protocol.
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DiyRex/LetzPlay/desktop/internal/api"
	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/config"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"github.com/DiyRex/LetzPlay/desktop/internal/player"
	"github.com/DiyRex/LetzPlay/desktop/internal/tui"
	"github.com/DiyRex/LetzPlay/desktop/internal/webui"
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

	server := api.NewServer(queue, mpv, authService, sessions, hub, assets)
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
