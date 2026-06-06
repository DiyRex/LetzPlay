package com.letzplay.musix.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.net.wifi.WifiManager
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import androidx.core.app.NotificationCompat
import com.letzplay.musix.R
import com.letzplay.musix.di.ServiceLocator
import com.letzplay.musix.server.WebRemoteServer
import com.letzplay.musix.util.LanAddress

/**
 * Keeps the jukebox alive while the box is "idle": runs the embedded web server, holds a partial
 * wake lock and a WiFi lock so Android won't suspend networking or the CPU mid-party, and shows a
 * persistent notification (required for a foreground service) with the connect URL.
 *
 * It deliberately owns only process/lifecycle concerns — all jukebox logic lives in the queue,
 * server, and player components it wires together via [ServiceLocator].
 */
class JukeboxService : Service() {

    private var server: WebRemoteServer? = null
    private var wakeLock: PowerManager.WakeLock? = null
    private var wifiLock: WifiManager.WifiLock? = null

    override fun onCreate() {
        super.onCreate()
        acquireLocks()
        startServer()
        startForeground(NOTIFICATION_ID, buildNotification())
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int = START_STICKY

    override fun onDestroy() {
        server?.stop()
        server = null
        releaseLocks()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun startServer() {
        server = WebRemoteServer(
            settings = ServiceLocator.settings,
            authService = ServiceLocator.authService,
            queue = ServiceLocator.queue,
            player = ServiceLocator.playerController,
            assets = assets,
            metadataClient = ServiceLocator.metadataClient,
        ).also { it.start() }
    }

    private fun acquireLocks() {
        val power = getSystemService(Context.POWER_SERVICE) as PowerManager
        wakeLock = power.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "LetzPlay::ServerWakeLock").apply {
            setReferenceCounted(false)
            acquire()
        }
        val wifi = applicationContext.getSystemService(Context.WIFI_SERVICE) as WifiManager
        wifiLock = wifi.createWifiLock(WifiManager.WIFI_MODE_FULL_HIGH_PERF, "LetzPlay::WifiLock").apply {
            setReferenceCounted(false)
            acquire()
        }
    }

    private fun releaseLocks() {
        wakeLock?.takeIf { it.isHeld }?.release()
        wifiLock?.takeIf { it.isHeld }?.release()
        wakeLock = null
        wifiLock = null
    }

    private fun buildNotification(): Notification {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                getString(R.string.notification_channel_name),
                NotificationManager.IMPORTANCE_LOW,
            )
            (getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager)
                .createNotificationChannel(channel)
        }
        val url = LanAddress.remoteUrl(ServiceLocator.settings.serverPort)
            ?: getString(R.string.notification_no_network)
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.notification_title))
            .setContentText(getString(R.string.notification_text, url))
            .setSmallIcon(R.drawable.ic_jukebox)
            .setOngoing(true)
            .build()
    }

    companion object {
        private const val CHANNEL_ID = "letzplay_server"
        private const val NOTIFICATION_ID = 1

        fun start(context: Context) {
            val intent = Intent(context, JukeboxService::class.java)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }
    }
}
