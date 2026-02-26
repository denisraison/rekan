import PocketBase from 'pocketbase';

export const pb = new PocketBase(
	typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8090',
);
