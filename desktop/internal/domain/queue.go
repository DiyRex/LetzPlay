package domain

import "sync"

// Queue is the single owner of jukebox state, mirroring the Android app's MusicQueue.
//
// It is mechanical only: it never touches the player or network. All mutations run under one
// mutex and then publish an immutable Snapshot to every subscriber, so the websocket broadcaster
// and the playback coordinator each react to the same consistent stream of states.
type Queue struct {
	mu          sync.Mutex
	snap        Snapshot
	subscribers map[int]chan Snapshot
	nextSubID   int
}

// NewQueue returns an empty, idle queue.
func NewQueue() *Queue {
	return &Queue{
		snap:        Snapshot{Queue: []Song{}, Status: StatusIdle, Volume: 100},
		subscribers: make(map[int]chan Snapshot),
	}
}

// Snapshot returns the current state (a value copy is safe to read concurrently).
func (q *Queue) Snapshot() Snapshot {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.snap
}

// Subscribe returns a channel that receives every future snapshot plus the current one, and an
// unsubscribe func the caller must invoke when done. The channel is buffered and lossy under
// backpressure: a slow client never blocks the queue.
func (q *Queue) Subscribe() (<-chan Snapshot, func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	id := q.nextSubID
	q.nextSubID++
	ch := make(chan Snapshot, 8)
	ch <- q.snap // prime with current state
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

// Add queues a song, promoting it to now-playing if nothing is playing.
func (q *Queue) Add(s Song) {
	q.mutate(func(cur Snapshot) Snapshot {
		if cur.NowPlaying == nil {
			cur.NowPlaying = &s
			cur.Status = StatusBuffering
		} else {
			cur.Queue = append(append([]Song{}, cur.Queue...), s)
		}
		return cur
	})
}

// Remove deletes a pending song by id, returning whether it was present.
func (q *Queue) Remove(songID string) bool {
	removed := false
	q.mutate(func(cur Snapshot) Snapshot {
		filtered := cur.Queue[:0:0]
		for _, s := range cur.Queue {
			if s.ID == songID {
				removed = true
				continue
			}
			filtered = append(filtered, s)
		}
		cur.Queue = filtered
		return cur
	})
	return removed
}

// Reorder moves a pending song to targetIndex (clamped). Returns whether it was found.
func (q *Queue) Reorder(songID string, targetIndex int) bool {
	moved := false
	q.mutate(func(cur Snapshot) Snapshot {
		from := -1
		for i, s := range cur.Queue {
			if s.ID == songID {
				from = i
				break
			}
		}
		if from < 0 {
			return cur
		}
		items := append([]Song{}, cur.Queue...)
		item := items[from]
		items = append(items[:from], items[from+1:]...)
		if targetIndex < 0 {
			targetIndex = 0
		}
		if targetIndex > len(items) {
			targetIndex = len(items)
		}
		items = append(items[:targetIndex], append([]Song{item}, items[targetIndex:]...)...)
		cur.Queue = items
		moved = true
		return cur
	})
	return moved
}

// OwnerOf returns who queued a pending song, or "" if it is not in the queue.
func (q *Queue) OwnerOf(songID string) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, s := range q.snap.Queue {
		if s.ID == songID {
			return s.AddedBy
		}
	}
	return ""
}

// Advance moves to the next track, going idle when the queue empties.
func (q *Queue) Advance() {
	q.mutate(func(cur Snapshot) Snapshot {
		if len(cur.Queue) == 0 {
			cur.NowPlaying = nil
			cur.Status = StatusIdle
			cur.PositionSeconds = 0
			cur.DurationSeconds = 0
			return cur
		}
		next := cur.Queue[0]
		cur.NowPlaying = &next
		cur.Queue = append([]Song{}, cur.Queue[1:]...)
		cur.Status = StatusBuffering
		cur.PositionSeconds = 0
		cur.DurationSeconds = 0
		return cur
	})
}

// OnStatus / OnProgress / SetVolume are pushed in from the player coordinator.

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

// mutate applies transform under the lock, then fans the new snapshot out to subscribers.
func (q *Queue) mutate(transform func(Snapshot) Snapshot) {
	q.mu.Lock()
	q.snap = transform(q.snap)
	if q.snap.Queue == nil {
		q.snap.Queue = []Song{}
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
		default: // subscriber is behind; drop — it will get the next one
		}
	}
}
