package com.letzplay.musix.server.routes

import com.letzplay.musix.domain.player.PlaybackController
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.dto.VolumeRequest
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.ApplicationCall
import io.ktor.server.application.call
import io.ktor.server.auth.principal
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.post
import io.ktor.server.routing.route

/**
 * `/api/player` — transport controls. Admin-only: in a party setting you don't want any guest
 * pausing the music. Guests influence playback only through the queue.
 */
fun Route.playerRoutes(
    queue: MusicQueue,
    player: PlaybackController,
) = route("/api/player") {

    post("/play") {
        if (!call.requireAdmin()) return@post
        player.play()
        call.respond(HttpStatusCode.OK)
    }

    post("/pause") {
        if (!call.requireAdmin()) return@post
        player.pause()
        call.respond(HttpStatusCode.OK)
    }

    post("/skip") {
        if (!call.requireAdmin()) return@post
        queue.advance()
        call.respond(HttpStatusCode.OK)
    }

    post("/volume") {
        if (!call.requireAdmin()) return@post
        val request = call.receive<VolumeRequest>()
        player.setVolume(request.volume)
        queue.setVolume(request.volume)
        call.respond(HttpStatusCode.OK)
    }
}

/** Responds 403 and returns false if the caller is not an admin. */
private suspend fun ApplicationCall.requireAdmin(): Boolean {
    val user = principal<UserSession>()
    if (user?.role?.isAdmin == true) return true
    respond(HttpStatusCode.Forbidden, ErrorResponse("Admin only"))
    return false
}
