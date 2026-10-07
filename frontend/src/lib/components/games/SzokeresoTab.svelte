<script>
    /**
     * The Szókereső puzzle editor, moved here from Játszótér's
     * /admin/szokereso (lamsza WAYS_OF_WORKING R18): seven words from the
     * dictionary pool placed by hand in eight directions, then Kitöltés fills
     * the rest. One published puzzle per date and difficulty (DATE_TAKEN); a
     * puzzle someone has already played is read-only (FROZEN).
     *
     * @type {{ onChanged?: () => void }}
     */
    let { onChanged = () => {} } = $props();

    import { onMount } from 'svelte';
    import HuDateInput from '$lib/components/HuDateInput.svelte';
    import { errorText, sectionFetch } from '$lib/sectionApi.js';

    const DIRECTIONS = [
        { label: '→', dr: 0, dc: 1 },
        { label: '←', dr: 0, dc: -1 },
        { label: '↓', dr: 1, dc: 0 },
        { label: '↑', dr: -1, dc: 0 },
        { label: '↘', dr: 1, dc: 1 },
        { label: '↙', dr: 1, dc: -1 },
        { label: '↗', dr: -1, dc: 1 },
        { label: '↖', dr: -1, dc: -1 }
    ];

    let loading = $state(true);
    let saving = $state(false);
    let error = $state('');
    let notice = $state('');
    let placementError = $state('');
    /** @type {string[]} */
    let wordPool = $state([]);
    /** @type {any[]} */
    let puzzles = $state([]);
    let editingId = $state('');
    let frozen = $state(false);

    let boardDate = $state('');
    let size = $state(10);
    let difficulty = $state(1);
    let words = $state(['', '', '', '', '', '', '']);
    /** @type {Record<string, { r: number, c: number }[]>} */
    let placements = $state({});
    /** @type {string[][]} */
    let grid = $state([]);
    let selectedWordIndex = $state(0);
    let selectedDir = $state(0);
    /** @type {string[] | null} */
    let previewGrid = $state(null);
    /** @type {Record<number, string>} */
    const previousWordSelects = {};

    const readonly = $derived(frozen);
    const selectedWord = $derived(words[selectedWordIndex] ?? '');
    const placementsMap = $derived(placements);
    const blockNotes = $derived.by(() => {
        const notes = [];
        for (const word of words) {
            if (!word) continue;
            for (const other of wordPool) {
                if (other !== word && word.includes(other)) {
                    notes.push(`A ${word} a ${other} szót is kiadja a rácson. Válassz másik szót a folytatáshoz.`);
                }
            }
        }
        if (words.every((word) => word)) {
            const missing = words.filter((word) => !placementsMap[word]?.length);
            if (missing.length) {
                notes.push(`Még nincs a rácson: ${missing.join(', ')}. Kattints egy mezőre a lerakáshoz.`);
            }
        }
        return notes;
    });

    onMount(async () => {
        await loadAll();
        resetForm();
    });

    /** Játszótér's today, a Bucharest day (lamsza WAYS_OF_WORKING R19), never the browser's. */
    let today = $state('');
    const todayISO = () => today;

    /** @param {number} n @returns {string[][]} */
    function emptyGrid(n) {
        return Array.from({ length: n }, () => Array(n).fill(''));
    }

    /** @param {string[]} rows @param {number} n */
    function gridFromRows(rows, n) {
        const g = emptyGrid(n);
        for (let r = 0; r < n; r++) {
            const row = rows[r] ?? '';
            for (let c = 0; c < n; c++) {
                g[r][c] = row[c] ?? '';
            }
        }
        return g;
    }

    /** @param {string[][]} g */
    function gridToRows(g) {
        return g.map((row) => row.map((ch) => ch || ' ').join(''));
    }

    function rebuildGridFromPlacements(nextPlacements = placementsMap) {
        const g = emptyGrid(size);
        for (const word of words) {
            if (!word) continue;
            const cells = nextPlacements[word];
            if (!cells?.length) continue;
            for (let i = 0; i < cells.length; i++) {
                const { r, c } = cells[i];
                g[r][c] = word[i] ?? '';
            }
        }
        grid = g;
    }

    /** @param {number} i */
    function onWordSelectFocus(i) {
        previousWordSelects[i] = words[i];
    }

    /** @param {number} i */
    function onWordSelectChange(i) {
        const prev = previousWordSelects[i] ?? '';
        const next = words[i];
        if (prev && prev !== next) {
            const nextPlacements = { ...placementsMap };
            delete nextPlacements[prev];
            placements = nextPlacements;
            rebuildGridFromPlacements(nextPlacements);
        }
        previousWordSelects[i] = next;
    }

    async function loadAll() {
        loading = true;
        error = '';
        try {
            const [wordsRes, listRes, stats] = await Promise.all([
                sectionFetch("games", 'szokereso/words'),
                sectionFetch("games", 'szokereso/puzzles'),
                sectionFetch("games", 'stats')
            ]);
            today = stats?.today ?? '';
            wordPool = wordsRes.words ?? [];
            puzzles = listRes.puzzles ?? [];
        } catch (e) {
            error = errorText(e, 'A feladványok nem tölthetők be.');
        } finally {
            loading = false;
        }
    }

    function resetForm() {
        editingId = '';
        frozen = false;
        boardDate = todayISO();
        size = 10;
        difficulty = 1;
        words = ['', '', '', '', '', '', ''];
        placements = {};
        grid = emptyGrid(10);
        selectedWordIndex = 0;
        selectedDir = 0;
        previewGrid = null;
        placementError = '';
        error = '';
        notice = '';
    }

    /** @param {any} p */
    function startEdit(p) {
        editingId = p.id;
        frozen = p.frozen === true;
        boardDate = p.board_date ?? todayISO();
        size = p.size ?? 10;
        difficulty = p.difficulty ?? 1;
        words = [...(p.words ?? [])];
        while (words.length < 7) words.push('');
        words = words.slice(0, 7);
        placements = { ...(p.placements ?? {}) };
        grid = gridFromRows(p.grid ?? [], size);
        selectedWordIndex = 0;
        selectedDir = 0;
        previewGrid = null;
        placementError = '';
        error = '';
    }

    function onSizeChange() {
        if (readonly) return;
        placements = {};
        grid = emptyGrid(size);
        previewGrid = null;
        placementError = '';
    }

    /** @param {string} status */
    function puzzleBody(status) {
        return {
            board_date: boardDate,
            status,
            size,
            difficulty: Number(difficulty) || 1,
            words: [...words],
            grid: gridToRows(grid),
            placements: { ...placementsMap }
        };
    }

    /** @param {number} r @param {number} c */
    function tryPlaceAt(r, c) {
        if (readonly) return;
        placementError = '';
        const word = selectedWord;
        if (!word) return;
        const { dr, dc } = DIRECTIONS[selectedDir];
        const letters = [...word];
        const cells = [];
        for (let i = 0; i < letters.length; i++) {
            const nr = r + i * dr;
            const nc = c + i * dc;
            if (nr < 0 || nr >= size || nc < 0 || nc >= size) {
                placementError = 'Ez a lerakás nem fér el.';
                return;
            }
            cells.push({ r: nr, c: nc });
        }
        const others = { ...placementsMap };
        delete others[word];
        const base = emptyGrid(size);
        for (const w of words) {
            if (!w || w === word) continue;
            const placed = others[w];
            if (!placed?.length) continue;
            for (let i = 0; i < placed.length; i++) {
                const { r: br, c: bc } = placed[i];
                base[br][bc] = w[i] ?? '';
            }
        }
        for (let i = 0; i < letters.length; i++) {
            const { r: cr, c: cc } = cells[i];
            const existing = base[cr][cc];
            if (existing && existing !== letters[i]) {
                placementError = 'Ez a lerakás nem fér el.';
                return;
            }
        }
        const nextPlacements = { ...placementsMap, [word]: cells };
        placements = nextPlacements;
        rebuildGridFromPlacements(nextPlacements);
    }

    async function runFill() {
        if (readonly) return;
        error = '';
        placementError = '';
        try {
            const res = await sectionFetch("games", 'szokereso/fill', {
                method: 'POST',
                body: puzzleBody('draft')
            });
            grid = gridFromRows(res.grid ?? [], size);
            previewGrid = null;
        } catch (e) {
            error = errorText(e, 'A kitöltés nem sikerült.');
        }
    }

    async function runPreview() {
        error = '';
        try {
            const res = await sectionFetch("games", 'szokereso/preview', {
                method: 'POST',
                body: puzzleBody('draft')
            });
            previewGrid = res.grid ?? null;
        } catch (e) {
            error = errorText(e, 'Az előnézet nem sikerült.');
        }
    }

    /** @param {string} status */
    async function savePuzzle(status) {
        if (readonly) return;
        saving = true;
        error = '';
        try {
            const body = puzzleBody(status);
            if (editingId) {
                await sectionFetch("games", `szokereso/puzzles/${editingId}`, {
                    method: 'PUT',
                    body
                });
            } else {
                const created = await sectionFetch("games", 'szokereso/puzzles', {
                    method: 'POST',
                    body
                });
                editingId = created.puzzle?.id ?? '';
                if (status === 'published' && editingId) {
                    await sectionFetch("games", `szokereso/puzzles/${editingId}`, {
                        method: 'PUT',
                        body: { ...body, status: 'published' }
                    });
                }
            }
            notice = status === 'published' ? 'Közzétéve.' : 'Piszkozat mentve.';
            onChanged();
            await loadAll();
            if (editingId) {
                const row = puzzles.find((p) => p.id === editingId);
                if (row) startEdit(row);
            }
        } catch (e) {
            const message = errorText(e, 'A mentés nem sikerült.');
            // A new puzzle is saved as a draft before it is published; when
            // the publish is refused, that draft still exists, so list it
            // (loadAll clears the error, so set it afterwards).
            if (editingId) await loadAll();
            error = message;
        } finally {
            saving = false;
        }
    }

    /** @param {string} s */
    function statusLabel(s) {
        return s === 'published' ? 'közzétéve' : 'piszkozat';
    }
