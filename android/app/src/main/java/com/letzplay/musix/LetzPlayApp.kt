package com.letzplay.musix

import android.app.Application
import com.letzplay.musix.di.ServiceLocator

/** Process entry point — initializes the dependency graph exactly once. */
class LetzPlayApp : Application() {
    override fun onCreate() {
        super.onCreate()
        ServiceLocator.init(this)
    }
}
