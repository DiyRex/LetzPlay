package com.letzplay.musix.data.lyrics

import com.letzplay.musix.server.dto.Lyrics
import com.letzplay.musix.server.dto.LyricsLine
import io.ktor.client.HttpClient
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.parameter
import io.ktor.client.statement.bodyAsText
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.util.concurrent.ConcurrentHashMap

/**
 * Fetches time-synced lyrics from lrclib.net (free, keyless), mirroring the desktop server's
 * lyrics client. Results are cached by title; failures degrade to Found=false.
 */
class LyricsClient(
    private val client: HttpClient = defaultClient(),
) {
    private val cache = ConcurrentHashMap<String, Lyrics>()
    private val parser = Json { ignoreUnknownKeys = true }

    suspend fun get(title: String): Lyrics {
        val key = title.trim()
        if (key.isEmpty()) return Lyrics()
        cache[key]?.let { return it }
        val result = runCatching { fetch(key) }.getOrDefault(Lyrics())
        cache[key] = result
        return result
    }

    /**
     * Tries several lrclib queries (structured artist/track, broad keyword, track-only, and a
     * latin-only variant for mixed-script titles) and returns the first synced result, falling
     * back to any plain lyrics. The extra attempts greatly improve non-English coverage.
     */
    private suspend fun fetch(title: String): Lyrics {
        val cleaned = cleanTitle(title)
        val (artist, track) = splitArtistTrack(cleaned)
        val attempts = mutableListOf<Map<String, String>>()
        if (artist.isNotEmpty() && track.isNotEmpty()) {
            attempts.add(mapOf("track_name" to track, "artist_name" to artist))
            attempts.add(mapOf("track_name" to artist, "artist_name" to track))
        }
        attempts.add(mapOf("q" to cleaned))
        if (track.isNotEmpty() && track != cleaned) attempts.add(mapOf("q" to track))
        asciiOnly(cleaned).takeIf { it.length >= 3 && it != cleaned }?.let { attempts.add(mapOf("q" to it)) }

        var fallbackPlain = ""
        for (params in attempts.distinct()) {
            for (hit in search(params)) {
                val lines = parseLrc(hit.jsonObject["syncedLyrics"]?.jsonPrimitive?.contentOrNull())
                if (lines.isNotEmpty()) {
                    val plain = hit.jsonObject["plainLyrics"]?.jsonPrimitive?.contentOrNull() ?: ""
                    return Lyrics(found = true, synced = lines, plain = plain)
                }
                if (fallbackPlain.isEmpty()) {
                    hit.jsonObject["plainLyrics"]?.jsonPrimitive?.contentOrNull()
                        ?.takeIf { it.isNotBlank() }?.let { fallbackPlain = it }
                }
            }
        }
        return if (fallbackPlain.isNotEmpty()) Lyrics(found = true, plain = fallbackPlain) else Lyrics()
    }

    private suspend fun search(params: Map<String, String>) = runCatching {
        val response = client.get(SEARCH_URL) {
            params.forEach { (k, v) -> parameter(k, v) }
            header("User-Agent", "LetzPlayMusix (https://github.com/DiyRex/LetzPlay)")
        }
        if (!response.status.isSuccess()) emptyList()
        else parser.parseToJsonElement(response.bodyAsText()).jsonArray.toList()
    }.getOrDefault(emptyList())

    private fun splitArtistTrack(s: String): Pair<String, String> {
        for (sep in listOf(" - ", " – ", " — ")) {
            val i = s.indexOf(sep)
            if (i > 0) return s.substring(0, i).trim() to s.substring(i + sep.length).trim()
        }
        return "" to ""
    }

    private fun asciiOnly(s: String): String =
        s.map { if (it.code < 128 && (it.isLetterOrDigit() || it == ' ' || it == '-' || it == '\'')) it else ' ' }
            .joinToString("")
            .replace(Regex("\\s+"), " ")
            .trim()

    private fun parseLrc(lrc: String?): List<LyricsLine> {
        if (lrc.isNullOrBlank()) return emptyList()
        val lines = mutableListOf<LyricsLine>()
        for (raw in lrc.split("\n")) {
            val matches = LRC_LINE.findAll(raw).toList()
            if (matches.isEmpty()) continue
            val text = raw.substring(matches.last().range.last + 1).trim()
            for (m in matches) {
                val min = m.groupValues[1].toIntOrNull() ?: 0
                val sec = m.groupValues[2].toIntOrNull() ?: 0
                val fracStr = m.groupValues[3].padEnd(3, '0').take(3)
                val ms = if (m.groupValues[3].isNotEmpty()) fracStr.toIntOrNull() ?: 0 else 0
                lines.add(LyricsLine(timeMs = (min * 60 + sec) * 1000 + ms, text = text))
            }
        }
        return lines
    }

    private fun cleanTitle(title: String): String {
        var s = title
        val pipe = s.indexOf('|')
        if (pipe > 0) s = s.substring(0, pipe)
        s = BRACKET_NOISE.replace(s, "")
        s = TRAILING_NOISE.replace(s, "")
        return s.trim()
    }

    private companion object {
        const val SEARCH_URL = "https://lrclib.net/api/search"
        val LRC_LINE = Regex("""\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?]""")
        val BRACKET_NOISE =
            Regex("""(?i)\s*[(\[][^)\]]*(official|video|audio|lyrics?|remaster|4k|hd|mv|cover|visualizer|full song)[^)\]]*[)\]]""")
        val TRAILING_NOISE =
            Regex("""(?i)\s*[-–—|]\s*(official\s+\w+|lyric\s+video|music\s+video|full\s+song|audio|visualizer|hd|4k)\s*$""")

        fun defaultClient(): HttpClient = HttpClient {
            install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
        }
    }
}

private fun kotlinx.serialization.json.JsonPrimitive.contentOrNull(): String? =
    if (isString) content else null
