package com.letzplay.musix.domain.queue

import com.letzplay.musix.domain.model.JukeboxSnapshot
import com.letzplay.musix.domain.model.PlaybackStatus
import com.letzplay.musix.domain.model.RepeatMode
import com.letzplay.musix.domain.model.Song
import kotlin.random.Random
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * The single owner of jukebox state, modelled as a persistent playlist with a moving cursor
 * ([JukeboxSnapshot.currentIndex]) rather than a consumed queue.
 *
 * A finished song stays in [JukeboxSnapshot.tracks]; the cursor simply advances. Play/previous/
 * jump are cursor moves; songs leave only via [remove]. This is what keeps the full list visible
 * on every remote.
 *
 * The class is mechanical and pure (no player, network, or Android APIs). All mutations funnel
 * through [mutate] under one lock, then publish an immutable snapshot consumers observe.
 */
class MusicQueue {

    private val lock = Any()
    private val _snapshot = MutableStateFlow(JukeboxSnapshot())
    val snapshot: StateFlow<JukeboxSnapshot> = _snapshot.asStateFlow()

    /** Append a song. If nothing is playing (idle), it becomes the cursor and starts. */
    fun add(song: Song): Unit = mutate { current ->
        val tracks = current.tracks + song
        if (current.status == PlaybackStatus.IDLE || current.currentIndex < 0) {
            current.copy(
                tracks = tracks,
                currentIndex = tracks.lastIndex,
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
        } else {
            current.copy(tracks = tracks)
        }
    }

    /** Remove a song (the only way one leaves the list). Returns false if not found. */
    fun remove(songId: String): Boolean {
        var removed = false
        mutate { current ->
            val idx = current.tracks.indexOfFirst { it.id == songId }
            if (idx < 0) return@mutate current
            removed = true
            val tracks = current.tracks.filterIndexed { i, _ -> i != idx }
            var index = current.currentIndex
            var status = current.status
            when {
                idx < current.currentIndex -> index--
                idx == current.currentIndex -> {
                    if (index >= tracks.size) index = tracks.lastIndex
                    status = if (index < 0) PlaybackStatus.IDLE else PlaybackStatus.BUFFERING
                }
            }
            current.copy(tracks = tracks, currentIndex = index, status = status, positionSeconds = 0f)
        }
        return removed
    }

    /** Move a song to [targetIndex], keeping the cursor on the same playing track. */
    fun reorder(songId: String, targetIndex: Int): Boolean {
        var moved = false
        mutate { current ->
            val from = current.tracks.indexOfFirst { it.id == songId }
            if (from < 0) return@mutate current
            val currentId = current.current?.id
            val mutable = current.tracks.toMutableList()
            val item = mutable.removeAt(from)
            mutable.add(targetIndex.coerceIn(0, mutable.size), item)
            moved = true
            current.copy(
                tracks = mutable,
                currentIndex = currentId?.let { id -> mutable.indexOfFirst { it.id == id } } ?: current.currentIndex,
            )
        }
        return moved
    }

    /** Who added a song, or null if it isn't in the list (for permission checks in routes). */
    fun ownerOf(songId: String): String? = snapshot.value.tracks.firstOrNull { it.id == songId }?.addedBy

    /**
     * Advance the cursor. Respects shuffle (random pick) and repeat-all (wrap). Repeat-one is
     * handled by the player looping the file, so advance isn't called then. Nothing is removed.
     */
    fun advance(): Unit = mutate { current ->
        val n = current.tracks.size
        when {
            n == 0 -> current.copy(currentIndex = -1, status = PlaybackStatus.IDLE, positionSeconds = 0f, durationSeconds = 0f)
            current.shuffle && n > 1 -> current.copy(
                currentIndex = randomOtherIndex(n, current.currentIndex),
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
            current.currentIndex + 1 < n -> current.copy(
                currentIndex = current.currentIndex + 1,
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
            current.repeat == RepeatMode.ALL -> current.copy(
                currentIndex = 0,
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
            else -> current.copy(status = PlaybackStatus.IDLE, positionSeconds = 0f, durationSeconds = 0f)
        }
    }

    fun setShuffle(on: Boolean): Unit = mutate { it.copy(shuffle = on) }

    fun setRepeat(mode: RepeatMode): Unit = mutate { it.copy(repeat = mode) }

    private fun randomOtherIndex(n: Int, current: Int): Int {
        if (n <= 1) return 0
        val i = Random.nextInt(n - 1)
        return if (i >= current) i + 1 else i
    }

    /** Move the cursor back one track. Returns false when already at the start. */
    fun previous(): Boolean {
        var moved = false
        mutate { current ->
            if (current.currentIndex <= 0) return@mutate current
            moved = true
            current.copy(
                currentIndex = current.currentIndex - 1,
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
        }
        return moved
    }

    /** Jump the cursor straight to a song (tap-to-play). Returns false if not found. */
    fun playNow(songId: String): Boolean {
        var moved = false
        mutate { current ->
            val idx = current.tracks.indexOfFirst { it.id == songId }
            if (idx < 0) return@mutate current
            moved = true
            current.copy(
                currentIndex = idx,
                status = PlaybackStatus.BUFFERING,
                positionSeconds = 0f,
                durationSeconds = 0f,
            )
        }
        return moved
    }

    fun clear(): Unit = mutate {
        JukeboxSnapshot(volume = it.volume, shuffle = it.shuffle, repeat = it.repeat)
    }

    // --- playback metadata, pushed in from the player coordinator ---

    fun onStatusChanged(status: PlaybackStatus): Unit = mutate { it.copy(status = status) }

    fun onProgress(positionSeconds: Float, durationSeconds: Float): Unit = mutate {
        it.copy(positionSeconds = positionSeconds, durationSeconds = durationSeconds)
    }

    fun setVolume(volume: Int): Unit = mutate { it.copy(volume = volume.coerceIn(0, 100)) }

    private inline fun mutate(transform: (JukeboxSnapshot) -> JukeboxSnapshot) {
        synchronized(lock) { _snapshot.value = transform(_snapshot.value) }
    }
}
