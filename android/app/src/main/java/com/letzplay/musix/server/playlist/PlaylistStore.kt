package com.letzplay.musix.server.playlist

import com.letzplay.musix.server.dto.Playlist
import com.letzplay.musix.server.dto.PlaylistSong
import com.letzplay.musix.server.dto.PlaylistSummary
import java.util.UUID
import java.util.concurrent.ConcurrentHashMap

/**
 * In-memory store of named playlists for the Android server. Mirrors the desktop store's API so
 * the shared web remote works against either backend. (Desktop persists to disk; here playlists
 * live for the app's lifetime — adequate for a TV box that stays running.)
 */
class PlaylistStore(private val clock: () -> Long = System::currentTimeMillis) {

    private val playlists = ConcurrentHashMap<String, Playlist>()

    fun list(): List<PlaylistSummary> =
        playlists.values
            .map { PlaylistSummary(it.id, it.name, it.songs.size) }
            .sortedBy { it.name }

    fun get(id: String): Playlist? = playlists[id]

    fun create(name: String, songs: List<PlaylistSong>): Playlist {
        val playlist = Playlist(UUID.randomUUID().toString(), name, songs, clock())
        playlists[playlist.id] = playlist
        return playlist
    }

    fun delete(id: String): Boolean = playlists.remove(id) != null

    /** Append a song, deduped by videoId. Returns false if the playlist is missing. */
    fun addSong(id: String, song: PlaylistSong): Boolean {
        val current = playlists[id] ?: return false
        if (current.songs.none { it.videoId == song.videoId }) {
            playlists[id] = current.copy(songs = current.songs + song, updatedAtMs = clock())
        }
        return true
    }

    fun removeSong(id: String, videoId: String): Boolean {
        val current = playlists[id] ?: return false
        playlists[id] = current.copy(
            songs = current.songs.filterNot { it.videoId == videoId },
            updatedAtMs = clock(),
        )
        return true
    }
}
