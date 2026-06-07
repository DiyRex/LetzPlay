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

    private suspend fun fetch(title: String): Lyrics {
        val response = client.get(SEARCH_URL) {
            parameter("q", cleanTitle(title))
            header("User-Agent", "LetzPlayMusix (https://github.com/DiyRex/LetzPlay)")
        }
        if (!response.status.isSuccess()) return Lyrics()

        val hits = parser.parseToJsonElement(response.bodyAsText()).jsonArray
        for (hit in hits) {
            val synced = hit.jsonObject["syncedLyrics"]?.jsonPrimitive?.contentOrNull()
            val lines = parseLrc(synced)
            if (lines.isNotEmpty()) {
                val plain = hit.jsonObject["plainLyrics"]?.jsonPrimitive?.contentOrNull() ?: ""
                return Lyrics(found = true, synced = lines, plain = plain)
            }
        }
        for (hit in hits) {
            val plain = hit.jsonObject["plainLyrics"]?.jsonPrimitive?.contentOrNull()
            if (!plain.isNullOrBlank()) return Lyrics(found = true, plain = plain)
        }
        return Lyrics()
    }

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

    private fun cleanTitle(title: String): String = NOISE.replace(title, "").trim()

    private companion object {
        const val SEARCH_URL = "https://lrclib.net/api/search"
        val LRC_LINE = Regex("""\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?]""")
        val NOISE = Regex("""(?i)\s*[(\[][^)\]]*(official|video|audio|lyrics?|remaster|4k|hd|mv)[^)\]]*[)\]]""")

        fun defaultClient(): HttpClient = HttpClient {
            install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
        }
    }
}

private fun kotlinx.serialization.json.JsonPrimitive.contentOrNull(): String? =
    if (isString) content else null
