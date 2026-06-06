package com.letzplay.musix.domain.player

/**
 * Abstraction over the actual video player. The server and queue depend on this interface,
 * never on the concrete YouTube WebView player — so playback logic stays testable and the
 * player implementation can be swapped without touching callers.
 */
interface PlaybackController {
    fun load(videoId: String)
    fun play()
    fun pause()
    fun seekTo(seconds: Float)
    fun setVolume(percent: Int)
}
