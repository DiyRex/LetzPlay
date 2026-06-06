package player

import (
	"context"

	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
)

// StartCoordinator bridges queue state to the player: whenever the now-playing track changes, it
// tells the player to load it. This is the only place that decides "what should be playing",
// keeping the queue free of player concerns (the Go twin of the Android PlaybackCoordinator).
func StartCoordinator(ctx context.Context, q *domain.Queue, p domain.Player) {
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
				current := ""
				if snap.NowPlaying != nil {
					current = snap.NowPlaying.VideoID
				}
				if current != "" && current != last {
					p.Load(current)
				}
				last = current
			}
		}
	}()
}
