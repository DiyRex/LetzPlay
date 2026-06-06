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

@Serializable
data class ReorderRequest(val songId: String, val targetIndex: Int)

@Serializable
data class VolumeRequest(val volume: Int)

@Serializable
data class ErrorResponse(val error: String)

/** One present remote, shown in the web "who's here" panel. */
@Serializable
data class ConnectedUser(val username: String, val role: UserRole)

/**
 * The websocket payload: jukebox snapshot plus live presence. Presence is a transport concern,
 * so it is merged in here at broadcast time rather than stored in the queue. This matches the
 * Go server's `LiveState` byte-for-byte so one React remote drives both backends.
 */
@Serializable
data class LiveState(val snapshot: JukeboxSnapshot, val users: List<ConnectedUser>)
