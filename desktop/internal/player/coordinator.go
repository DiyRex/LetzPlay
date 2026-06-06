package player

import (
	"context"

	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
)

// StartCoordinator bridges queue state to the mpv player and drives smooth transitions.
//
// Responsibilities (the Go twin of the Android PlaybackCoordinator, plus prefetch):
//   - When the now-playing track changes, load it — preferring a pre-resolved direct URL from the
//     Prefetcher so the switch is near-instant instead of waiting on yt-dlp.
//   - Continuously warm the *next* track so its stream URL is ready before it's needed.
//
// It is the only place that decides "what should be playing", keeping the queue free of player
// concerns and the player free of queue logic.
func StartCoordinator(ctx context.Context, q *domain.Queue, mpv *Mpv, pf *Prefetcher) {
	updates, unsubscribe := q.Subscribe()
	go func() {
		defer unsubscribe()
		lastLoaded := ""
		for {
			select {
			case <-ctx.Done():
				return
			case snap, ok := <-updates:
				if !ok {
					return
				}
				current := videoID(snap.NowPlaying)
				next := ""
				if len(snap.Queue) > 0 {
					next = snap.Queue[0].VideoID
				}

				if current != "" && current != lastLoaded {
					// Prefer a pre-resolved direct URL (instant); fall back to letting mpv resolve.
					if url, ready := pf.Peek(current); ready {
						mpv.LoadURL(url)
					} else {
						mpv.Load(current)
					}
					lastLoaded = current
				}
				if current == "" {
					lastLoaded = ""
				}

				// Resolve the next track ahead of time so the upcoming transition is smooth.
				pf.Warm(ctx, next)
			}
		}
	}()
}

func videoID(s *domain.Song) string {
	if s == nil {
		return ""
	}
	return s.VideoID
}
