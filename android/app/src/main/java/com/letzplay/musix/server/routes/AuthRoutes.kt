package com.letzplay.musix.server.routes

import com.letzplay.musix.server.auth.AuthService
import com.letzplay.musix.server.auth.UserSession
import com.letzplay.musix.server.dto.ErrorResponse
import com.letzplay.musix.server.dto.LoginRequest
import com.letzplay.musix.server.dto.SessionResponse
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.call
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import io.ktor.server.routing.post
import io.ktor.server.routing.route
import io.ktor.server.sessions.clear
import io.ktor.server.sessions.get
import io.ktor.server.sessions.sessions
import io.ktor.server.sessions.set

/** `/api/auth` — login, logout, and "who am I". */
fun Route.authRoutes(authService: AuthService) = route("/api/auth") {

    post("/login") {
        val request = call.receive<LoginRequest>()
        val role = authService.authenticate(request.password)
        if (role == null) {
            call.respond(HttpStatusCode.Unauthorized, ErrorResponse("Incorrect password"))
            return@post
        }
        val username = request.username.trim().ifBlank { "Guest" }
        call.sessions.set(UserSession(username = username, role = role))
        call.respond(SessionResponse(username, role))
    }

    post("/logout") {
        call.sessions.clear<UserSession>()
        call.respond(HttpStatusCode.OK)
    }

    get("/me") {
        val session = call.sessions.get<UserSession>()
        if (session == null) {
            call.respond(HttpStatusCode.Unauthorized, ErrorResponse("Not logged in"))
        } else {
            call.respond(SessionResponse(session.username, session.role))
        }
    }
}
