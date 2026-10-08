/**
 * The user details on every public app's Fiók tab (UI_BASELINE "acc-page").
 * Shared from lamsza by scripts/sync-shared-frontend.sh, so Lámsza, Szótár
 * and Játszótér show the same rows in the same order. Each app passes its
 * own `/me` answer through accountFromMe; Lámsza maps its auth store.
 */

/**
 * @typedef {{
 *   picture: string, displayName: string, name: string, givenName: string,
 *   familyName: string, email: string, googleSub: string,
 *   lastLoginAt: string | null, createdAt: string | null
 * }} Account
 */

/**
 * One account from any app's `/me` user object (snake_case). Játszótér calls
 * the Google ID `google_id`; the others `google_sub`.
 *
 * @param {Record<string, any> | null | undefined} me
 * @returns {Account}
 */
export function accountFromMe(me) {
	const u = me || {};
	return {
		picture: String(u.picture ?? ''),
		displayName: String(u.display_name ?? '').trim(),
		name: String(u.name ?? ''),
		givenName: String(u.given_name ?? ''),
		familyName: String(u.family_name ?? ''),
		email: String(u.email ?? ''),
		googleSub: String(u.google_sub ?? u.google_id ?? ''),
		lastLoginAt: u.last_login_at ?? null,
		createdAt: u.created_at ?? null
	};
}

/**
 * A date and time as Hungarians read it, on Bucharest's clock (R19).
 *
 * @param {string | null | undefined} iso
 */
export function formatAccountTime(iso) {
	if (!iso) return '';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return String(iso);
	return d.toLocaleString('hu-HU', { timeZone: 'Europe/Bucharest' });
}

/**
 * The rows to show, in order, without the empty ones.
 *
 * @param {Partial<Account>} account
 * @returns {{ key: string, label: string, value: string, kind: 'photo' | 'text' }[]}
 */
export function accountDetailRows(account) {
	const a = account || {};
	const rows = [
		['picture', 'Fénykép', a.picture, 'photo'],
		['displayName', 'Megjelenített név', a.displayName, 'text'],
		['name', 'Név', a.name, 'text'],
		['givenName', 'Keresztnév', a.givenName, 'text'],
		['familyName', 'Vezetéknév', a.familyName, 'text'],
		['email', 'E-mail', a.email, 'text'],
		['googleSub', 'Google-azonosító', a.googleSub, 'text'],
		['lastLoginAt', 'Utolsó belépés', formatAccountTime(a.lastLoginAt), 'text'],
		['createdAt', 'Fiók létrehozva', formatAccountTime(a.createdAt), 'text']
	];
	// An app that shows Google's name when none was chosen (Játszótér) sends
	// it as the display name too: one row is enough.
	const sameAsName = String(a.displayName ?? '').trim() === String(a.name ?? '').trim();
	return rows
		.filter(([key]) => !(key === 'displayName' && sameAsName))
		.filter(([, , value]) => String(value ?? '').trim() !== '')
		.map(([key, label, value, kind]) => ({ key, label, value: String(value), kind }));
}

/** The name the toolbar menu and greetings use: the chosen one, else Google's. */
export function accountShownName(account) {
	const a = account || {};
	return String(a.displayName || a.name || a.email || '').trim();
}
