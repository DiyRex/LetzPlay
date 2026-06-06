package youtube

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// PlaylistEntry is one video discovered inside a playlist.
type PlaylistEntry struct {
	VideoID string
	Title   string
}

// IsPlaylistURL reports whether a URL refers to a YouTube playlist (has a `list=` parameter).
func IsPlaylistURL(url string) bool {
	return strings.Contains(url, "list=")
}

// ExpandPlaylist lists a playlist's entries via yt-dlp's flat (no per-video network) mode.
// Returns at most `limit` entries. Desktop-only — relies on yt-dlp being installed.
func ExpandPlaylist(ctx context.Context, url string, limit int) ([]PlaylistEntry, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--flat-playlist",
		"--dump-single-json",
		"--playlist-end", itoa(limit),
		url,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var payload struct {
		Entries []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, err
	}

	entries := make([]PlaylistEntry, 0, len(payload.Entries))
	for _, e := range payload.Entries {
		if e.ID == "" {
			continue // unavailable/private entries have no id
		}
		entries = append(entries, PlaylistEntry{VideoID: e.ID, Title: e.Title})
		if len(entries) >= limit {
			break
		}
	}
	return entries, nil
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
