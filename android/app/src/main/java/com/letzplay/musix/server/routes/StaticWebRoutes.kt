package com.letzplay.musix.server.routes

import android.content.res.AssetManager
import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.call
import io.ktor.server.response.respondBytes
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import java.io.IOException

/**
 * Serves the compiled React remote (Vite output bundled into `assets/web/`) as a single-page app.
 *
 * The SPA fallback is the key behaviour: any path that isn't a real asset returns `index.html`,
 * so client-side routes like `/remote` survive a hard refresh. API and websocket paths are
 * registered before this catch-all, so they always win.
 */
fun Route.staticWebRoutes(assets: AssetManager, basePath: String = "web") {

    get("/{path...}") {
        val segments = call.parameters.getAll("path").orEmpty()
        val requested = segments.joinToString("/")
        val assetPath = resolveAsset(assets, basePath, requested)
        val bytes = readAssetOrNull(assets, assetPath)
            ?: readAssetOrNull(assets, "$basePath/index.html") // SPA fallback
        if (bytes == null) {
            call.respond(HttpStatusCode.NotFound)
            return@get
        }
        call.respondBytes(bytes, contentTypeFor(assetPath))
    }
}

private fun resolveAsset(assets: AssetManager, basePath: String, requested: String): String {
    val clean = requested.trim('/')
    if (clean.isEmpty()) return "$basePath/index.html"
    return if (assetExists(assets, "$basePath/$clean")) "$basePath/$clean" else "$basePath/index.html"
}

private fun assetExists(assets: AssetManager, path: String): Boolean =
    runCatching { assets.open(path).use { true } }.getOrDefault(false)

private fun readAssetOrNull(assets: AssetManager, path: String): ByteArray? = try {
    assets.open(path).use { it.readBytes() }
} catch (_: IOException) {
    null
}

private fun contentTypeFor(path: String): ContentType = when (path.substringAfterLast('.', "")) {
    "html" -> ContentType.Text.Html
    "js", "mjs" -> ContentType.Application.JavaScript
    "css" -> ContentType.Text.CSS
    "json" -> ContentType.Application.Json
    "svg" -> ContentType.Image.SVG
    "png" -> ContentType.Image.PNG
    "jpg", "jpeg" -> ContentType.Image.JPEG
    "ico" -> ContentType("image", "x-icon")
    "woff2" -> ContentType("font", "woff2")
    else -> ContentType.Application.OctetStream
}
