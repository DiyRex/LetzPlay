package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// Metadata is a video's title and thumbnail, resolved from YouTube's keyless oEmbed endpoint.
type Metadata struct {
	Title        string
	ThumbnailURL string
}

// MetadataClient fetches titles via oEmbed (no API key, no quota). Failures return ok=false so
// callers can fall back to the raw video id.
type MetadataClient struct {
	http *http.Client
}

// NewMetadataClient builds a client with a short timeout suitable for an interactive "add song".
func NewMetadataClient() *MetadataClient {
	return &MetadataClient{http: &http.Client{Timeout: 5 * time.Second}}
}

// Fetch resolves metadata for a video id.
func (c *MetadataClient) Fetch(ctx context.Context, videoID string) (Metadata, bool) {
	endpoint := "https://www.youtube.com/oembed?format=json&url=" +
		url.QueryEscape("https://www.youtube.com/watch?v="+videoID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Metadata{}, false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Metadata{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Metadata{}, false
	}

	var body struct {
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Metadata{}, false
	}
	return Metadata{Title: body.Title, ThumbnailURL: body.ThumbnailURL}, true
}
