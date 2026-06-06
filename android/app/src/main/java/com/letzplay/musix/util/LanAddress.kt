package com.letzplay.musix.util

import java.net.Inet4Address
import java.net.NetworkInterface

/** Resolves the device's LAN IPv4 address so the TV can show a "connect here" URL. */
object LanAddress {

    /** @return the first site-local IPv4 (e.g. 192.168.x.x), or null if offline. */
    fun current(): String? = runCatching {
        NetworkInterface.getNetworkInterfaces().asSequence()
            .filter { it.isUp && !it.isLoopback }
            .flatMap { it.inetAddresses.asSequence() }
            .filterIsInstance<Inet4Address>()
            .firstOrNull { it.isSiteLocalAddress }
            ?.hostAddress
    }.getOrNull()

    /** Full base URL for the remote, or null if the address can't be determined. */
    fun remoteUrl(port: Int): String? = current()?.let { "http://$it:$port" }
}
