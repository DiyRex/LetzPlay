package com.letzplay.musix.server.auth

import android.util.Base64
import java.security.SecureRandom
import javax.crypto.SecretKeyFactory
import javax.crypto.spec.PBEKeySpec

/**
 * Salted PBKDF2 password hashing. Stored format is `iterations:saltB64:hashB64`.
 *
 * PBKDF2 with HMAC-SHA256 is part of the Android platform (no extra dependency) and is a
 * deliberate, slow KDF — appropriate for the small number of credentials this app holds.
 */
object PasswordHasher {

    private const val ALGORITHM = "PBKDF2WithHmacSHA256"
    private const val ITERATIONS = 120_000
    private const val KEY_LENGTH_BITS = 256
    private const val SALT_BYTES = 16

    fun hash(plain: String): String {
        val salt = ByteArray(SALT_BYTES).also { SecureRandom().nextBytes(it) }
        val hash = pbkdf2(plain.toCharArray(), salt, ITERATIONS)
        return "$ITERATIONS:${salt.toB64()}:${hash.toB64()}"
    }

    fun verify(plain: String, stored: String): Boolean {
        val parts = stored.split(":")
        if (parts.size != 3) return false
        val iterations = parts[0].toIntOrNull() ?: return false
        val salt = parts[1].fromB64()
        val expected = parts[2].fromB64()
        val actual = pbkdf2(plain.toCharArray(), salt, iterations)
        return constantTimeEquals(expected, actual)
    }

    private fun pbkdf2(password: CharArray, salt: ByteArray, iterations: Int): ByteArray {
        val spec = PBEKeySpec(password, salt, iterations, KEY_LENGTH_BITS)
        return SecretKeyFactory.getInstance(ALGORITHM).generateSecret(spec).encoded
    }

    /** Length-constant comparison to avoid leaking match progress via timing. */
    private fun constantTimeEquals(a: ByteArray, b: ByteArray): Boolean {
        if (a.size != b.size) return false
        var diff = 0
        for (i in a.indices) diff = diff or (a[i].toInt() xor b[i].toInt())
        return diff == 0
    }

    private fun ByteArray.toB64() = Base64.encodeToString(this, Base64.NO_WRAP)
    private fun String.fromB64() = Base64.decode(this, Base64.NO_WRAP)
}
