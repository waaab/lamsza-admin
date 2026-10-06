// Drift guard for the modules lamsza shares with lamsza-admin.
//
// lamsza owns these files; lamsza-admin carries a generated copy. The manifest
// of SHA-256 hashes is committed in both repos, so this test catches drift with
// only this one repo checked out - no sibling clone, no submodule.
//
// Red here means: edit the module in lamsza, run
// scripts/sync-shared-frontend.sh, and commit both repos.
// See docs/network/SHARED_FRONTEND_MODULES.md.

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath = join(repoRoot, 'shared-frontend-modules.json');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));

// In lamsza the frontend is the repo root (WAYS_OF_WORKING §5); in
// lamsza-admin the same copy of this test resolves against frontend/.
const frontendRoot = repoRoot;

test('the shared-module manifest is not empty', () => {
	assert.ok(
		Object.keys(manifest.modules).length > 0,
		'shared-frontend-modules.json lists no modules'
	);
});

for (const [relPath, expected] of Object.entries(manifest.modules)) {
	test(`shared module matches the manifest: ${relPath}`, () => {
		let source;
		try {
			source = readFileSync(join(frontendRoot, relPath));
		} catch (err) {
			assert.fail(
				`${relPath} is in shared-frontend-modules.json but missing from this repo (${err.code}). ` +
					'Either restore it or drop it from the manifest with scripts/sync-shared-frontend.sh.'
			);
		}
		const actual = `sha256:${createHash('sha256').update(source).digest('hex')}`;
		assert.equal(
			actual,
			expected,
			`${relPath} drifted from the shared copy. Edit it in lamsza, run ` +
				'scripts/sync-shared-frontend.sh, and commit lamsza and lamsza-admin together.'
		);
	});
}
