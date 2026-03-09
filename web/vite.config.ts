import basicSsl from '@vitejs/plugin-basic-ssl';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { mockApi } from './src/mocks/vite-plugin';

const apiTarget = process.env.VITE_API_URL ?? 'http://127.0.0.1:8090';
const useMockApi = process.env.VITE_MOCK_API === 'true';

export default defineConfig({
	envDir: '..',
	plugins: [
		...(useMockApi ? [mockApi()] : []),
		basicSsl(),
		tailwindcss(),
		sveltekit(),
	],
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
