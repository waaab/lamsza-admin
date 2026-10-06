import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		appDir: 'app',
		adapter: adapter({
			pages: 'dist',
			assets: 'dist',
			fallback: 'app.html',
			precompress: false,
			strict: false
		}),
		prerender: {
			handleUnseenRoutes: 'ignore'
		},
		// Content-Security-Policy, the same shape lamsza carries (BOG-17). The
		// pages are static, so SvelteKit writes this into a <meta http-equiv> tag
		// in every built page and hashes its own inline scripts and styles for us.
		//
		// Admin is the highest-value target in the family - every route behind it
		// writes the directory the main app reads - so script-src 'self' matters
		// most here: it is the lock on any injected tag, and connect-src names the
		// only hosts a page may talk to, so the exfiltration leg is blocked too.
		// Keep both lists as short as the app allows.
		//
		// Nginx also serves security headers. frame-ancestors cannot work from a
		// meta tag, so X-Frame-Options there stays the control for framing.
		csp: {
			mode: 'hash',
			directives: {
				'default-src': ['self'],
				'base-uri': ['self'],
				'object-src': ['none'],
				'form-action': ['self'],
				// Google Identity Services is injected by GoogleSignIn.svelte.
				'script-src': ['self', 'https://accounts.google.com/gsi/client'],
				'style-src': ['self', 'unsafe-inline', 'https://accounts.google.com/gsi/style'],
				// Entry, event and attraction pictures are URLs an admin pastes in,
				// plus Google avatars, so the host cannot be listed one by one.
				'img-src': ['self', 'data:', 'https:'],
				'font-src': ['self', 'data:'],
				// src/lib/api.js resolves the API to window.location.origin in the
				// browser (Vite proxy in dev, reverse proxy in production), so
				// 'self' covers every admin API call.
				'connect-src': ['self', 'https://accounts.google.com/gsi/'],
				'frame-src': ['self', 'https://accounts.google.com/gsi/'],
				'manifest-src': ['self'],
				'worker-src': ['self']
			}
		}
	}
};

export default config;
