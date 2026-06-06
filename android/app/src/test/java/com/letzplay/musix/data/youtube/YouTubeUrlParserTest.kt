package com.letzplay.musix.data.youtube

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class YouTubeUrlParserTest {

    @Test
    fun `extracts id from standard watch url`() {
        assertEquals("dQw4w9WgXcQ", YouTubeUrlParser.extractVideoId("https://www.youtube.com/watch?v=dQw4w9WgXcQ"))
    }

    @Test
    fun `extracts id from watch url with extra params`() {
        assertEquals(
            "dQw4w9WgXcQ",
            YouTubeUrlParser.extractVideoId("https://youtube.com/watch?list=abc&v=dQw4w9WgXcQ&t=30s"),
        )
    }

    @Test
    fun `extracts id from short youtu_be link`() {
        assertEquals("dQw4w9WgXcQ", YouTubeUrlParser.extractVideoId("https://youtu.be/dQw4w9WgXcQ?si=xyz"))
    }

    @Test
    fun `extracts id from shorts and embed and live`() {
        assertEquals("abcdefghijk", YouTubeUrlParser.extractVideoId("https://www.youtube.com/shorts/abcdefghijk"))
        assertEquals("abcdefghijk", YouTubeUrlParser.extractVideoId("https://www.youtube.com/embed/abcdefghijk"))
        assertEquals("abcdefghijk", YouTubeUrlParser.extractVideoId("https://www.youtube.com/live/abcdefghijk"))
    }

    @Test
    fun `accepts a bare 11 char id`() {
        assertEquals("dQw4w9WgXcQ", YouTubeUrlParser.extractVideoId("dQw4w9WgXcQ"))
    }

    @Test
    fun `rejects non youtube and malformed input`() {
        assertNull(YouTubeUrlParser.extractVideoId("https://example.com/watch?v=tooShort"))
        assertNull(YouTubeUrlParser.extractVideoId(""))
        assertNull(YouTubeUrlParser.extractVideoId("just some text"))
    }
}
