import PocketBase from 'pocketbase';
import { browser, dev } from '$app/environment';

export const pb = new PocketBase(
	typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8090',
);

if (browser) {
	// Load auth from cookie so it survives page refreshes.
	if (document.cookie.includes('pb_auth=')) {
		pb.authStore.loadFromCookie(document.cookie);
	}

	// Sync auth state back to cookie on every change (login/logout/refresh).
	pb.authStore.onChange(() => {
		document.cookie = pb.authStore.isValid
			? pb.authStore.exportToCookie({ httpOnly: false, secure: !dev, sameSite: 'Lax' })
			: 'pb_auth=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
	});
}
