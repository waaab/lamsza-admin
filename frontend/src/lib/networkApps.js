/**
 * The network's public apps, in the order the apps launcher shows them
 * (UI_BASELINE "tb-apps-launcher"). Shared from lamsza by
 * scripts/sync-shared-frontend.sh, so a new app is added here once and
 * appears in every toolbar. The admin app is not listed: no public app links
 * to it (UI_BASELINE "tb-admin-link").
 */
import { lamszaOrigin, szotarOrigin, jatszoterOrigin } from './networkOrigins.js';

export const NETWORK_APPS = [
	{ id: 'lamsza', name: 'Lámsza', icon: 'home', origin: lamszaOrigin },
	{ id: 'szotar', name: 'Szótár', icon: 'szotar', origin: szotarOrigin },
	{ id: 'jatszoter', name: 'Játszótér', icon: 'jatszoter', origin: jatszoterOrigin }
];

/**
 * The launcher's entries for the app the visitor is on.
 *
 * @param {string} current the id of the current app ('lamsza' | 'szotar' | 'jatszoter')
 * @param {string} [hostname] request host (pass $page.url.hostname: the apps prerender)
 * @returns {{ id: string, name: string, icon: string, href: string, current: boolean }[]}
 */
export function launcherApps(current, hostname) {
	return NETWORK_APPS.map(({ id, name, icon, origin }) => ({
		id,
		name,
		icon,
		href: `${origin(hostname)}/`,
		current: id === current
	}));
}
