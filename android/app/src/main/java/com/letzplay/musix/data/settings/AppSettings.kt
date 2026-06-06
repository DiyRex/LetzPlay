package com.letzplay.musix.data.settings

import android.content.Context
import android.content.SharedPreferences
import com.letzplay.musix.server.auth.PasswordHasher

/**
 * Persisted configuration: the server port and the credentials used to log into the remote.
 *
 * Passwords are never stored in plaintext — only a salted PBKDF2 hash (see [PasswordHasher]).
 * On first launch sensible defaults are seeded; the README instructs the owner to change them.
 */
class AppSettings(context: Context) {

    private val prefs: SharedPreferences =
        context.getSharedPreferences("letzplay_settings", Context.MODE_PRIVATE)

    init {
        if (!prefs.contains(KEY_ADMIN_HASH)) {
            setAdminPassword(DEFAULT_ADMIN_PASSWORD)
        }
        if (!prefs.contains(KEY_GUEST_HASH)) {
            setGuestPassword(DEFAULT_GUEST_PASSWORD)
        }
    }

    var serverPort: Int
        get() = prefs.getInt(KEY_PORT, DEFAULT_PORT)
        set(value) = prefs.edit().putInt(KEY_PORT, value).apply()

    /** Whether guests must enter the party password (false = open to anyone on the LAN). */
    var guestPasswordRequired: Boolean
        get() = prefs.getBoolean(KEY_GUEST_REQUIRED, true)
        set(value) = prefs.edit().putBoolean(KEY_GUEST_REQUIRED, value).apply()

    fun setAdminPassword(plain: String) =
        prefs.edit().putString(KEY_ADMIN_HASH, PasswordHasher.hash(plain)).apply()

    fun setGuestPassword(plain: String) =
        prefs.edit().putString(KEY_GUEST_HASH, PasswordHasher.hash(plain)).apply()

    fun verifyAdminPassword(plain: String): Boolean =
        prefs.getString(KEY_ADMIN_HASH, null)?.let { PasswordHasher.verify(plain, it) } ?: false

    fun verifyGuestPassword(plain: String): Boolean =
        prefs.getString(KEY_GUEST_HASH, null)?.let { PasswordHasher.verify(plain, it) } ?: false

    private companion object {
        const val KEY_PORT = "server_port"
        const val KEY_ADMIN_HASH = "admin_hash"
        const val KEY_GUEST_HASH = "guest_hash"
        const val KEY_GUEST_REQUIRED = "guest_required"

        const val DEFAULT_PORT = 8080
        // Documented defaults — the owner is told to change these on first run.
        const val DEFAULT_ADMIN_PASSWORD = "admin"
        const val DEFAULT_GUEST_PASSWORD = "party"
    }
}
