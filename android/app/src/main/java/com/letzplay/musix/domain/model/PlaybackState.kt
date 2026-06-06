package com.letzplay.musix.domain.model

import kotlinx.serialization.Serializable

/** Coarse playback status mirrored from the YouTube IFrame player to every remote. */
@Serializable
enum class PlaybackStatus { IDLE, BUFFERING, PLAYING, PAUSED, ENDED }

/**
 * A full, serializable snapshot of what the jukebox is doing right now.
 * This single object is the contract between the player, the queue, and the web remotes —
 * the server broadcasts it on every change so all clients render from one source of truth.
 */
@Serializable
data class JukeboxSnapshot(
    val nowPlaying: Song? = null,
    val queue: List<Song> = emptyList(),
    val status: PlaybackStatus = PlaybackStatus.IDLE,
    val positionSeconds: Float = 0f,
    val durationSeconds: Float = 0f,
    val volume: Int = 100,
)
