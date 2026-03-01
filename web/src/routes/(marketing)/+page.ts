import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { pb } from '$lib/pb';

export function load() {
	if (browser && pb.authStore.isValid) {
		redirect(302, '/operador');
	}
}
