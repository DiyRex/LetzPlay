package com.letzplay.musix.player

import android.os.Handler
import android.os.Looper
import com.letzplay.musix.domain.model.PlaybackStatus
import com.letzplay.musix.domain.player.PlaybackController
import com.pierfrancescosoffritti.androidyoutubeplayer.core.player.PlayerConstants
import com.pierfrancescosoffritti.androidyoutubeplayer.core.player.YouTubePlayer
import com.pierfrancescosoffritti.androidyoutubeplayer.core.player.listeners.AbstractYouTubePlayerListener

/**
 * The single point of contact with the YouTube IFrame player library. Nothing else in the app
 * imports the player SDK — callers depend on the [PlaybackController] interface instead.
 *
 * Threading: the underlying player must be driven on the main thread, so every command is
 * posted to a main-looper handler. Player operations issued before the player is ready (or after
 * the screen is gone) are silently dropped, which is the correct behaviour for a TV that is the
 * sole renderer.
 *
 * State flows the other way through [listener]: the host view registers it, and the player's
 * own callbacks are translated into the app's [PlaybackStatus] and pushed out via the constructor
 * callbacks (wired to the queue in the DI layer).
 */
class YouTubePlayerController(
    private val onStatusChanged: (PlaybackStatus) -> Unit,
    private val onProgress: (positionSeconds: Float, durationSeconds: Float) -> Unit,
    private val onEnded: () -> Unit,
) : PlaybackController {

    private val main = Handler(Looper.getMainLooper())

    @Volatile
    private var player: YouTubePlayer? = null

    private var lastPosition = 0f
    private var lastDuration = 0f

    @Volatile
    private var loop = false

    @Volatile
    private var lastVideoId: String? = null

    /** Register this on the [com.pierfrancescosoffritti.androidyoutubeplayer.core.views.YouTubePlayerView]. */
    val listener = object : AbstractYouTubePlayerListener() {
        override fun onReady(youTubePlayer: YouTubePlayer) {
            player = youTubePlayer
        }

        override fun onStateChanged(youTubePlayer: YouTubePlayer, state: PlayerConstants.PlayerState) {
            when (state) {
                PlayerConstants.PlayerState.PLAYING -> onStatusChanged(PlaybackStatus.PLAYING)
                PlayerConstants.PlayerState.PAUSED -> onStatusChanged(PlaybackStatus.PAUSED)
                PlayerConstants.PlayerState.BUFFERING -> onStatusChanged(PlaybackStatus.BUFFERING)
                PlayerConstants.PlayerState.ENDED -> {
                    if (loop) {
                        // Repeat-one: replay the current track instead of advancing.
                        lastVideoId?.let { id -> onMain { it.loadVideo(id, 0f) } }
                    } else {
                        onStatusChanged(PlaybackStatus.ENDED)
                        onEnded() // auto-advance the queue
                    }
                }
                else -> Unit
            }
        }

        override fun onCurrentSecond(youTubePlayer: YouTubePlayer, second: Float) {
            lastPosition = second
            onProgress(lastPosition, lastDuration)
        }

        override fun onVideoDuration(youTubePlayer: YouTubePlayer, duration: Float) {
            lastDuration = duration
            onProgress(lastPosition, lastDuration)
        }
    }

    fun release() {
        player = null
    }

    override fun load(videoId: String) {
        lastVideoId = videoId
        onMain { it.loadVideo(videoId, 0f) }
    }
    override fun play() = onMain { it.play() }
    override fun pause() = onMain { it.pause() }
    override fun seekTo(seconds: Float) = onMain { it.seekTo(seconds) }
    override fun setVolume(percent: Int) = onMain { it.setVolume(percent.coerceIn(0, 100)) }
    override fun setLoop(loop: Boolean) { this.loop = loop }

    private inline fun onMain(crossinline action: (YouTubePlayer) -> Unit) {
        main.post { player?.let(action) }
    }
}
