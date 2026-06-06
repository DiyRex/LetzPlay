package player

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Prefetcher resolves a YouTube video's direct audio stream URL ahead of time using yt-dlp, so
// that when a track is about to play mpv can open the direct URL immediately instead of waiting
// for yt-dlp at transition time (the main cause of the gap/"stuck buffering" between songs).
//
// Resolved URLs are cached with a TTL (googlevideo URLs are time-limited) and resolution is
// de-duplicated so the same id is never resolved twice concurrently.
type Prefetcher struct {
	mu       sync.Mutex
	cache    map[string]entry
	inflight map[string]bool
	ttl      time.Duration
	clock    func() time.Time
}

type entry struct {
	url     string
	expires time.Time
}

// NewPrefetcher returns a prefetcher with a sensible TTL for stream URLs.
func NewPrefetcher() *Prefetcher {
	return &Prefetcher{
		cache:    make(map[string]entry),
		inflight: make(map[string]bool),
		ttl:      3 * time.Hour,
		clock:    time.Now,
	}
}

// Peek returns a cached, non-expired direct URL for videoID without doing any work.
func (p *Prefetcher) Peek(videoID string) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.cache[videoID]; ok && p.clock().Before(e.expires) {
		return e.url, true
	}
	return "", false
}

// Warm resolves videoID in the background if it isn't already cached or in flight. Safe to call
// repeatedly (e.g. on every queue update) — it self-deduplicates and returns immediately.
func (p *Prefetcher) Warm(ctx context.Context, videoID string) {
	if videoID == "" {
		return
	}
	p.mu.Lock()
	if _, ok := p.cache[videoID]; ok {
		p.mu.Unlock()
		return
	}
	if p.inflight[videoID] {
		p.mu.Unlock()
		return
	}
	p.inflight[videoID] = true
	p.mu.Unlock()

	go func() {
		url, ok := resolveDirectURL(ctx, videoID)
		p.mu.Lock()
		delete(p.inflight, videoID)
		if ok {
			p.cache[videoID] = entry{url: url, expires: p.clock().Add(p.ttl)}
		}
		p.mu.Unlock()
	}()
}

// resolveDirectURL shells out to yt-dlp to get the best audio stream's direct URL.
func resolveDirectURL(ctx context.Context, videoID string) (string, bool) {
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "bestaudio/best",
		"-g", "https://www.youtube.com/watch?v="+videoID,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	// -g may print multiple URLs (one per line); take the first non-empty.
	for _, line := range strings.Split(string(out), "\n") {
		if u := strings.TrimSpace(line); u != "" {
			return u, true
		}
	}
	return "", false
}
