package com.letzplay.musix.domain.queue

import com.letzplay.musix.domain.model.JukeboxSnapshot
import com.letzplay.musix.domain.model.PlaybackStatus
import com.letzplay.musix.domain.model.Song
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * The single owner of jukebox state: now-playing, the pending queue, and playback metadata.
 *
 * Design notes:
 * - This class is *mechanical*. It never touches the player, the network, or Android APIs.
 *   That keeps it pure, unit-testable, and free of the lifecycle bugs that plague god-objects.
 * - Authorization (who may remove/skip) lives in the server route layer, not here.
 * - All mutations funnel through [mutate] under a single lock, then publish one immutable
 *   [JukeboxSnapshot]. Consumers (player coordinator, websocket broadcaster) observe [snapshot].
 */
class MusicQueue {

    private val lock = Any()
    private val _snapshot = MutableStateFlow(JukeboxSnapshot())

    /** Observable, always-consistent view of the whole jukebox. */
    val snapshot: StateFlow<JukeboxSnapshot> = _snapshot.asStateFlow()

    /**
     * Queue a song. If nothing is playing it is promoted straight to now-playing so the
     * coordinator will start it immediately; otherwise it joins the back of the queue.
     */
    fun add(song: Song): Unit = mutate { current ->
        if (current.nowPlaying == null) {
            current.copy(nowPlaying = song, status = PlaybackStatus.BUFFERING)
        } else {
            current.copy(queue = current.queue + song)
        }
    }

    /** Remove a pending song by its queue id. Returns false if it was not found. */
    fun remove(songId: String): Boolean {
        var removed = false
        mutate { current ->
            val filtered = current.queue.filterNot { it.id == songId }
            removed = filtered.size != current.queue.size
            if (removed) current.copy(queue = filtered) else current
        }
        return removed
    }

    /** Move a pending song to [targetIndex] (clamped to bounds). No-op if not found. */
    fun reorder(songId: String, targetIndex: Int): Boolean {
        var moved = false
        mutate { current ->
            val from = current.queue.indexOfFirst { it.id == songId }
            if (from < 0) return@mutate current
            val mutableQueue = current.queue.toMutableList()
            val item = mutableQueue.removeAt(from)
            val clamped = targetIndex.coerceIn(0, mutableQueue.size)
            mutableQueue.add(clamped, item)
            moved = true
            current.copy(queue = mutableQueue)
        }
        return moved
    }

    /** Find who queued a song (for "remove your own" permission checks in the route layer). */
    fun ownerOf(songId: String): String? = snapshot.value.queue.firstOrNull { it.id == songId }?.addedBy

    /** Advance to the next track. Used both on natural song-end and explicit skip. */
    fun advance(): Unit = mutate { current ->
        val next = current.queue.firstOrNull()
        if (next == null) {
            current.copy(nowPlaying = null, status = PlaybackStatus.IDLE, positionSeconds = 0f, durationSeconds = 0f)
        } else {
            current.copy(
                nowPlaying = next,
                queue = current.queue.drop(1),
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
        }
    }

    fun clear(): Unit = mutate { current ->
        current.copy(nowPlaying = null, queue = emptyList(), status = PlaybackStatus.IDLE, positionSeconds = 0f)
    }

    // --- Playback metadata, pushed in from the player coordinator ---

    fun onStatusChanged(status: PlaybackStatus): Unit = mutate { it.copy(status = status) }

    fun onProgress(positionSeconds: Float, durationSeconds: Float): Unit = mutate {
        it.copy(positionSeconds = positionSeconds, durationSeconds = durationSeconds)
    }

    fun setVolume(volume: Int): Unit = mutate { it.copy(volume = volume.coerceIn(0, 100)) }

    private inline fun mutate(transform: (JukeboxSnapshot) -> JukeboxSnapshot) {
        synchronized(lock) { _snapshot.value = transform(_snapshot.value) }
    }
}
