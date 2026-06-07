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

    // Skip-vote + sleep-timer state (transport concerns; not in the queue).
    private val voteLock = Any()
    private var voteKey: String = ""
    private val voters = mutableSetOf<String>()

    @Volatile
    private var sleepAtMs: Long = 0

    init {
        queue.snapshot
            .onEach { snapshot -> broadcast(snapshot) }
            .launchIn(scope)
    }

    /** Records a vote to skip [videoId]; votes reset when the track changes. */
    fun voteSkip(videoId: String, username: String): Triple<Int, Int, Boolean> {
        val votes = synchronized(voteLock) {
            if (voteKey != videoId) {
                voteKey = videoId
                voters.clear()
            }
            voters.add(username)
            voters.size
        }
        val needed = skipThreshold(distinctUsers().size)
        broadcast(queue.snapshot.value) // push the new count to every device immediately
        return Triple(votes, needed, votes >= needed)
    }

    fun resetVotes() {
        synchronized(voteLock) { voteKey = ""; voters.clear() }
        broadcast(queue.snapshot.value)
    }

    fun setSleepAt(ms: Long) {
        sleepAtMs = ms
        broadcast(queue.snapshot.value)
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

    private fun distinctUsers(): List<ConnectedUser> =
        sessions.values
            .groupBy { it.username }
            .map { (name, entries) ->
                ConnectedUser(name, entries.firstOrNull { it.role.isAdmin }?.role ?: entries.first().role)
            }
            .sortedBy { it.username }

    /** Serializes the full [LiveState]: presence + skip votes + sleep timer. */
    private fun encode(snapshot: JukeboxSnapshot): String {
        val users = distinctUsers()
        val currentVideoId = snapshot.current?.videoId
        val votes = synchronized(voteLock) { if (voteKey == currentVideoId) voters.size else 0 }
        val state = LiveState(
            snapshot = snapshot,
            users = users,
            skipVotes = votes,
            skipNeeded = skipThreshold(users.size),
            sleepAtMs = sleepAtMs,
        )
        return json.encodeToString(LiveState.serializer(), state)
    }
}

/** A simple majority of connected users (at least 1). */
private fun skipThreshold(users: Int): Int = if (users < 1) 1 else users / 2 + 1
