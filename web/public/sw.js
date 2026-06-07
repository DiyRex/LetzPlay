// Minimal service worker for installability + an offline app shell.
// API and WebSocket traffic always go to the network; only the static shell is cached.
const CACHE = "letzplay-shell-v1"

self.addEventListener("install", (event) => {
  self.skipWaiting()
  event.waitUntil(caches.open(CACHE).then((c) => c.add("./")))
})

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))),
  )
  self.clients.claim()
})

self.addEventListener("fetch", (event) => {
  const { request } = event
  const url = new URL(request.url)

  // Never cache API/WS — always live.
  if (request.method !== "GET" || url.pathname.startsWith("/api") || url.pathname.startsWith("/ws")) {
    return
  }

  // Navigations: network-first, fall back to the cached shell when offline.
  if (request.mode === "navigate") {
    event.respondWith(fetch(request).catch(() => caches.match("./")))
    return
  }

  // Hashed assets: cache-first (their names change on every build).
  event.respondWith(
    caches.match(request).then(
      (cached) =>
        cached ||
        fetch(request).then((res) => {
          const copy = res.clone()
          caches.open(CACHE).then((c) => c.put(request, copy))
          return res
        }),
    ),
  )
})
