package domain

import "testing"

func song(id, by string) Song {
	return Song{ID: id, VideoID: "vid" + id, Title: "Song " + id, AddedBy: by}
}

func TestFirstSongBecomesNowPlaying(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	snap := q.Snapshot()
	if snap.NowPlaying == nil || snap.NowPlaying.ID != "a" {
		t.Fatalf("expected 'a' now playing, got %+v", snap.NowPlaying)
	}
	if len(snap.Queue) != 0 {
		t.Fatalf("expected empty queue, got %d", len(snap.Queue))
	}
	if snap.Status != StatusBuffering {
		t.Fatalf("expected BUFFERING, got %s", snap.Status)
	}
}

func TestSubsequentSongsQueue(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	snap := q.Snapshot()
	if snap.NowPlaying.ID != "a" || len(snap.Queue) != 1 || snap.Queue[0].ID != "b" {
		t.Fatalf("unexpected state: now=%v queue=%v", snap.NowPlaying, snap.Queue)
	}
}

func TestAdvanceAndExhaustion(t *testing.T) {
	q := NewQueue()
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	q.Advance()
	if q.Snapshot().NowPlaying.ID != "b" {
		t.Fatalf("expected 'b' after advance")
	}
	q.Advance()
	snap := q.Snapshot()
	if snap.NowPlaying != nil || snap.Status != StatusIdle {
		t.Fatalf("expected idle after exhausting queue, got %+v / %s", snap.NowPlaying, snap.Status)
	}
}

func TestRemoveReorderOwner(t *testing.T) {
	q := NewQueue()
	q.Add(song("now", "alice"))
	q.Add(song("a", "alice"))
	q.Add(song("b", "bob"))
	q.Add(song("c", "carol"))

	if !q.Remove("b") {
		t.Fatal("expected remove to succeed")
	}
	if got := ids(q.Snapshot().Queue); got != "a,c" {
		t.Fatalf("after remove want a,c got %s", got)
	}
	q.Reorder("c", 0)
	if got := ids(q.Snapshot().Queue); got != "c,a" {
		t.Fatalf("after reorder want c,a got %s", got)
	}
	if q.Remove("now") {
		t.Fatal("now-playing should not be removable from the pending queue")
	}
	if q.OwnerOf("c") != "carol" {
		t.Fatalf("expected carol owns c")
	}
	if q.OwnerOf("now") != "" {
		t.Fatalf("now-playing is not in pending queue, owner should be empty")
	}
}

func ids(songs []Song) string {
	out := ""
	for i, s := range songs {
		if i > 0 {
			out += ","
		}
		out += s.ID
	}
	return out
}
