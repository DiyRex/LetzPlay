package domain

import "testing"

func song(id, by string) Song {
	return Song{ID: id, VideoID: "vid" + id, Title: "Song " + id, AddedBy: by}
}

func ids(tracks []Song) string {
	out := ""
	for i, s := range tracks {
		if i > 0 {
			out += ","
		}
		out += s.ID
	}
	return out
}

func TestFirstAddStartsPlaying(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	s := q.Snapshot()
	if s.CurrentIndex != 0 || s.Current().ID != "a" {
		t.Fatalf("want cursor on 'a', got idx=%d", s.CurrentIndex)
	}
	if s.Status != StatusBuffering {
		t.Fatalf("want BUFFERING, got %s", s.Status)
	}
}

func TestSongsArentConsumed(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	q.Advance() // a -> b
	s := q.Snapshot()
	// The full list must still contain BOTH songs; only the cursor moved.
	if ids(s.Tracks) != "a,b" {
		t.Fatalf("tracks should persist, got %s", ids(s.Tracks))
	}
	if s.CurrentIndex != 1 || s.Current().ID != "b" {
		t.Fatalf("cursor should be on b, got idx=%d", s.CurrentIndex)
	}
}

func TestAdvancePastEndGoesIdleButKeepsList(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Advance()
	s := q.Snapshot()
	if s.Status != StatusIdle {
		t.Fatalf("want IDLE at end, got %s", s.Status)
	}
	if ids(s.Tracks) != "a" || s.CurrentIndex != 0 {
		t.Fatalf("list should keep 'a' with cursor on it, got %s idx=%d", ids(s.Tracks), s.CurrentIndex)
	}
}

func TestPreviousAndPlayNow(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	q.Add(song("c", "carol"))
	q.Advance() // -> b
	q.Advance() // -> c
	if !q.Previous() || q.Snapshot().Current().ID != "b" {
		t.Fatalf("previous should return to b")
	}
	if !q.PlayNow("a") || q.Snapshot().Current().ID != "a" {
		t.Fatalf("playNow should jump to a")
	}
	if q.PlayNow("zzz") {
		t.Fatalf("playNow on missing id should be false")
	}
	// List unchanged throughout.
	if ids(q.Snapshot().Tracks) != "a,b,c" {
		t.Fatalf("list must be intact, got %s", ids(q.Snapshot().Tracks))
	}
}

func TestRemoveAdjustsCursor(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	q.Add(song("c", "carol"))
	q.Advance() // cursor on b (index 1)

	q.Remove("a") // removing before cursor shifts it left
	s := q.Snapshot()
	if ids(s.Tracks) != "b,c" || s.Current().ID != "b" {
		t.Fatalf("after removing a: tracks=%s current=%s", ids(s.Tracks), s.Current().ID)
	}
	if !q.Remove("b") { // remove current -> advances into the song now at that slot
		t.Fatal("expected remove b to succeed")
	}
	if q.Snapshot().Current().ID != "c" {
		t.Fatalf("after removing current b, cursor should land on c, got %s", q.Snapshot().Current().ID)
	}
}

func TestReorderKeepsCursorOnSameSong(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	q.Add(song("c", "carol"))
	q.Advance() // cursor on b
	q.Reorder("c", 0)
	s := q.Snapshot()
	if ids(s.Tracks) != "c,a,b" {
		t.Fatalf("reorder want c,a,b got %s", ids(s.Tracks))
	}
	if s.Current().ID != "b" {
		t.Fatalf("cursor should still be on b after reorder, got %s", s.Current().ID)
	}
}

func TestFairQueueInterleaves(t *testing.T) {
	q := NewQueue()
	q.SetFairQueue(true)
	q.Add(song("a1", "alice")) // now playing
	q.Add(song("a2", "alice")) // upcoming: a2
	q.Add(song("a3", "alice")) // upcoming: a2,a3
	q.Add(song("b1", "bob"))   // bob's 1st jumps ahead of alice's 3rd
	// Upcoming should interleave: a2, b1, a3 (bob doesn't wait behind alice's backlog).
	if got := ids(q.Snapshot().Tracks); got != "a1,a2,b1,a3" {
		t.Fatalf("fair interleave want a1,a2,b1,a3 got %s", got)
	}
}

func TestOwnerOf(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	if q.OwnerOf("b") != "bob" {
		t.Fatalf("owner of b should be bob")
	}
	if q.OwnerOf("zzz") != "" {
		t.Fatalf("owner of missing should be empty")
	}
}
