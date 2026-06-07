package com.letzplay.musix.server.routes

import com.letzplay.musix.data.youtube.YouTubeMetadataClient
import com.letzplay.musix.data.youtube.YouTubeUrlParser
import com.letzplay.musix.domain.model.Song
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.AddResult
import com.letzplay.musix.server.dto.AddSongRequest
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.dto.NamedRequest
import com.letzplay.musix.server.dto.PlaylistSong
import com.letzplay.musix.server.playlist.PlaylistStore
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
 * `/api/playlists` — named song collections. Mirrors the desktop server's playlist API so the same
 * web remote works here. Any logged-in user may manage playlists (party model).
 */
fun Route.playlistRoutes(
    store: PlaylistStore,
    queue: MusicQueue,
    metadataClient: YouTubeMetadataClient,
    nowMillis: () -> Long,
    newId: () -> String,
) = route("/api/playlists") {

    get { call.respond(store.list()) }

    post {
        val name = call.receive<NamedRequest>().name.trim()
        if (name.isEmpty()) return@post call.respond(HttpStatusCode.BadRequest, ErrorResponse("A name is required"))
        call.respond(HttpStatusCode.Created, store.create(name, emptyList()))
    }

    post("/save-queue") {
        val name = call.receive<NamedRequest>().name.trim()
        if (name.isEmpty()) return@post call.respond(HttpStatusCode.BadRequest, ErrorResponse("A name is required"))
        val songs = queue.snapshot.value.tracks.map {
            PlaylistSong(videoId = it.videoId, title = it.title, thumbnailUrl = it.thumbnailUrl)
        }
        if (songs.isEmpty()) return@post call.respond(HttpStatusCode.BadRequest, ErrorResponse("The queue is empty"))
        call.respond(HttpStatusCode.Created, store.create(name, songs))
    }

    get("/{id}") {
        val playlist = store.get(call.parameters["id"].orEmpty())
            ?: return@get call.respond(HttpStatusCode.NotFound, ErrorResponse("Playlist not found"))
        call.respond(playlist)
    }

    delete("/{id}") {
        if (!store.delete(call.parameters["id"].orEmpty())) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("Playlist not found"))
        } else {
            call.respond(HttpStatusCode.OK)
        }
    }

    post("/{id}/songs") {
        val id = call.parameters["id"].orEmpty()
        val videoId = YouTubeUrlParser.extractVideoId(call.receive<AddSongRequest>().url)
            ?: return@post call.respond(HttpStatusCode.BadRequest, ErrorResponse("Not a valid YouTube link"))
        val metadata = metadataClient.fetch(videoId)
        val song = PlaylistSong(
            videoId = videoId,
            title = metadata?.title ?: videoId,
            thumbnailUrl = metadata?.thumbnailUrl ?: "https://i.ytimg.com/vi/$videoId/hqdefault.jpg",
        )
        if (!store.addSong(id, song)) {
            return@post call.respond(HttpStatusCode.NotFound, ErrorResponse("Playlist not found"))
        }
        call.respond(store.get(id)!!)
    }

    delete("/{id}/songs/{videoId}") {
        val ok = store.removeSong(call.parameters["id"].orEmpty(), call.parameters["videoId"].orEmpty())
        if (!ok) call.respond(HttpStatusCode.NotFound, ErrorResponse("Playlist not found"))
        else call.respond(HttpStatusCode.OK)
    }

    post("/{id}/enqueue") {
        val user = call.principal<UserSession>()!!
        val playlist = store.get(call.parameters["id"].orEmpty())
            ?: return@post call.respond(HttpStatusCode.NotFound, ErrorResponse("Playlist not found"))
        playlist.songs.forEach { song ->
            queue.add(
                Song(
                    id = newId(),
                    videoId = song.videoId,
                    title = song.title,
                    thumbnailUrl = song.thumbnailUrl,
                    addedBy = user.username,
                    addedAtEpochMs = nowMillis(),
                ),
            )
        }
        call.respond(HttpStatusCode.OK, AddResult(added = playlist.songs.size, song = playlist.songs.firstOrNull()?.let {
            Song(id = newId(), videoId = it.videoId, title = it.title, thumbnailUrl = it.thumbnailUrl, addedBy = user.username, addedAtEpochMs = nowMillis())
        } ?: Song(id = "", videoId = "", title = "", addedBy = user.username, addedAtEpochMs = nowMillis())))
    }
}
