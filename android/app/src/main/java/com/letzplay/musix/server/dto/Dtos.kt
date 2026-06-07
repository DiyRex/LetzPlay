package com.letzplay.musix.server.dto

import com.letzplay.musix.domain.model.JukeboxSnapshot
import com.letzplay.musix.domain.model.UserRole
import kotlinx.serialization.Serializable

/**
 * Wire contracts between the React remote and the server. Kept in one file because they are
 * small, tightly related, and versioned together. Domain models ([com.letzplay.musix.domain])
 * are serialized directly where they already suit the wire; these cover request bodies and
 * responses that have no domain equivalent.
 */

@Serializable
data class LoginRequest(val username: String, val password: String)

@Serializable
data class SessionResponse(val username: String, val role: UserRole)

@Serializable
data class AddSongRequest(val url: String)

/**
 * Response to POST /api/queue. [added] is how many tracks were queued (>1 for a playlist; the
 * Android app adds the single video and returns 1), [song] is the representative track. Matches
 * the Go server's `addResult`.
 */
@Serializable
data class AddResult(val added: Int, val song: com.letzplay.musix.domain.model.Song)

@Serializable
data class ReorderRequest(val songId: String, val targetIndex: Int)

@Serializable
data class VolumeRequest(val volume: Int)

@Serializable
data class SeekRequest(val seconds: Float)

@Serializable
data class ShuffleRequest(val shuffle: Boolean)

@Serializable
data class RepeatRequest(val repeat: com.letzplay.musix.domain.model.RepeatMode)

@Serializable
data class NormalizeRequest(val normalize: Boolean)

@Serializable
data class EqRequest(val eq: String)

@Serializable
data class SpeedRequest(val speed: Float)

@Serializable
data class FairQueueRequest(val fairQueue: Boolean)

@Serializable
data class StatCount(val label: String, val count: Int)

@Serializable
data class Stats(val mostPlayed: List<StatCount> = emptyList(), val topRequesters: List<StatCount> = emptyList())

@Serializable
data class ErrorResponse(val error: String)

// --- Playlists ---

@Serializable
data class PlaylistSong(val videoId: String, val title: String, val thumbnailUrl: String? = null)

@Serializable
data class PlaylistSummary(val id: String, val name: String, val count: Int)

@Serializable
data class Playlist(
    val id: String,
    val name: String,
    val songs: List<PlaylistSong> = emptyList(),
    val updatedAtMs: Long = 0,
)

@Serializable
data class NamedRequest(val name: String)

/** One present remote, shown in the web "who's here" panel. */
@Serializable
data class ConnectedUser(val username: String, val role: UserRole)

/**
 * The websocket payload: jukebox snapshot plus live presence. Presence is a transport concern,
 * so it is merged in here at broadcast time rather than stored in the queue. This matches the
 * Go server's `LiveState` byte-for-byte so one React remote drives both backends.
 */
@Serializable
data class LiveState(
    val snapshot: JukeboxSnapshot,
    val users: List<ConnectedUser>,
    val skipVotes: Int = 0,
    val skipNeeded: Int = 1,
    val sleepAtMs: Long = 0,
)

@Serializable
data class SleepRequest(val minutes: Int)

@Serializable
data class AutoplayRequest(val autoplay: Boolean)

@Serializable
data class LockRequest(val locked: Boolean)

@Serializable
data class PasswordRequest(val admin: String = "", val guest: String = "")

@Serializable
data class SearchResult(val videoId: String, val title: String, val channel: String, val thumbnailUrl: String)

@Serializable
data class LyricsLine(val timeMs: Int, val text: String)

@Serializable
data class Lyrics(val found: Boolean = false, val synced: List<LyricsLine> = emptyList(), val plain: String = "")