</script>

{#if loading}
    <p>Betöltés…</p>
{:else}
    <p class="admin-info">
        {puzzles.length} feladvány. Hét szót helyezz el a rácsban, majd töltsd ki a maradék mezőket.
    </p>
    {#if notice}
        <div class="info-box success" role="status"><p>{notice}</p></div>
    {/if}
    {#if error}
        <div class="info-box error" role="alert"><p>{error}</p></div>
    {/if}
    {#if placementError}
        <div class="info-box error" role="alert"><p>{placementError}</p></div>
    {/if}

    <section class="szk-editor" aria-labelledby="szk-editor-title">
        <h3 id="szk-editor-title">
            {editingId ? `Szerkesztés: ${editingId}` : 'Új feladvány'}
            {#if frozen}
                <span class="frozen-tag">zárolt</span>
            {/if}
        </h3>

        <div class="form-grid">
            <label for="szk-date">
                Dátum
                <HuDateInput id="szk-date" name="board_date" bind:value={boardDate} disabled={readonly} required />
            </label>
            <label for="szk-difficulty">
                Nehézség
                <select id="szk-difficulty" name="difficulty" bind:value={difficulty} disabled={readonly}>
                    <option value={1}>Könnyű</option>
                    <option value={2}>Közepes</option>
                    <option value={3}>Nehéz</option>
                </select>
            </label>
            <label for="szk-size">
                Méret
                <select id="szk-size" name="size" bind:value={size} onchange={onSizeChange} disabled={readonly}>
                    {#each [8, 9, 10, 11, 12] as n (n)}
                        <option value={n}>{n}×{n}</option>
                    {/each}
                </select>
            </label>
        </div>

        <div class="word-selects">
            {#each words as word, i (i)}
                <label for="szk-word-{i}">
                    {i + 1}. szó
                    <select
                        id="szk-word-{i}"
                        name="word_{i}"
                        bind:value={words[i]}
                        disabled={readonly}
                        onfocus={() => onWordSelectFocus(i)}
                        onchange={() => onWordSelectChange(i)}
                    >
                        <option value="">- válassz -</option>
                        {#each wordPool as poolWord (poolWord)}
                            <option value={poolWord}>{poolWord}</option>
                        {/each}
                    </select>
                </label>
            {/each}
        </div>

        <div class="placement-bar">
            <label for="szk-place-word">
                Lerakás szava
                <select id="szk-place-word" name="place_word" bind:value={selectedWordIndex} disabled={readonly}>
                    {#each words as w, i (i)}
                        <option value={i} disabled={!w}>{i + 1}. {w || '-'}</option>
                    {/each}
                </select>
            </label>
            <div class="dir-buttons" role="group" aria-label="Irány">
                {#each DIRECTIONS as dir, i (i)}
                    <button
                        type="button"
                        class="btn btn-sm"
                        class:active={selectedDir === i}
                        aria-pressed={selectedDir === i}
                        disabled={readonly || !selectedWord}
                        onclick={() => (selectedDir = i)}
                    >
                        {dir.label}
                    </button>
                {/each}
            </div>
        </div>

        <div class="grid-wrap">
            <table class="letter-grid">
                <tbody>
                    {#each grid as row, r (r)}
                        <tr>
                            {#each row as cell, c (`${r}-${c}`)}
                                <td>
                                    <button
                                        type="button"
                                        class="cell-btn"
                                        class:filled={cell}
                                        disabled={readonly || !selectedWord}
                                        aria-label="{r + 1}. sor, {c + 1}. oszlop"
                                        onclick={() => tryPlaceAt(r, c)}
                                    >
                                        {cell}
                                    </button>
                                </td>
                            {/each}
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>

        {#if blockNotes.length}
            <div class="block-notes">
                {#each blockNotes as note (note)}
                    <div class="info-box warning" role="status"><p>{note}</p></div>
                {/each}
            </div>
        {/if}

        <div class="admin-modal-actions">
            <button type="button" class="btn-update" onclick={runFill} disabled={readonly || saving || blockNotes.length > 0}>
                Kitöltés
            </button>
            <button type="button" class="btn-update" onclick={runPreview} disabled={readonly || saving || blockNotes.length > 0}>
                Előnézet
            </button>
            <button
                type="button"
                class="btn-update"
                onclick={() => savePuzzle('draft')}
                disabled={readonly || saving || blockNotes.length > 0}
            >
                {saving ? 'Mentés…' : 'Piszkozat'}
            </button>
            <button
                type="button"
                class="admin-submit-btn"
                onclick={() => savePuzzle('published')}
                disabled={readonly || saving || blockNotes.length > 0}
            >
                Közzététel
            </button>
            {#if editingId}
                <button type="button" class="btn-update" onclick={resetForm} disabled={saving}>Új feladvány</button>
            {/if}
        </div>

        {#if previewGrid}
            <div class="preview">
                <p class="preview-label">Előnézet (játékos nézet)</p>
                <table class="letter-grid preview-grid">
                    <tbody>
                        {#each previewGrid as row, ri (ri)}
                            <tr>
                                {#each [...row] as ch, ci (`${ri}-${ci}`)}
                                    <td><span class="preview-cell">{ch}</span></td>
                                {/each}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>

    <h3 class="admin-subsection-title">Összes feladvány</h3>
    <div class="admin-table-wrapper">
        <table class="admin-table">
            <thead>
                <tr>
                    <th>Dátum</th>
                    <th>Státusz</th>
                    <th>Nehézség</th>
                    <th>Méret</th>
                    <th>Szavak</th>
                    <th class="admin-table-col--action">Szerk.</th>
                </tr>
            </thead>
            <tbody>
                {#each puzzles as p (p.id)}
                    <tr class:admin-row-muted={p.frozen}>
                        <td><code>{p.board_date}</code></td>
                        <td>{statusLabel(p.status)}{p.frozen ? ' (zárolt)' : ''}</td>
                        <td>{['', 'Könnyű', 'Közepes', 'Nehéz'][p.difficulty] ?? p.difficulty}</td>
                        <td>{p.size}×{p.size}</td>
                        <td class="words-col">{(p.words ?? []).join(', ')}</td>
                        <td>
                            <button type="button" class="btn-update" onclick={() => startEdit(p)}>
                                {p.frozen ? 'Megnyitás' : 'Szerk.'}
                            </button>
                        </td>
                    </tr>
                {:else}
                    <tr><td colspan="6">Még nincs feladvány.</td></tr>
                {/each}
            </tbody>
        </table>
    </div>
{/if}

<style>
    .admin-info {
        color: var(--text-faint, #666);
        margin-bottom: 1rem;
    }

    .szk-editor {
        margin-bottom: 2rem;
        padding: 1.5rem;
        border: 1px solid var(--border-color);
        border-radius: 6px;
    }

    .szk-editor > h3 {
        margin: 0 0 1rem;
        font-size: var(--text-lg);
    }

    label {
        display: block;
        margin-bottom: 0.75rem;
        font-weight: 500;
        color: var(--text-primary);
    }

    select,
    label :global(input) {
        display: block;
        box-sizing: border-box;
        width: 100%;
        margin-top: 0.35rem;
        padding: 0.5rem 0.65rem;
        border: 1px solid var(--border-color);
        border-radius: 4px;
        background: var(--card-bg);
        color: var(--text-primary);
        font: inherit;
    }

    .form-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 1rem;
    }

    .word-selects {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
        gap: 0.75rem;
        margin: 1rem 0;
    }

    .placement-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: flex-end;
        gap: 1rem;
        margin-bottom: 1rem;
    }

    .placement-bar label {
        flex: 1 1 12rem;
        margin-bottom: 0;
    }

    .dir-buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 0.35rem;
    }

    .dir-buttons .btn.active {
        background: var(--szekely-green);
        color: var(--card-bg);
        border-color: var(--szekely-green);
    }

    .grid-wrap {
        overflow-x: auto;
        margin-bottom: 1rem;
    }

    .letter-grid {
        border-collapse: collapse;
        margin: 0 auto;
    }

    .letter-grid td {
        padding: 2px;
    }

    .cell-btn {
        width: 2.25rem;
        height: 2.25rem;
        padding: 0;
        border: 1px solid var(--border-color);
        border-radius: 6px;
        background: var(--card-bg);
        color: var(--text-primary);
        font-family: inherit;
        font-size: var(--text-sm);
        font-weight: 700;
        text-transform: lowercase;
        cursor: pointer;
    }

    .cell-btn.filled {
        background: color-mix(in srgb, var(--szekely-green) 12%, var(--card-bg));
    }

    .cell-btn:disabled {
        cursor: default;
        opacity: 0.7;
    }

    .preview-grid .preview-cell {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 2.25rem;
        height: 2.25rem;
        font-weight: 700;
        text-transform: lowercase;
    }

    .block-notes {
        margin: 0.75rem 0;
    }

    .preview {
        margin-top: 1rem;
        padding: 0.85rem;
        border: 1px solid var(--border-color);
        border-radius: 6px;
        background: var(--card-bg);
    }

    .preview-label {
        margin: 0 0 0.5rem;
        font-size: var(--text-xs);
        text-transform: uppercase;
        letter-spacing: 0.1em;
        color: var(--text-muted);
    }

    .words-col {
        max-width: 16rem;
        word-break: break-word;
    }

    .frozen-tag {
        margin-left: 0.5rem;
        font-size: var(--text-xs);
        font-weight: 600;
        text-transform: uppercase;
        color: var(--text-muted);
    }

    @media (max-width: 640px) {
        .form-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
