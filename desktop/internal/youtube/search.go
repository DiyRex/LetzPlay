package youtube

import (
	"context"
	"encoding/json"
	"os/exec"
)

// SearchResult is one hit from a keyword search.
type SearchResult struct {
	VideoID string
	Title   string
	Channel string
}

// Search runs a keyless YouTube search via yt-dlp's `ytsearch` and returns up to `limit` results.
// Desktop-only (needs yt-dlp on PATH).
func Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 15
	}
	// The search spec (ytsearchN:query) is the final positional arg.
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--flat-playlist",
		"--dump-single-json",
		"ytsearch"+itoa(limit)+":"+query,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var payload struct {
		Entries []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Uploader string `json:"uploader"`
			Channel  string `json:"channel"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(payload.Entries))
	for _, e := range payload.Entries {
		if e.ID == "" {
			continue
		}
		channel := e.Channel
		if channel == "" {
			channel = e.Uploader
		}
		results = append(results, SearchResult{VideoID: e.ID, Title: e.Title, Channel: channel})
	}
	return results, nil
}
