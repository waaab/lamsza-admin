// Shared from lamsza (scripts/sync-shared-frontend.sh): the Fiók tab's rows.
import test from 'node:test';
import assert from 'node:assert/strict';
import { accountDetailRows, accountFromMe, accountShownName, formatAccountTime } from '../src/lib/accountDetails.js';

const me = {
	picture: 'https://lh3.googleusercontent.com/a/x',
	display_name: ' Attis ',
	name: 'Bogozi Attila',
	given_name: 'Attila',
	family_name: 'Bogozi',
	email: 'a@example.com',
	google_sub: '1234567890',
	last_login_at: '2026-10-08T09:30:00Z',
	created_at: '2026-01-02T22:15:00Z'
};

test('every row in the agreed order', () => {
	assert.deepEqual(
		accountDetailRows(accountFromMe(me)).map((r) => r.label),
		['Fénykép', 'Megjelenített név', 'Név', 'Keresztnév', 'Vezetéknév', 'E-mail', 'Google-azonosító', 'Utolsó belépés', 'Fiók létrehozva']
	);
});

test('empty fields are left out', () => {
	const rows = accountDetailRows(accountFromMe({ email: 'a@example.com', name: '', picture: null }));
	assert.deepEqual(rows.map((r) => r.key), ['email']);
});

test('Játszótér google_id is the Google ID', () => {
	assert.equal(accountFromMe({ google_id: '42' }).googleSub, '42');
});

test('times are shown on the Bucharest clock', () => {
	// 22:15 UTC on 2 January is 00:15 on 3 January in Bucharest (UTC+2).
	assert.match(formatAccountTime('2026-01-02T22:15:00Z'), /2026\. 01\. 03\. 0?0:15/);
});

test('the shown name is the chosen one, else Google\'s, else the email', () => {
	assert.equal(accountShownName(accountFromMe(me)), 'Attis');
	assert.equal(accountShownName(accountFromMe({ ...me, display_name: '' })), 'Bogozi Attila');
	assert.equal(accountShownName(accountFromMe({ email: 'a@example.com' })), 'a@example.com');
});

test('a display name equal to the Google name is not listed twice', () => {
	const rows = accountDetailRows(accountFromMe({ ...me, display_name: 'Bogozi Attila' }));
	assert.equal(rows.filter((r) => r.value === 'Bogozi Attila').length, 1);
	assert.ok(!rows.some((r) => r.key === 'displayName'));
});
