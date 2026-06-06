package com.letzplay.musix.ui

import android.os.Bundle
import android.view.WindowManager
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.letzplay.musix.databinding.ActivityMainBinding
import com.letzplay.musix.di.ServiceLocator
import com.letzplay.musix.domain.model.JukeboxSnapshot
import com.letzplay.musix.player.PlaybackCoordinator
import com.letzplay.musix.service.JukeboxService
import com.pierfrancescosoffritti.androidyoutubeplayer.core.player.options.IFramePlayerOptions
import kotlinx.coroutines.launch
import kotlin.math.roundToInt

/**
 * The always-on "now playing" screen and sole video renderer for the TV box.
 *
 * Responsibilities are kept thin: host the player view, render the latest [JukeboxSnapshot] the
 * [MainViewModel] exposes, and ensure the server service is running. All decisions about *what*
 * plays live in the queue/coordinator, not here.
 */
class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private val viewModel: MainViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        // A jukebox screen must never sleep mid-song.
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)

        setupPlayer()
        JukeboxService.start(this)
        renderConnectPanel()
        observeJukebox()
    }

    private fun setupPlayer() {
        val playerView = binding.youtubePlayer
        // We initialize manually (see enableAutomaticInitialization=false in the layout) so the
        // player reports into our controller's listener rather than a throwaway default one.
        lifecycle.addObserver(playerView)
        val options = IFramePlayerOptions.Builder().controls(0).rel(0).build()
        playerView.initialize(ServiceLocator.playerController.listener, options)

        // Drive playback from queue changes for the lifetime of this screen.
        PlaybackCoordinator(ServiceLocator.queue, ServiceLocator.playerController, lifecycleScope)
    }

    private fun renderConnectPanel() {
        binding.connectUrl.text = viewModel.remoteUrl ?: getString(
            com.letzplay.musix.R.string.notification_no_network,
        )
        viewModel.connectQr?.let(binding.connectQr::setImageBitmap)
    }

    private fun observeJukebox() {
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.snapshot.collect(::render)
            }
        }
    }

    private fun render(snapshot: JukeboxSnapshot) {
        val playing = snapshot.current
        // When nothing is playing, surface the "scan to connect" panel; otherwise show now-playing.
        binding.connectPanel.visibility = if (playing == null) android.view.View.VISIBLE else android.view.View.GONE
        binding.nowPlayingBar.visibility = if (playing == null) android.view.View.GONE else android.view.View.VISIBLE

        if (playing != null) {
            val upNext = (snapshot.tracks.size - snapshot.currentIndex - 1).coerceAtLeast(0)
            binding.nowPlayingTitle.text = playing.title
            binding.nowPlayingMeta.text = resources.getQuantityString(
                com.letzplay.musix.R.plurals.songs_in_queue,
                upNext,
                upNext,
            )
            binding.progressBar.max = snapshot.durationSeconds.roundToInt().coerceAtLeast(1)
            binding.progressBar.progress = snapshot.positionSeconds.roundToInt()
        }
    }
}
