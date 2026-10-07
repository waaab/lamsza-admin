// No em dash (U+2014) in UI text or code comments (the owner's rule D5,
// `.cursor/rules/no-emdash.mdc`). Documentation (Markdown) is exempt.
//
// Shared with every app through lamsza/scripts/sync-shared-frontend.sh, so each
// repo checks its own code in its own `npm test` (and CI). In lamsza the
// frontend is the repo root; in the other apps this file sits in
// frontend/tests and the repo root is one level up from frontend/.
//
// Red here means: end the sentence, or use a hyphen with spaces (" - ").

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, extname, join, relative } from 'node:path';

const frontendRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = existsSync(join(frontendRoot, 'backend')) ? frontendRoot : join(frontendRoot, '..');

/** Code and UI text. Markdown is exempt; data files (.json) are content, not code. */
const CHECKED = new Set(['.go', '.js', '.mjs', '.cjs', '.svelte', '.css', '.html', '.sh', '.sql', '.yml', '.yaml']);

/** Generated output, dependencies, local tool state, data and docs. */
const SKIPPED_DIRS = new Set([
	'node_modules',
	'.svelte-kit',
	'dist',
	'build',
	'extension',
	'.git',
	'.worktrees',
	'.superpowers',
	'.claude',
	'.cursor',
	'docs',
	'data',
	'tmp'
]);

const EM_DASH = '\u2014'; // written as an escape so this file passes its own check

/** @param {string} dir @returns {string[]} */
function codeFiles(dir) {
	/** @type {string[]} */
	const out = [];
	for (const name of readdirSync(dir)) {
		const path = join(dir, name);
		const stat = statSync(path);
		if (stat.isDirectory()) {
			if (!SKIPPED_DIRS.has(name)) out.push(...codeFiles(path));
		} else if (CHECKED.has(extname(name))) {
			out.push(path);
		}
	}
	return out;
}

test('no em dash in code or UI text (rule D5)', () => {
	/** @type {string[]} */
	const hits = [];
	for (const file of codeFiles(repoRoot)) {
		const lines = readFileSync(file, 'utf8').split('\n');
		lines.forEach((line, i) => {
			if (line.includes(EM_DASH)) hits.push(`${relative(repoRoot, file)}:${i + 1}: ${line.trim()}`);
		});
	}
	assert.equal(
		hits.length,
		0,
		`em dash in code or UI text; use " - " or end the sentence (.cursor/rules/no-emdash.mdc):\n${hits.join('\n')}`
	);
});
