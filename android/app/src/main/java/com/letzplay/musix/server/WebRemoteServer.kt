package com.letzplay.musix.server

import android.content.res.AssetManager
import com.letzplay.musix.data.settings.AppSettings
import com.letzplay.musix.data.youtube.YouTubeMetadataClient
import com.letzplay.musix.domain.player.PlaybackController
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.server.auth.AuthService
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.routes.authRoutes
import com.letzplay.musix.server.routes.adminRoutes
import com.letzplay.musix.server.routes.featureRoutes
import com.letzplay.musix.server.routes.playerRoutes
import com.letzplay.musix.server.routes.playlistRoutes
import com.letzplay.musix.server.routes.queueRoutes
import com.letzplay.musix.server.routes.staticWebRoutes
import com.letzplay.musix.server.ws.QueueBroadcaster
import io.ktor.serialization.kotlinx.json.json
import io.ktor.server.application.install
import io.ktor.server.auth.Authentication
import io.ktor.server.auth.authenticate
import io.ktor.server.auth.principal
import io.ktor.server.auth.session
import io.ktor.server.cio.CIO
import io.ktor.server.engine.ApplicationEngine
import io.ktor.server.engine.embeddedServer
import io.ktor.server.plugins.contentnegotiation.ContentNegotiation
import io.ktor.server.plugins.statuspages.StatusPages
import io.ktor.server.response.respond
import io.ktor.server.routing.routing
import io.ktor.server.sessions.SessionTransportTransformerMessageAuthentication
import io.ktor.server.sessions.Sessions
import io.ktor.server.sessions.cookie
import io.ktor.server.websocket.WebSockets
import io.ktor.server.websocket.webSocket
import io.ktor.http.HttpStatusCode
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import java.security.SecureRandom
import java.util.UUID

/**
 * Owns the embedded Ktor server lifecycle and wires the dependency graph for the HTTP/WS layer.
 *
 * Construction is explicit dependency injection — every collaborator is passed in, nothing is
 * looked up globally — which keeps the server unit-testable and the wiring readable in one place.
 * Route registration order matters: API and websocket routes are declared *before* the static
 * SPA catch-all so they are never shadowed by the index.html fallback.
 */
class WebRemoteServer(
    private val settings: AppSettings,
    private val authService: AuthService,
    private val queue: MusicQueue,
    private val player: PlaybackController,
    private val assets: AssetManager,
    private val playlistStore: com.letzplay.musix.server.playlist.PlaylistStore,
    private val metadataClient: YouTubeMetadataClient = YouTubeMetadataClient(),
    private val clock: () -> Long = System::currentTimeMillis,
) {
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
    private val scope = CoroutineScope(SupervisorJob())
    private val broadcaster = QueueBroadcaster(queue, scope, json)
    private val lyricsClient = com.letzplay.musix.data.lyrics.LyricsClient()

    // Fresh signing key each launch: sessions don't need to survive restarts, and a per-launch
    // secret means a stolen cookie is useless after the box reboots.
    private val sessionKey = ByteArray(32).also { SecureRandom().nextBytes(it) }

    private var engine: ApplicationEngine? = null
    private var sleepJob: kotlinx.coroutines.Job? = null

    /** Schedules (minutes>0) or cancels (0) the server-side auto-pause; mirrors the desktop sleep timer. */
    private fun scheduleSleep(minutes: Int) {
        sleepJob?.cancel()
        if (minutes <= 0) {
            broadcaster.setSleepAt(0)
            return
        }
        val durationMs = minutes * 60_000L
        broadcaster.setSleepAt(System.currentTimeMillis() + durationMs)
        sleepJob = scope.launch {
            kotlinx.coroutines.delay(durationMs)
            player.pause()
            broadcaster.setSleepAt(0)
        }
    }

    fun start() {
        if (engine != null) return
        engine = embeddedServer(CIO, port = settings.serverPort) {
            install(ContentNegotiation) { json(json) }
            install(WebSockets)
            install(Sessions) {
                cookie<UserSession>(SESSION_COOKIE) {
                    cookie.path = "/"
                    cookie.httpOnly = true
                    transform(SessionTransportTransformerMessageAuthentication(sessionKey))
                }
            }
            install(Authentication) {
                session<UserSession>(AUTH_SESSION) {
                    validate { it }
                    challenge {
                        call.respond(HttpStatusCode.Unauthorized, ErrorResponse("Login required"))
                    }
                }
            }
            install(StatusPages) {
                exception<Throwable> { call, cause ->
                    call.respond(
                        HttpStatusCode.InternalServerError,
                        ErrorResponse(cause.message ?: "Unexpected server error"),
                    )
                }
            }

            routing {
                authRoutes(authService)

                authenticate(AUTH_SESSION) {
                    queueRoutes(
                        queue = queue,
                        metadataClient = metadataClient,
                        nowMillis = clock,
                        newId = { UUID.randomUUID().toString() },
                        maxPerUser = { settings.maxPerUser },
                    )
                    playerRoutes(queue, player, broadcaster) { minutes -> scheduleSleep(minutes) }
                    featureRoutes(queue, lyricsClient)
                    adminRoutes(queue, settings)
                    playlistRoutes(
                        store = playlistStore,
                        queue = queue,
                        metadataClient = metadataClient,
                        nowMillis = clock,
                        newId = { UUID.randomUUID().toString() },
                    )
                    webSocket("/ws") {
                        val user = call.principal<UserSession>()!!
                        broadcaster.handle(this, user)
                    }
                }

                // Catch-all SPA host — must be registered last.
                staticWebRoutes(assets)
            }
        }.also { it.start(wait = false) }
    }

    fun stop() {
        engine?.stop(gracePeriodMillis = 500, timeoutMillis = 1500)
        engine = null
        scope.cancel()
    }

    private companion object {
        const val SESSION_COOKIE = "LETZPLAY_SESSION"
        const val AUTH_SESSION = "auth-session"
    }
}
