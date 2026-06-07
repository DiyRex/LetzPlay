package com.letzplay.musix.server.routes

import com.letzplay.musix.data.youtube.YouTubeMetadataClient
import com.letzplay.musix.data.youtube.YouTubeUrlParser
import com.letzplay.musix.domain.model.Song
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.AddResult
import com.letzplay.musix.server.dto.AddSongRequest
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.dto.ReorderRequest
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.call
import io.ktor.server.auth.principal
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.delete
import io.ktor.server.routing.get
import io.ktor.server.routing.post
import io.ktor.server.routing.route

/**
 * `/api/queue` — read the queue and mutate it.
 *
 * Authorization rule for removal/reorder: admins may touch anything; guests may only touch
 * songs they themselves added. Song ids are generated server-side to avoid client collisions.
 */
fun Route.queueRoutes(
    queue: MusicQueue,
    metadataClient: YouTubeMetadataClient,
    nowMillis: () -> Long,
    newId: () -> String,
    maxPerUser: () -> Int = { 0 },
) = route("/api/queue") {

    get {
        call.respond(queue.snapshot.value)
    }

    post {
        val user = call.principal<UserSession>()!!
        // Admin queue lock and per-user request limits apply to guests only.
        if (!user.role.isAdmin) {
            if (queue.snapshot.value.locked) {
                call.respond(HttpStatusCode.Forbidden, ErrorResponse("The host has locked the queue"))
                return@post
            }
            val limit = maxPerUser()
            if (limit > 0 && queue.countByUser(user.username) >= limit) {
                call.respond(
                    HttpStatusCode.TooManyRequests,
                    ErrorResponse("You've reached your limit of $limit queued songs"),
                )
                return@post
            }
        }
        val request = call.receive<AddSongRequest>()
        val videoId = YouTubeUrlParser.extractVideoId(request.url)
        if (videoId == null) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("Not a valid YouTube link"))
            return@post
        }
        val metadata = metadataClient.fetch(videoId)
        val song = Song(
            id = newId(),
            videoId = videoId,
            title = metadata?.title ?: videoId,
            thumbnailUrl = metadata?.thumbnailUrl,
            addedBy = user.username,
            addedAtEpochMs = nowMillis(),
        )
        queue.add(song)
        // Android adds a single video (playlist expansion needs yt-dlp, desktop-only) → added = 1.
        call.respond(HttpStatusCode.Created, AddResult(added = 1, song = song))
    }

    delete("/{id}") {
        val user = call.principal<UserSession>()!!
        val id = call.parameters["id"] ?: return@delete call.respond(HttpStatusCode.BadRequest)
        val owner = queue.ownerOf(id)
        if (owner == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("Song not in queue"))
            return@delete
        }
        if (!user.role.isAdmin && owner != user.username) {
            call.respond(HttpStatusCode.Forbidden, ErrorResponse("You can only remove songs you added"))
            return@delete
        }
        queue.remove(id)
        call.respond(HttpStatusCode.OK)
    }

    post("/reorder") {
        val user = call.principal<UserSession>()!!
        val request = call.receive<ReorderRequest>()
        val owner = queue.ownerOf(request.songId)
        if (owner == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("Song not in queue"))
            return@post
        }
        if (!user.role.isAdmin && owner != user.username) {
            call.respond(HttpStatusCode.Forbidden, ErrorResponse("You can only reorder songs you added"))
            return@post
        }
        queue.reorder(request.songId, request.targetIndex)
        call.respond(HttpStatusCode.OK)
    }

    // Radio-from-song needs yt-dlp (desktop-only); report gracefully on Android.
    post("/{id}/radio") {
        call.respond(HttpStatusCode.BadGateway, ErrorResponse("Radio isn't available on this server"))
    }

    // Tap-to-play: jump the cursor straight to a song. Any logged-in user may do this.
    post("/{id}/play") {
        val id = call.parameters["id"] ?: return@post call.respond(HttpStatusCode.BadRequest)
        if (!queue.playNow(id)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("Song not in the list"))
            return@post
        }
        call.respond(HttpStatusCode.OK)
    }
}
