// Package lyrics fetches (and caches) time-synced lyrics from lrclib.net, a free, keyless API.
package lyrics

import (
	"context"
	"encoding/json"
	"fmt"
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

type hit struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

// fetch tries several lrclib queries (structured artist/track first, then broad keyword searches,
// then track-only) and returns the first synced result — falling back to any plain lyrics. The
// extra attempts substantially improve coverage for non-English / "Artist - Title" style titles.
func (c *Client) fetch(ctx context.Context, title string) Lyrics {
	cleaned := cleanTitle(title)
	artist, track := splitArtistTrack(cleaned)

	type query map[string]string
	attempts := make([]query, 0, 5)
	if artist != "" && track != "" {
		attempts = append(attempts, query{"track_name": track, "artist_name": artist})
		attempts = append(attempts, query{"track_name": artist, "artist_name": track}) // swapped order
	}
	attempts = append(attempts, query{"q": cleaned})
	if track != "" && track != cleaned {
		attempts = append(attempts, query{"q": track})
	}
	// Mixed-script titles (e.g. romanized + native): try the latin-only form as a last resort.
	if ascii := asciiOnly(cleaned); ascii != "" && ascii != cleaned && len(ascii) >= 3 {
		attempts = append(attempts, query{"q": ascii})
	}

	var fallbackPlain string
	seen := map[string]bool{}
	for _, params := range attempts {
		key := fmt.Sprint(params)
		if seen[key] {
			continue
		}
		seen[key] = true
		for _, h := range c.search(ctx, params) {
			if synced := parseLRC(h.SyncedLyrics); len(synced) > 0 {
				return Lyrics{Found: true, Synced: synced, Plain: h.PlainLyrics}
			}
			if fallbackPlain == "" && strings.TrimSpace(h.PlainLyrics) != "" {
				fallbackPlain = h.PlainLyrics
			}
		}
	}
	if fallbackPlain != "" {
		return Lyrics{Found: true, Plain: fallbackPlain}
	}
	return Lyrics{}
}

// search calls lrclib /api/search with the given params and returns its hits (empty on any error).
func (c *Client) search(ctx context.Context, params map[string]string) []hit {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	endpoint := "https://lrclib.net/api/search?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "LetzPlayMusix (https://github.com/DiyRex/LetzPlay)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var hits []hit
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return nil
	}
	return hits
}

// splitArtistTrack splits a "Artist - Title" string. Returns ("","") when there's no separator.
func splitArtistTrack(s string) (artist, track string) {
	for _, sep := range []string{" - ", " – ", " — ", " | "} {
		if i := strings.Index(s, sep); i > 0 {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(sep):])
		}
	}
	return "", ""
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

// cleanTitle strips common noise to improve matching: bracketed tags ("(Official Video)", "[4K]"),
// trailing marketing phrases, and anything after a "|" (usually a channel/tag suffix).
var (
	bracketNoise  = regexp.MustCompile(`(?i)\s*[\(\[][^\)\]]*(official|video|audio|lyrics?|remaster|4k|hd|mv|m/v|cover|visualizer|full song)[^\)\]]*[\)\]]`)
	trailingNoise = regexp.MustCompile(`(?i)\s*[-–—|]\s*(official\s+\w+|lyric\s+video|music\s+video|full\s+song|audio|visualizer|hd|4k)\s*$`)
)

func cleanTitle(title string) string {
	s := title
	if i := strings.Index(s, "|"); i > 0 {
		s = s[:i] // drop channel/tag suffix after a pipe
	}
	s = bracketNoise.ReplaceAllString(s, "")
	s = trailingNoise.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// asciiOnly keeps ASCII letters/digits/spaces, collapsing runs of whitespace. Useful for titles
// that repeat a romanized name alongside native script (common for Sinhala/Tamil/Hindi uploads).
func asciiOnly(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		switch {
		case r < 128 && (r == ' ' || r == '-' || r == '\'' || isAlphaNum(r)):
			if r == ' ' {
				if prevSpace {
					continue
				}
				prevSpace = true
			} else {
				prevSpace = false
			}
			b.WriteRune(r)
		default:
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
