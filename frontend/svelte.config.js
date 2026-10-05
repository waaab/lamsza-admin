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
		}
	}
};

export default config;
