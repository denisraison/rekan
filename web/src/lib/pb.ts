import PocketBase from 'pocketbase';

// PUBLIC_POCKETBASE_URL set via .env (VITE_PUBLIC_POCKETBASE_URL not needed,
// SvelteKit static env requires build-time values; hardcode for now and override in prod)
export const pb = new PocketBase(
	typeof window !== 'undefined'
		? (import.meta.env.VITE_POCKETBASE_URL ?? 'http://localhost:8090')
		: 'http://localhost:8090',
);
