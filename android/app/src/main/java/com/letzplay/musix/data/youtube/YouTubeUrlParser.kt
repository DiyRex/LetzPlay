package com.letzplay.musix.data.youtube

/**
 * Extracts the 11-character video id from the many shapes a YouTube link can take:
 * watch URLs, youtu.be short links, /embed, /shorts, /live, or a bare id pasted directly.
 *
 * Pure and side-effect free so it can be exhaustively unit tested.
 */
object YouTubeUrlParser {

    private val VIDEO_ID = Regex("^[A-Za-z0-9_-]{11}$")

    // Ordered patterns; first match wins. Each capture group 1 is the candidate id.
    private val PATTERNS = listOf(
        Regex("""youtu\.be/([A-Za-z0-9_-]{11})"""),
        Regex("""[?&]v=([A-Za-z0-9_-]{11})"""),
        Regex("""/embed/([A-Za-z0-9_-]{11})"""),
        Regex("""/shorts/([A-Za-z0-9_-]{11})"""),
        Regex("""/live/([A-Za-z0-9_-]{11})"""),
    )

    /** @return the video id, or null if [input] contains no recognizable YouTube id. */
    fun extractVideoId(input: String): String? {
        val trimmed = input.trim()
        if (trimmed.isEmpty()) return null
        if (VIDEO_ID.matches(trimmed)) return trimmed
        return PATTERNS.firstNotNullOfOrNull { it.find(trimmed)?.groupValues?.get(1) }
    }
}
