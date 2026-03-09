// Self-destructing SW: replaces the old Workbox SW, clears all caches, then unregisters.
self.addEventListener('install', () => self.skipWaiting());

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys()
			.then((keys) => Promise.all(keys.map((k) => caches.delete(k))))
			.then(() => self.clients.claim())
			.then(() => self.registration.unregister())
	);
});
