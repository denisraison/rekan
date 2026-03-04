import basicSsl from '@vitejs/plugin-basic-ssl';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { VitePWA } from 'vite-plugin-pwa';

const apiTarget = process.env.VITE_API_URL ?? 'http://127.0.0.1:8090';

export default defineConfig({
	envDir: '..',
	plugins: [
		basicSsl(),
		tailwindcss(),
		sveltekit(),
		VitePWA({
			registerType: 'autoUpdate',
			devOptions: { enabled: true },
			workbox: {
				navigateFallback: '/200.html',
				globPatterns: ['**/*.{js,css,html,svg,png,woff2}']
			},
			manifest: {
				name: 'Rekan',
				short_name: 'Rekan',
				description: 'Seu parceiro de conteúdo pro Instagram, direto no WhatsApp.',
				lang: 'pt-BR',
				theme_color: '#f97368',
				background_color: '#fafaf7',
				display: 'standalone',
				start_url: '/',
				scope: '/',
				icons: [
					{ src: 'icons/icon-192.png', sizes: '192x192', type: 'image/png' },
					{ src: 'icons/icon-512.png', sizes: '512x512', type: 'image/png' },
					{
						src: 'icons/icon-512.png',
						sizes: '512x512',
						type: 'image/png',
						purpose: 'maskable'
					}
				]
			}
		})
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
