package com.letzplay.musix.ui

import android.app.Application
import android.graphics.Bitmap
import androidx.lifecycle.AndroidViewModel
import com.letzplay.musix.di.ServiceLocator
import com.letzplay.musix.domain.model.JukeboxSnapshot
import com.letzplay.musix.util.LanAddress
import com.letzplay.musix.util.QrCodes
import kotlinx.coroutines.flow.StateFlow

/**
 * Surfaces exactly what the TV screen needs: the live jukebox state and the static "connect"
 * details (URL + QR). Keeping this here keeps [MainActivity] focused on view binding.
 */
class MainViewModel(application: Application) : AndroidViewModel(application) {

    val snapshot: StateFlow<JukeboxSnapshot> = ServiceLocator.queue.snapshot

    val remoteUrl: String? = LanAddress.remoteUrl(ServiceLocator.settings.serverPort)

    val connectQr: Bitmap? = remoteUrl?.let { QrCodes.encode(it) }
}
