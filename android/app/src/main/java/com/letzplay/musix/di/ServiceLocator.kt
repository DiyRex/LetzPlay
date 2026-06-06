package com.letzplay.musix.di

import android.content.Context
import com.letzplay.musix.data.settings.AppSettings
import com.letzplay.musix.data.youtube.YouTubeMetadataClient
import com.letzplay.musix.domain.queue.MusicQueue
import com.letzplay.musix.player.YouTubePlayerController
import com.letzplay.musix.server.auth.AuthService

/**
 * Minimal manual dependency injection. A full DI framework (Hilt/Koin) would be overkill for an
 * app this size; a single locator initialized once from [com.letzplay.musix.LetzPlayApp] gives us
 * shared singletons without the runtime weight or annotation-processing build cost.
 *
 * Everything here is process-scoped and shared between the always-on player screen
 * ([com.letzplay.musix.ui.MainActivity]) and the foreground server
 * ([com.letzplay.musix.service.JukeboxService]) — most importantly the one [MusicQueue], which is
 * the single source of truth both sides read and write.
 */
object ServiceLocator {

    lateinit var settings: AppSettings
        private set
    lateinit var queue: MusicQueue
        private set
    lateinit var authService: AuthService
        private set
    lateinit var playerController: YouTubePlayerController
        private set

    val metadataClient: YouTubeMetadataClient by lazy { YouTubeMetadataClient() }

    fun init(context: Context) {
        if (::settings.isInitialized) return
        settings = AppSettings(context.applicationContext)
        queue = MusicQueue()
        authService = AuthService(settings)
        // Player callbacks are wired straight into the queue: status/progress in, auto-advance on end.
        playerController = YouTubePlayerController(
            onStatusChanged = queue::onStatusChanged,
            onProgress = queue::onProgress,
            onEnded = queue::advance,
        )
    }
}
