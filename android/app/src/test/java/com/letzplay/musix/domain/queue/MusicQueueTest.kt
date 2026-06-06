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

    private fun ids(queue: MusicQueue) = queue.snapshot.value.tracks.map { it.id }

    @Test
    fun `first added song starts playing`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        val snap = queue.snapshot.value
        assertEquals(0, snap.currentIndex)
        assertEquals("a", snap.current?.id)
        assertEquals(PlaybackStatus.BUFFERING, snap.status)
    }

    @Test
    fun `songs are not consumed - full list persists through advance`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        queue.advance()
        assertEquals(listOf("a", "b"), ids(queue)) // both still present
        assertEquals("b", queue.snapshot.value.current?.id)
    }

    @Test
    fun `advance past end goes idle but keeps the list`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.advance()
        assertEquals(PlaybackStatus.IDLE, queue.snapshot.value.status)
        assertEquals(listOf("a"), ids(queue))
        assertEquals(0, queue.snapshot.value.currentIndex)
    }

    @Test
    fun `previous and playNow move the cursor without changing the list`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        queue.add(song(id = "c"))
        queue.advance() // b
        queue.advance() // c
        assertTrue(queue.previous())
        assertEquals("b", queue.snapshot.value.current?.id)
        assertTrue(queue.playNow("a"))
        assertEquals("a", queue.snapshot.value.current?.id)
        assertFalse(queue.playNow("missing"))
        assertEquals(listOf("a", "b", "c"), ids(queue))
    }

    @Test
    fun `remove adjusts the cursor`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        queue.add(song(id = "c"))
        queue.advance() // cursor on b

        assertTrue(queue.remove("a")) // before cursor -> shifts left
        assertEquals(listOf("b", "c"), ids(queue))
        assertEquals("b", queue.snapshot.value.current?.id)

        assertTrue(queue.remove("b")) // remove current -> lands on c
        assertEquals("c", queue.snapshot.value.current?.id)
    }

    @Test
    fun `reorder keeps the cursor on the same song`() {
        val queue = MusicQueue()
        queue.add(song(id = "a"))
        queue.add(song(id = "b"))
        queue.add(song(id = "c"))
        queue.advance() // cursor on b
        queue.reorder("c", 0)
        assertEquals(listOf("c", "a", "b"), ids(queue))
        assertEquals("b", queue.snapshot.value.current?.id)
    }

    @Test
    fun `ownerOf resolves who added a song`() {
        val queue = MusicQueue()
        queue.add(song(id = "a", by = "alice"))
        queue.add(song(id = "b", by = "bob"))
        assertEquals("bob", queue.ownerOf("b"))
        assertNull(queue.ownerOf("missing"))
    }
}
