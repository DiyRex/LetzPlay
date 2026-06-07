// Package playlist provides named, persisted collections of songs that users can build on the web
// and load into the queue. State is saved to a JSON file so playlists survive restarts.
package playlist

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Song is a lightweight playlist entry (no queue/session fields).
type Song struct {
	VideoID      string `json:"videoId"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
}

// Playlist is a named, ordered collection of songs.
type Playlist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Songs       []Song `json:"songs"`
	UpdatedAtMs int64  `json:"updatedAtMs"`
}

// Summary is the list-view form (no songs, just a count).
type Summary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Store is a thread-safe, file-backed collection of playlists.
type Store struct {
	mu        sync.Mutex
	path      string
	playlists map[string]*Playlist
	now       func() int64
}

// NewStore loads playlists from path (missing/corrupt files start empty).
func NewStore(path string) *Store {
	s := &Store{
		path:      path,
		playlists: make(map[string]*Playlist),
		now:       func() int64 { return time.Now().UnixMilli() },
	}
	s.load()
	return s
}

// DefaultPath returns a per-user config path for the playlist file.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "letzplay", "playlists.json")
}

// List returns summaries sorted by name.
func (s *Store) List() []Summary {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Summary, 0, len(s.playlists))
	for _, p := range s.playlists {
		out = append(out, Summary{ID: p.ID, Name: p.Name, Count: len(p.Songs)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get returns a deep-ish copy of a playlist.
func (s *Store) Get(id string) (Playlist, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.playlists[id]
	if !ok {
		return Playlist{}, false
	}
	cp := *p
	cp.Songs = append([]Song{}, p.Songs...)
	return cp, true
}

// Create makes a new playlist (optionally seeded with songs) and returns it.
func (s *Store) Create(name string, songs []Song) Playlist {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &Playlist{ID: newID(), Name: name, Songs: songs, UpdatedAtMs: s.now()}
	s.playlists[p.ID] = p
	s.persist()
	return *p
}

// Delete removes a playlist. Returns false if it didn't exist.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.playlists[id]; !ok {
		return false
	}
	delete(s.playlists, id)
	s.persist()
	return true
}

// Rename changes a playlist's name. Returns false if not found.
func (s *Store) Rename(id, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.playlists[id]
	if !ok {
		return false
	}
	p.Name = name
	p.UpdatedAtMs = s.now()
	s.persist()
	return true
}

// AddSong appends a song (deduped by videoId). Returns false if the playlist is missing.
func (s *Store) AddSong(id string, song Song) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.playlists[id]
	if !ok {
		return false
	}
	for _, existing := range p.Songs {
		if existing.VideoID == song.VideoID {
			return true // already present; no-op
		}
	}
	p.Songs = append(p.Songs, song)
	p.UpdatedAtMs = s.now()
	s.persist()
	return true
}

// RemoveSong drops a song by videoId. Returns false if the playlist is missing.
func (s *Store) RemoveSong(id, videoID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.playlists[id]
	if !ok {
		return false
	}
	filtered := p.Songs[:0:0]
	for _, song := range p.Songs {
		if song.VideoID != videoID {
			filtered = append(filtered, song)
		}
	}
	p.Songs = filtered
	p.UpdatedAtMs = s.now()
	s.persist()
	return true
}

// --- persistence ---

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var stored []Playlist
	if json.Unmarshal(data, &stored) != nil {
		return
	}
	for i := range stored {
		p := stored[i]
		s.playlists[p.ID] = &p
	}
}

// persist writes the store atomically (write temp + rename). Caller holds the lock.
func (s *Store) persist() {
	all := make([]Playlist, 0, len(s.playlists))
	for _, p := range s.playlists {
		all = append(all, *p)
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}
