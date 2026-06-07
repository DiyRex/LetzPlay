// Package lyrics fetches (and caches) time-synced lyrics from lrclib.net, a free, keyless API.
package lyrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Line is one synced lyric line.
type Line struct {
	TimeMs int    `json:"timeMs"`
	Text   string `json:"text"`
}

// Lyrics is the response: synced lines if available, else plain text.
type Lyrics struct {
	Found  bool   `json:"found"`
	Synced []Line `json:"synced"`
	Plain  string `json:"plain"`
}

// Client fetches lyrics with a small in-memory cache keyed by the search title.
type Client struct {
	http  *http.Client
	mu    sync.Mutex
	cache map[string]Lyrics
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 6 * time.Second}, cache: make(map[string]Lyrics)}
}

var lrcLine = regexp.MustCompile(`\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?\]`)

// Get returns lyrics for a track title (typically the YouTube title). Cached; failures return
// Found=false rather than an error so the UI can simply show "no lyrics".
func (c *Client) Get(ctx context.Context, title string) Lyrics {
	key := strings.TrimSpace(title)
	if key == "" {
		return Lyrics{}
	}
	c.mu.Lock()
	if cached, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return cached
	}
	c.mu.Unlock()

	result := c.fetch(ctx, key)
	c.mu.Lock()
	c.cache[key] = result
	c.mu.Unlock()
	return result
}

func (c *Client) fetch(ctx context.Context, title string) Lyrics {
	endpoint := "https://lrclib.net/api/search?q=" + url.QueryEscape(cleanTitle(title))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Lyrics{}
	}
	req.Header.Set("User-Agent", "LetzPlayMusix (https://github.com/DiyRex/LetzPlay)")
	resp, err := c.http.Do(req)
	if err != nil {
		return Lyrics{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Lyrics{}
	}

	var hits []struct {
		SyncedLyrics string `json:"syncedLyrics"`
		PlainLyrics  string `json:"plainLyrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return Lyrics{}
	}
	for _, h := range hits {
		if synced := parseLRC(h.SyncedLyrics); len(synced) > 0 {
			return Lyrics{Found: true, Synced: synced, Plain: h.PlainLyrics}
		}
	}
	for _, h := range hits {
		if strings.TrimSpace(h.PlainLyrics) != "" {
			return Lyrics{Found: true, Plain: h.PlainLyrics}
		}
	}
	return Lyrics{}
}

// parseLRC turns an LRC string into ordered timed lines.
func parseLRC(lrc string) []Line {
	if strings.TrimSpace(lrc) == "" {
		return nil
	}
	var lines []Line
	for _, raw := range strings.Split(lrc, "\n") {
		matches := lrcLine.FindAllStringSubmatchIndex(raw, -1)
		if len(matches) == 0 {
			continue
		}
		// Text is whatever follows the last timestamp on the line.
		text := strings.TrimSpace(raw[matches[len(matches)-1][1]:])
		for _, m := range matches {
			min, _ := strconv.Atoi(raw[m[2]:m[3]])
			sec, _ := strconv.Atoi(raw[m[4]:m[5]])
			ms := 0
			if m[6] >= 0 {
				frac := raw[m[6]:m[7]]
				for len(frac) < 3 {
					frac += "0"
				}
				ms, _ = strconv.Atoi(frac[:3])
			}
			lines = append(lines, Line{TimeMs: (min*60+sec)*1000 + ms, Text: text})
		}
	}
	return lines
}

// cleanTitle strips common noise ("(Official Video)", "[4K]", "feat.") to improve matching.
var noise = regexp.MustCompile(`(?i)\s*[\(\[][^\)\]]*(official|video|audio|lyrics?|remaster|4k|hd|mv|m/v)[^\)\]]*[\)\]]`)

func cleanTitle(title string) string {
	return strings.TrimSpace(noise.ReplaceAllString(title, ""))
}
