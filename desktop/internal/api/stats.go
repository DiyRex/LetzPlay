package api

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"github.com/DiyRex/LetzPlay/desktop/internal/auth"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
)

// Stats tracks what actually played this session: how many times each video was played and how
// many plays each requester contributed. It observes the queue (a track counts as "played" when
// the cursor lands on it) so the numbers reflect reality, not just what's queued.
type Stats struct {
	mu         sync.Mutex
	plays      map[string]*playCount // videoId -> count + title
	requesters map[string]int        // username -> plays contributed
}

type playCount struct {
	title string
	count int
}

type countDTO struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type statsDTO struct {
	MostPlayed    []countDTO `json:"mostPlayed"`
	TopRequesters []countDTO `json:"topRequesters"`
}

// NewStats starts observing the queue for the lifetime of ctx.
func NewStats(ctx context.Context, q *domain.Queue) *Stats {
	s := &Stats{plays: map[string]*playCount{}, requesters: map[string]int{}}
	updates, unsubscribe := q.Subscribe()
	go func() {
		defer unsubscribe()
		last := ""
		for {
			select {
			case <-ctx.Done():
				return
			case snap, ok := <-updates:
				if !ok {
					return
				}
				cur := snap.Current()
				if cur != nil && cur.VideoID != last {
					last = cur.VideoID
					s.record(cur.VideoID, cur.Title, cur.AddedBy)
				}
			}
		}
	}()
	return s
}

func (s *Stats) record(videoID, title, addedBy string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pc := s.plays[videoID]
	if pc == nil {
		pc = &playCount{}
		s.plays[videoID] = pc
	}
	pc.title = title
	pc.count++
	if addedBy != "" {
		s.requesters[addedBy]++
	}
}

func (s *Stats) top() statsDTO {
	s.mu.Lock()
	defer s.mu.Unlock()
	songs := make([]countDTO, 0, len(s.plays))
	for _, pc := range s.plays {
		songs = append(songs, countDTO{Label: pc.title, Count: pc.count})
	}
	people := make([]countDTO, 0, len(s.requesters))
	for name, n := range s.requesters {
		people = append(people, countDTO{Label: name, Count: n})
	}
	topN := func(c []countDTO) []countDTO {
		sort.Slice(c, func(i, j int) bool { return c[i].Count > c[j].Count })
		if len(c) > 10 {
			c = c[:10]
		}
		return c
	}
	return statsDTO{MostPlayed: topN(songs), TopRequesters: topN(people)}
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request, _ auth.Session) {
	writeJSON(w, http.StatusOK, s.stats.top())
}
