package com.letzplay.musix.server.routes

import com.letzplay.musix.data.lyrics.LyricsClient
import com.letzplay.musix.data.settings.AppSettings
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.dto.Lyrics
import com.letzplay.musix.server.dto.LockRequest
import com.letzplay.musix.server.dto.PasswordRequest
import com.letzplay.musix.server.dto.SearchResult
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.call
import io.ktor.server.auth.principal
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import io.ktor.server.routing.post
import io.ktor.server.routing.route

/**
 * `/api/search` and `/api/lyrics`. Search needs yt-dlp (desktop-only), so on Android it returns an
 * empty list — the web remote then shows "no results". Lyrics work everywhere via lrclib.
 */
fun Route.featureRoutes(queue: MusicQueue, lyricsClient: LyricsClient) {
    get("/api/search") {
        // No yt-dlp on Android → no in-app search; the URL/paste path still works.
        call.respond(emptyList<SearchResult>())
    }

    get("/api/lyrics") {
        val videoId = call.request.queryParameters["videoId"]
        val title = queue.snapshot.value.tracks.firstOrNull { it.videoId == videoId }?.title
            ?: call.request.queryParameters["title"].orEmpty()
        if (title.isBlank()) {
            call.respond(Lyrics(found = false))
            return@get
        }
        call.respond(lyricsClient.get(title))
    }

    // Session stats aren't tracked on Android (desktop-only) — return empty so the web shows
    // "Nothing yet" rather than an error.
    get("/api/stats") {
        call.respond(com.letzplay.musix.server.dto.Stats())
    }
}

/** `/api/admin` — admin-only: lock the queue and change passwords (persisted in AppSettings). */
fun Route.adminRoutes(queue: MusicQueue, settings: AppSettings) = route("/api/admin") {

    post("/lock") {
        if (!call.requireAdmin()) return@post
        queue.setLocked(call.receive<LockRequest>().locked)
        call.respond(HttpStatusCode.OK)
    }

    post("/password") {
        if (!call.requireAdmin()) return@post
        val request = call.receive<PasswordRequest>()
        if (request.admin.isNotBlank()) settings.setAdminPassword(request.admin)
        if (request.guest.isNotBlank()) settings.setGuestPassword(request.guest)
        call.respond(HttpStatusCode.OK)
    }
}

private suspend fun io.ktor.server.application.ApplicationCall.requireAdmin(): Boolean {
    if (principal<UserSession>()?.role?.isAdmin == true) return true
    respond(HttpStatusCode.Forbidden, ErrorResponse("Admin only"))
    return false
}
