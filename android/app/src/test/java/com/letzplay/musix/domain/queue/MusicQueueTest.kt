package com.letzplay.musix.domain.queue

import com.letzplay.musix.domain.model.PlaybackStatus
import com.letzplay.musix.domain.model.Song
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicQueueTest {

    private var seq = 0
    private fun song(id: String = "id${seq++}", by: String = "alice") =
        Song(id = id, videoId = "vid$id", title = "Song $id", addedBy = by, addedAtEpochMs = 0)

    @Test
    fun `first added song becomes now playing`() {
        val queue = MusicQueue()
        val first = song()
        queue.add(first)
        assertEquals(first, queue.snapshot.value.nowPlaying)
        assertTrue(queue.snapshot.value.queue.isEmpty())
        assertEquals(PlaybackStatus.BUFFERING, queue.snapshot.value.status)
    }

    @Test
    fun `subsequent songs queue behind now playing`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        assertEquals("a", queue.snapshot.value.nowPlaying?.id)
        assertEquals(listOf("b"), queue.snapshot.value.queue.map { it.id })
    }

    @Test
    fun `advance promotes the next song`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        queue.advance()
        assertEquals("b", queue.snapshot.value.nowPlaying?.id)
        assertTrue(queue.snapshot.value.queue.isEmpty())
    }

    @Test
    fun `advance past the end goes idle`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.advance()
        assertNull(queue.snapshot.value.nowPlaying)
        assertEquals(PlaybackStatus.IDLE, queue.snapshot.value.status)
    }

    @Test
    fun `remove and reorder only affect the pending queue`() {
        val queue = MusicQueue()
        queue.add(song(id = "now"))
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        queue.add(song(id = "c"))

        assertTrue(queue.remove("b"))
        assertEquals(listOf("a", "c"), queue.snapshot.value.queue.map { it.id })

        queue.reorder("c", 0)
        assertEquals(listOf("c", "a"), queue.snapshot.value.queue.map { it.id })

        assertFalse(queue.remove("now")) // now-playing is not in the pending queue
    }

    @Test
    fun `ownerOf resolves who queued a pending song`() {
        val queue = MusicQueue()
        queue.add(song(id = "now", by = "alice"))
        queue.add(song(id = "a", by = "bob"))
        assertEquals("bob", queue.ownerOf("a"))
        assertNull(queue.ownerOf("now"))
    }
}
