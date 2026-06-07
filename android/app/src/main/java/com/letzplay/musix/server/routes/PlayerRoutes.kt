package com.letzplay.musix.server.routes

import com.letzplay.musix.domain.player.PlaybackController
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.dto.RepeatRequest
import com.letzplay.musix.server.dto.SeekRequest
import com.letzplay.musix.server.dto.ShuffleRequest
import com.letzplay.musix.server.dto.VolumeRequest
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.call
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.post
import io.ktor.server.routing.route

/**
 * `/api/player` — transport + volume controls. Available to any logged-in user (shared "party
 * remote" model); the route group is already behind session auth. Role only gates managing other
 * people's songs (see [queueRoutes]).
 */
fun Route.playerRoutes(
    queue: MusicQueue,
    player: PlaybackController,
) = route("/api/player") {

    post("/play") {
        player.play()
        call.respond(HttpStatusCode.OK)
    }

    post("/pause") {
        player.pause()
        call.respond(HttpStatusCode.OK)
    }

    post("/skip") {
        queue.advance()
        call.respond(HttpStatusCode.OK)
    }

    post("/previous") {
        queue.previous()
        call.respond(HttpStatusCode.OK)
    }

    post("/volume") {
        val request = call.receive<VolumeRequest>()
        player.setVolume(request.volume)
        queue.setVolume(request.volume)
        call.respond(HttpStatusCode.OK)
    }

    post("/seek") {
        val request = call.receive<SeekRequest>()
        player.seekTo(request.seconds)
        call.respond(HttpStatusCode.OK)
    }

    post("/shuffle") {
        queue.setShuffle(call.receive<ShuffleRequest>().shuffle)
        call.respond(HttpStatusCode.OK)
    }

    post("/repeat") {
        queue.setRepeat(call.receive<RepeatRequest>().repeat)
        call.respond(HttpStatusCode.OK)
    }

    post("/clear") {
        queue.clear()
        call.respond(HttpStatusCode.OK)
    }
}
