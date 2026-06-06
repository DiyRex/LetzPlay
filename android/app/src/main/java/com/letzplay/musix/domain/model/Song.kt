package com.letzplay.musix.domain.model

import kotlinx.serialization.Serializable

/**
 * One item in the queue. Immutable — the queue replaces entries rather than mutating them,
 * which keeps the [com.letzplay.musix.domain.queue.MusicQueue] reasoning simple.
 *
 * @param id          stable identity for queue operations (remove/reorder). Not the video id.
 * @param videoId     the 11-char YouTube video id used by the IFrame player.
 * @param title       human-readable title (from oEmbed); falls back to the videoId until resolved.
 * @param thumbnailUrl optional artwork for the remote UI.
 * @param addedBy     username of whoever queued it — drives "remove your own song" permission.
 * @param addedAtEpochMs insertion time, for stable display ordering of ties.
 */
@Serializable
data class Song(
    val id: String,
    val videoId: String,
    val title: String,
    val thumbnailUrl: String? = null,
    val addedBy: String,
    val addedAtEpochMs: Long,
)
