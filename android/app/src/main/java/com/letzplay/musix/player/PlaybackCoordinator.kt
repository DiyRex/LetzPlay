package com.letzplay.musix.player

import com.letzplay.musix.domain.player.PlaybackController
import com.letzplay.musix.domain.queue.MusicQueue
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.onEach

/**
 * Bridges queue state to the player: whenever the now-playing track changes, it tells the
 * [PlaybackController] to load that video. This is the only place that decides "what should be
 * on screen", which keeps the queue free of player concerns and the player free of queue logic.
 *
 * Scoped to the host view's lifecycle — the collection stops when the screen goes away.
 */
class PlaybackCoordinator(
    queue: MusicQueue,
    controller: PlaybackController,
    scope: CoroutineScope,
) {
    init {
        queue.snapshot
            .map { it.nowPlaying?.videoId }
            .distinctUntilChanged()
            .onEach { videoId -> if (videoId != null) controller.load(videoId) }
            .launchIn(scope)
    }
}
