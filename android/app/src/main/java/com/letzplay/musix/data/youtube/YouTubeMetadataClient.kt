package com.letzplay.musix.data.youtube

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.get
import io.ktor.client.request.parameter
import io.ktor.client.statement.HttpResponse
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** Title + thumbnail for a video, resolved from YouTube's keyless oEmbed endpoint. */
data class VideoMetadata(val title: String, val thumbnailUrl: String?)

/**
 * Resolves human-readable titles via https://www.youtube.com/oembed — no API key, no quota.
 * Network failures degrade gracefully to a null result so the caller can fall back to the id.
 */
class YouTubeMetadataClient(
    private val client: HttpClient = defaultClient(),
) {
    suspend fun fetch(videoId: String): VideoMetadata? = runCatching {
        val response: HttpResponse = client.get(OEMBED_URL) {
            parameter("url", "https://www.youtube.com/watch?v=$videoId")
            parameter("format", "json")
        }
        if (!response.status.isSuccess()) return null
        val body: OEmbedResponse = response.body()
        VideoMetadata(title = body.title, thumbnailUrl = body.thumbnailUrl)
    }.getOrNull()

    @Serializable
    private data class OEmbedResponse(
        val title: String,
        @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
    )

    companion object {
        private const val OEMBED_URL = "https://www.youtube.com/oembed"

        fun defaultClient(): HttpClient = HttpClient {
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true })
            }
        }
    }
}
