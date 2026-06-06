package com.letzplay.musix.domain.model

/**
 * Authorization tiers for the web remote.
 *
 * - [GUEST] can add songs and remove/reorder songs they added themselves.
 * - [ADMIN] can control playback (skip/pause/volume) and manage any song in the queue.
 */
enum class UserRole {
    GUEST,
    ADMIN;

    val isAdmin: Boolean get() = this == ADMIN
}
