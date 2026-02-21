// Auth state lives in localStorage (PocketBase JS SDK).
// SSR cannot access it, so disable server rendering for all app routes.
export const ssr = false;
