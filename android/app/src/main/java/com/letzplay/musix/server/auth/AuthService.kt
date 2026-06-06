package com.letzplay.musix.server.auth

import com.letzplay.musix.data.settings.AppSettings
import com.letzplay.musix.domain.model.UserRole

/**
 * Decides a user's role from the password they present. There are no per-user accounts —
 * the password itself selects the tier, which fits a "shared party device" trust model:
 *
 * - admin password  -> [UserRole.ADMIN]
 * - guest password  -> [UserRole.GUEST]
 * - any password when guests are open -> [UserRole.GUEST]
 */
class AuthService(private val settings: AppSettings) {

    fun authenticate(password: String): UserRole? = when {
        settings.verifyAdminPassword(password) -> UserRole.ADMIN
        settings.verifyGuestPassword(password) -> UserRole.GUEST
        !settings.guestPasswordRequired -> UserRole.GUEST
        else -> null
    }
}
