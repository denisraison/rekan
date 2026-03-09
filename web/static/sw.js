// No caching. This SW exists only for PWA installability.
// On activate, delete all caches left by the old Workbox SW.

self.addEventListener('install', () => self.skipWaiting());

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys().then((keys) => Promise.all(keys.map((k) => caches.delete(k))))
			.then(() => self.clients.claim())
	);
});

self.addEventListener('message', (event) => {
	if (event.data && event.data.type === 'SKIP_WAITING') self.skipWaiting();
});
