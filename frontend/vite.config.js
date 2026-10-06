import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	// One `.env` at the repo root serves the Go backend and this frontend, which
	// is what `.env.example` documents. Without this Vite only looks in
	// `frontend/`, so every `VITE_*` value in the root file was silently ignored
	// and the app used its hard-coded localhost defaults instead. Same choice as
	// jatszoter/frontend/vite.config.js. Only `VITE_*` names reach the client, so
	// the weather keys in that file stay out of the bundle.
	envDir: '..',
	server: {
		port: 5173,
		strictPort: true,
		allowedHosts: ['.test', 'localhost', '127.0.0.1', '::1'],
		proxy: {
			'/api': {
				target: 'http://localhost:3000',
				changeOrigin: true
			}
		}
	}
});
