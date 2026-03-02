import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const apiTarget = process.env.VITE_API_URL ?? 'http://127.0.0.1:8090';

export default defineConfig({
	envDir: '..',
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: apiTarget,
				changeOrigin: true,
			},
			'/_': {
				target: apiTarget,
				changeOrigin: true,
			},
		},
	},
});
