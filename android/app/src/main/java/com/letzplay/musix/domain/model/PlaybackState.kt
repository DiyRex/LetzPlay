package com.letzplay.musix.domain.model

import kotlinx.serialization.Serializable

/** Coarse playback status mirrored from the YouTube IFrame player to every remote. */
@Serializable
enum class PlaybackStatus { IDLE, BUFFERING, PLAYING, PAUSED, ENDED }

/** What happens at the end of a track. */
@Serializable
enum class RepeatMode { OFF, ALL, ONE }

/**
 * A full, serializable snapshot of the jukebox. The model is a persistent playlist with a moving
 * cursor — NOT a consumed queue: [tracks] keeps every added song (played, current, upcoming) and
 * [currentIndex] points at the one playing (-1 when none). Advancing/previous/jumping only move
 * the cursor; songs leave only via explicit removal.
 *
 * This single object is the contract between player, queue, and web remotes; the server broadcasts
 * it on every change so all clients render from one source of truth.
 */
@Serializable
data class JukeboxSnapshot(
    val tracks: List<Song> = emptyList(),
    val currentIndex: Int = -1,
    val status: PlaybackStatus = PlaybackStatus.IDLE,
    val positionSeconds: Float = 0f,
    val durationSeconds: Float = 0f,
    val volume: Int = 100,
    val shuffle: Boolean = false,
    val repeat: RepeatMode = RepeatMode.OFF,
) {
    /** The currently-playing track, or null when [currentIndex] is out of range. */
    val current: Song? get() = tracks.getOrNull(currentIndex)
}
