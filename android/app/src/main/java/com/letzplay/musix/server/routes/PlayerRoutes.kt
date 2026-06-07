package com.letzplay.musix.server.routes

import com.letzplay.musix.domain.player.PlaybackController
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.AutoplayRequest
import com.letzplay.musix.server.dto.EqRequest
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.dto.FairQueueRequest
import com.letzplay.musix.server.dto.NormalizeRequest
import com.letzplay.musix.server.dto.RepeatRequest
import com.letzplay.musix.server.dto.SeekRequest
import com.letzplay.musix.server.dto.ShuffleRequest
import com.letzplay.musix.server.dto.SleepRequest
import com.letzplay.musix.server.dto.SpeedRequest
import com.letzplay.musix.server.dto.VolumeRequest
import com.letzplay.musix.server.ws.QueueBroadcaster
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.call
import io.ktor.server.auth.principal
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.post
import io.ktor.server.routing.route

/**
 * `/api/player` — transport, volume, vote-skip, sleep, and radio. Available to any logged-in user
 * (shared "party remote" model); the route group is already behind session auth. Role only gates
 * managing other people's songs (see [queueRoutes]).
 *
 * @param onSleep schedules (minutes>0) or cancels (0) the server-side auto-pause.
 */
fun Route.playerRoutes(
    queue: MusicQueue,
    player: PlaybackController,
    broadcaster: QueueBroadcaster,
    onSleep: (minutes: Int) -> Unit,
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

    post("/voteskip") {
        val user = call.principal<UserSession>()!!
        val current = queue.snapshot.value.current
            ?: return@post call.respond(HttpStatusCode.Conflict, ErrorResponse("Nothing is playing"))
        val (_, _, reached) = broadcaster.voteSkip(current.videoId, user.username)
        if (reached) {
            queue.advance()
            broadcaster.resetVotes()
        }
        call.respond(HttpStatusCode.OK)
    }

    post("/sleep") {
        onSleep(call.receive<SleepRequest>().minutes)
        call.respond(HttpStatusCode.OK)
    }

    post("/autoplay") {
        queue.setAutoplay(call.receive<AutoplayRequest>().autoplay)
        call.respond(HttpStatusCode.OK)
    }

    // Audio-shaping toggles. On the IFrame player these only affect the shared UI state (the player
    // can't apply EQ/normalization/speed); the desktop server applies them for real.
    post("/normalize") {
        queue.setNormalize(call.receive<NormalizeRequest>().normalize)
        call.respond(HttpStatusCode.OK)
    }

    post("/eq") {
        queue.setEqualizer(call.receive<EqRequest>().eq)
        call.respond(HttpStatusCode.OK)
    }

    post("/speed") {
        queue.setSpeed(call.receive<SpeedRequest>().speed)
        call.respond(HttpStatusCode.OK)
    }

    post("/fairqueue") {
        queue.setFairQueue(call.receive<FairQueueRequest>().fairQueue)
        call.respond(HttpStatusCode.OK)
    }
}
