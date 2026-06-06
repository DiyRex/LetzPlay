package com.letzplay.musix.server.ws

import com.letzplay.musix.domain.model.JukeboxSnapshot
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.ConnectedUser
import com.letzplay.musix.server.dto.LiveState
import io.ktor.websocket.DefaultWebSocketSession
import io.ktor.websocket.Frame
import io.ktor.websocket.close
import io.ktor.websocket.send
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.flow.onEach
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import java.util.concurrent.ConcurrentHashMap

/**
 * Pushes live state to all connected web remotes and tracks presence (who's connected).
 *
 * The wire payload is [LiveState] — snapshot + deduplicated user list — so every phone sees the
 * same queue *and* the same "who's here" list. Presence is owned here (a transport concern), not
 * in the queue, which stays a pure state machine. Broadcasts fire on queue changes and on
 * connect/disconnect.
 */
class QueueBroadcaster(
    private val queue: MusicQueue,
    scope: CoroutineScope,
    private val json: Json,
) {
    private val sessions = ConcurrentHashMap<DefaultWebSocketSession, ConnectedUser>()

    init {
        queue.snapshot
            .onEach { snapshot -> broadcast(snapshot) }
            .launchIn(scope)
    }

    /** Registers the authenticated user as present, streams updates, and cleans up on disconnect. */
    suspend fun handle(session: DefaultWebSocketSession, user: UserSession) {
        sessions[session] = ConnectedUser(user.username, user.role)
        broadcast(queue.snapshot.value) // announce arrival to everyone
        try {
            session.send(Frame.Text(encode(queue.snapshot.value)))
            for (frame in session.incoming) { /* clients only receive; ignore inbound */ }
        } finally {
            sessions.remove(session)
            broadcast(queue.snapshot.value) // announce departure
        }
    }

    private fun broadcast(snapshot: JukeboxSnapshot) {
        if (sessions.isEmpty()) return
        val payload = encode(snapshot)
        sessions.keys.forEach { session ->
            session.launch {
                runCatching { session.send(Frame.Text(payload)) }
                    .onFailure { sessions.remove(session); runCatching { session.close() } }
            }
        }
    }

    /** Deduplicates presence by username (admin wins) and serializes the full [LiveState]. */
    private fun encode(snapshot: JukeboxSnapshot): String {
        val users = sessions.values
            .groupBy { it.username }
            .map { (name, entries) ->
                ConnectedUser(name, entries.firstOrNull { it.role.isAdmin }?.role ?: entries.first().role)
            }
            .sortedBy { it.username }
        return json.encodeToString(LiveState.serializer(), LiveState(snapshot, users))
    }
}
