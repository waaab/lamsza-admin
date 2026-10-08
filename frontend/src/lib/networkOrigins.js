const APPS = {
	lamsza: {
		envKey: 'VITE_LAMSZA_ORIGIN',
		local: 'https://lamsza.test',
		/** Vite strictPort: Admin 5173, Lámsza 5174, Szótár 5175, Játszótér 5176. */
		localVite: 'http://localhost:5174',
		prod: 'https://lamsza.com'
	},
	szotar: {
		envKey: 'VITE_SZOTAR_ORIGIN',
		local: 'https://szotar.lamsza.test',
		localVite: 'http://localhost:5175',
		prod: 'https://szotar.lamsza.com'
	},
	jatszoter: {
		envKey: 'VITE_JATSZOTER_ORIGIN',
		local: 'https://jatszoter.lamsza.test',
		localVite: 'http://localhost:5176',
		prod: 'https://jatszoter.lamsza.com'
	}
};

/** @param {string} hostname */
function isLocalhost(hostname) {
	const host = String(hostname || '');
	return host === 'localhost' || host === '127.0.0.1';
}

/** @param {string} hostname */
function isLocalNetworkHost(hostname) {
	const host = String(hostname || '');
	return isLocalhost(host) || host === 'lamsza.test' || host.endsWith('.lamsza.test');
}

/**
 * @param {'lamsza'|'szotar'|'jatszoter'} app
 * @param {{ envOrigin?: string, hostname?: string }} [opts]
 */
export function resolveNetworkOrigin(app, opts = {}) {
	const cfg = APPS[app];
	if (!cfg) throw new Error(`Unknown network app: ${app}`);
	const env = String(opts.envOrigin ?? '')
		.trim()
		.replace(/\/$/, '');
	if (env) return env;
	const host = String(opts.hostname || '');
	if (isLocalhost(host)) return cfg.localVite || cfg.local;
	if (isLocalNetworkHost(host)) return cfg.local;
	return cfg.prod;
}

/** @param {string} origin @param {string} [path] */
export function joinOriginPath(origin, path = '') {
	const base = String(origin || '').replace(/\/$/, '');
	const p = String(path || '').trim();
	if (!p) return base;
	return `${base}${p.startsWith('/') ? p : `/${p}`}`;
}

/**
 * Read one of the network-origin overrides.
 *
 * Must stay a static switch. A computed lookup (`import.meta.env[key]`) cannot be
 * replaced at build time, so Vite inlines the whole env object into the client
 * bundle - that puts every `VITE_*` value in reach of any visitor. One
 * `import.meta.env.VITE_X` per key keeps the replacement static.
 *
 * @param {string} key
 */
function readEnv(key) {
	try {
		switch (key) {
			case 'VITE_LAMSZA_ORIGIN':
				return import.meta.env.VITE_LAMSZA_ORIGIN;
			case 'VITE_SZOTAR_ORIGIN':
				return import.meta.env.VITE_SZOTAR_ORIGIN;
			case 'VITE_JATSZOTER_ORIGIN':
				return import.meta.env.VITE_JATSZOTER_ORIGIN;
			default:
				return undefined;
		}
	} catch {
		return undefined;
	}
}

function currentHostname() {
	if (typeof window !== 'undefined' && window.location?.hostname) {
		return window.location.hostname;
	}
	return '';
}

/**
 * @param {'lamsza'|'szotar'|'jatszoter'} app
 * @param {string} [hostname] request host (prefer $page.url.hostname during SSR)
 */
function originFor(app, hostname) {
	const cfg = APPS[app];
	return resolveNetworkOrigin(app, {
		envOrigin: readEnv(cfg.envKey),
		hostname: hostname ?? currentHostname()
	});
}

/** @param {string} [hostname] */
export function lamszaOrigin(hostname) {
	return originFor('lamsza', hostname);
}
/** @param {string} [hostname] */
export function szotarOrigin(hostname) {
	return originFor('szotar', hostname);
}
/** @param {string} [hostname] */
export function jatszoterOrigin(hostname) {
	return originFor('jatszoter', hostname);
}

/** @param {string} [path] @param {string} [hostname] */
export function lamszaUrl(path = '', hostname) {
	return joinOriginPath(lamszaOrigin(hostname), path);
}
/** @param {string} [path] @param {string} [hostname] */
export function szotarUrl(path = '', hostname) {
	return joinOriginPath(szotarOrigin(hostname), path);
}
/** @param {string} [path] @param {string} [hostname] */
export function jatszoterUrl(path = '', hostname) {
	return joinOriginPath(jatszoterOrigin(hostname), path);
}
