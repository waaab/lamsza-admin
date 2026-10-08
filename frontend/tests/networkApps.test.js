// Shared from lamsza (scripts/sync-shared-frontend.sh): the apps launcher's list.
import test from 'node:test';
import assert from 'node:assert/strict';
import { NETWORK_APPS, launcherApps } from '../src/lib/networkApps.js';

test('the launcher lists the three public apps in order, never admin', () => {
	assert.deepEqual(
		NETWORK_APPS.map((app) => app.id),
		['lamsza', 'szotar', 'jatszoter']
	);
	assert.deepEqual(
		NETWORK_APPS.map((app) => [app.name, app.icon]),
		[
			['Lámsza', 'home'],
			['Szótár', 'szotar'],
			['Játszótér', 'jatszoter']
		]
	);
});

test('exactly the current app is marked current', () => {
	for (const current of ['lamsza', 'szotar', 'jatszoter']) {
		const apps = launcherApps(current, 'lamsza.com');
		assert.deepEqual(
			apps.filter((app) => app.current).map((app) => app.id),
			[current]
		);
	}
});

test('links go to each app home on the host the visitor is on', () => {
	const hrefs = (hostname) => launcherApps('lamsza', hostname).map((app) => app.href);
	assert.deepEqual(hrefs('localhost'), [
		'http://localhost:5174/',
		'http://localhost:5175/',
		'http://localhost:5176/'
	]);
	assert.deepEqual(hrefs('szotar.lamsza.test'), [
		'https://lamsza.test/',
		'https://szotar.lamsza.test/',
		'https://jatszoter.lamsza.test/'
	]);
	assert.deepEqual(hrefs('jatszoter.lamsza.com'), [
		'https://lamsza.com/',
		'https://szotar.lamsza.com/',
		'https://jatszoter.lamsza.com/'
	]);
});
