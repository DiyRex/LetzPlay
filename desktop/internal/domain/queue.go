package domain

import "sync"

// Queue owns jukebox state as a persistent playlist with a moving cursor (CurrentIndex), mirroring
// the Android app's MusicQueue.
//
// Crucially it does NOT consume tracks: a song that finishes stays in the list, and the cursor
// simply advances. Play/Previous/PlayNow are all cursor moves. Songs leave only via an explicit
// Remove. This is what keeps the full list visible on every remote.
//
// It is mechanical only (no player/network). All mutations run under one mutex and publish an
// immutable Snapshot to subscribers (the websocket broadcaster and the playback coordinator).
type Queue struct {
	mu          sync.Mutex
	snap        Snapshot
	subscribers map[int]chan Snapshot
	nextSubID   int
}

// NewQueue returns an empty, idle queue.
func NewQueue() *Queue {
	return &Queue{
		snap:        Snapshot{Tracks: []Song{}, CurrentIndex: -1, Status: StatusIdle, Volume: 100},
		subscribers: make(map[int]chan Snapshot),
	}
}

// Snapshot returns the current state (a value copy is safe to read concurrently).
func (q *Queue) Snapshot() Snapshot {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.snap
}

// Subscribe returns a channel primed with the current snapshot that then receives every future
// one, plus an unsubscribe func. The channel is buffered and lossy: a slow client never blocks.
func (q *Queue) Subscribe() (<-chan Snapshot, func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	id := q.nextSubID
	q.nextSubID++
	ch := make(chan Snapshot, 8)
	ch <- q.snap
	q.subscribers[id] = ch
	unsubscribe := func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		if c, ok := q.subscribers[id]; ok {
			delete(q.subscribers, id)
			close(c)
		}
	}
	return ch, unsubscribe
}

// Add appends a song to the playlist. If nothing is currently playing (idle), the new song becomes
// the cursor and starts; otherwise it just joins the list and plays in turn.
func (q *Queue) Add(s Song) {
	q.mutate(func(cur Snapshot) Snapshot {
		cur.Tracks = append(append([]Song{}, cur.Tracks...), s)
		if cur.Status == StatusIdle || cur.CurrentIndex < 0 {
			cur.CurrentIndex = len(cur.Tracks) - 1
			cur.Status = StatusBuffering
			cur.PositionSeconds = 0
			cur.DurationSeconds = 0
		}
		return cur
	})
}

// Remove deletes a song from the playlist (the only way a song leaves). Returns false if not found.
// The cursor is kept pointing at the same logical position; removing the current song advances to
// whatever shifts into its place.
func (q *Queue) Remove(songID string) bool {
	removed := false
	q.mutate(func(cur Snapshot) Snapshot {
		idx := indexOf(cur.Tracks, songID)
		if idx < 0 {
			return cur
		}
		removed = true
		cur.Tracks = append(append([]Song{}, cur.Tracks[:idx]...), cur.Tracks[idx+1:]...)

		switch {
		case idx < cur.CurrentIndex:
			cur.CurrentIndex--
		case idx == cur.CurrentIndex:
			if cur.CurrentIndex >= len(cur.Tracks) {
				cur.CurrentIndex = len(cur.Tracks) - 1
			}
			if cur.CurrentIndex < 0 {
				cur.Status = StatusIdle
			} else {
				cur.Status = StatusBuffering
			}
			cur.PositionSeconds = 0
			cur.DurationSeconds = 0
		}
		return cur
	})
	return removed
}

// Reorder moves a song to targetIndex, keeping the cursor on the same playing track.
func (q *Queue) Reorder(songID string, targetIndex int) bool {
	moved := false
	q.mutate(func(cur Snapshot) Snapshot {
		from := indexOf(cur.Tracks, songID)
		if from < 0 {
			return cur
		}
		currentID := ""
		if c := cur.Current(); c != nil {
			currentID = c.ID
		}
		items := append([]Song{}, cur.Tracks...)
		item := items[from]
		items = append(items[:from], items[from+1:]...)
		if targetIndex < 0 {
			targetIndex = 0
		}
		if targetIndex > len(items) {
			targetIndex = len(items)
		}
		items = append(items[:targetIndex], append([]Song{item}, items[targetIndex:]...)...)
		cur.Tracks = items
		if currentID != "" {
			cur.CurrentIndex = indexOf(cur.Tracks, currentID)
		}
		moved = true
		return cur
	})
	return moved
}

// Advance moves the cursor to the next track, going idle at the end (without removing anything).
func (q *Queue) Advance() {
	q.mutate(func(cur Snapshot) Snapshot {
		if cur.CurrentIndex+1 < len(cur.Tracks) {
			cur.CurrentIndex++
			cur.Status = StatusBuffering
		} else {
			cur.Status = StatusIdle // reached the end; cursor stays on the last track
		}
		cur.PositionSeconds = 0
		cur.DurationSeconds = 0
		return cur
	})
}

// Previous moves the cursor back one track. No-op (false) when already at the start.
func (q *Queue) Previous() bool {
	moved := false
	q.mutate(func(cur Snapshot) Snapshot {
		if cur.CurrentIndex <= 0 {
			return cur
		}
		cur.CurrentIndex--
		cur.Status = StatusBuffering
		cur.PositionSeconds = 0
		cur.DurationSeconds = 0
		moved = true
		return cur
	})
	return moved
}

// PlayNow jumps the cursor to a specific song (tap-to-play). Returns false if not found.
func (q *Queue) PlayNow(songID string) bool {
	moved := false
	q.mutate(func(cur Snapshot) Snapshot {
		idx := indexOf(cur.Tracks, songID)
		if idx < 0 {
			return cur
		}
		cur.CurrentIndex = idx
		cur.Status = StatusBuffering
		cur.PositionSeconds = 0
		cur.DurationSeconds = 0
		moved = true
		return cur
	})
	return moved
}

// OwnerOf returns who added a song, or "" if it isn't in the playlist.
func (q *Queue) OwnerOf(songID string) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if i := indexOf(q.snap.Tracks, songID); i >= 0 {
		return q.snap.Tracks[i].AddedBy
	}
	return ""
}

// --- playback metadata, pushed in from the player coordinator ---

func (q *Queue) OnStatus(status PlaybackStatus) {
	q.mutate(func(cur Snapshot) Snapshot { cur.Status = status; return cur })
}

func (q *Queue) OnProgress(position, duration float64) {
	q.mutate(func(cur Snapshot) Snapshot {
		cur.PositionSeconds = position
		cur.DurationSeconds = duration
		return cur
	})
}

func (q *Queue) SetVolume(volume int) {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	q.mutate(func(cur Snapshot) Snapshot { cur.Volume = volume; return cur })
}

func indexOf(tracks []Song, songID string) int {
	for i, s := range tracks {
		if s.ID == songID {
			return i
		}
	}
	return -1
}

// mutate applies transform under the lock, then fans the new snapshot out to subscribers.
func (q *Queue) mutate(transform func(Snapshot) Snapshot) {
	q.mu.Lock()
	q.snap = transform(q.snap)
	if q.snap.Tracks == nil {
		q.snap.Tracks = []Song{}
	}
	next := q.snap
	subs := make([]chan Snapshot, 0, len(q.subscribers))
	for _, ch := range q.subscribers {
		subs = append(subs, ch)
	}
	q.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- next:
		default:
		}
	}
}
