self.addEventListener('activate', (event) => {
    skipWaiting()
    event.waitUntil(clients.claim())
})

self.addEventListener('fetch', (event) => {
    console.log(event.request.url)
    // event.respondWith(fetch("https://developer.mozilla.org"))
})
