package com.letzplay.musix.server.auth

import com.letzplay.musix.domain.model.UserRole
import io.ktor.server.auth.Principal
import kotlinx.serialization.Serializable

/**
 * Server-signed session cookie payload. Ktor signs this with a per-launch secret, so a client
 * cannot forge or tamper with their role — the role is decided server-side at login time.
 *
 * Doubles as the authenticated [Principal] so protected routes can read the user via
 * `call.principal<UserSession>()`.
 */
@Serializable
data class UserSession(
    val username: String,
    val role: UserRole,
) : Principal
