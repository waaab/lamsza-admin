<script>
    /**
     * One Tájszórejtvény puzzle for review (spec §10.3, §16.6.4), with its
     * answers: approve, regenerate, replace with a spare of the same level,
     * edit a clue (optionally kept as the word's game clue), choose the
     * proverb, block a word. A live puzzle only takes clue edits and blocks
     * (the block then only applies to later puzzles). Regenerating and
     * blocking run in Játszótér's background (202); this view waits for them.
     *
     * @type {{ id: string, spares?: any[], onChanged?: () => void, onClose?: () => void, onOpen?: (id: string) => void }}
     */
    let { id, spares = [], onChanged = () => {}, onClose = () => {}, onOpen = () => {} } = $props();

    import PuzzleGrid from "$lib/components/games/tajszorejtveny/PuzzleGrid.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";
    import { FLAG_LABELS, LEVEL_LABELS, STATUS_LABELS, dayName, waitForTask } from "$lib/tajszorejtvenyAdmin.js";

    /** @type {any} */
    let p = $state(null);
    let error = $state("");
    let notice = $state("");
    let busy = $state("");
    let selected = $state(-1);
    /** @type {number | null} */
    let editing = $state(null);
    let clueDraft = $state("");
    let keepClue = $state(false);
    let proverbDraft = $state("");
    let spareId = $state("");

    const live = $derived(p?.status === "live");
    const puzzle = $derived(p?.puzzle);
    const sameLevelSpares = $derived(spares.filter((s) => s.level === p?.level && s.status !== "live"));

    async function load() {
        try {
            p = await sectionFetch("games", `tajszorejtveny/puzzles/${encodeURIComponent(id)}`);
            proverbDraft = p?.puzzle?.solution?.proverb ?? "";
            error = "";
        } catch (err) {
            error = errorText(err, "A rejtvény nem tölthető be.");
        }
    }

    $effect(() => {
        id;
        p = null;
        selected = -1;
        editing = null;
        notice = "";
        load();
    });

    /** @param {string} label @param {() => Promise<any>} fn @param {string} done */
    async function run(label, fn, done) {
        if (busy) return;
        busy = label;
        error = "";
        notice = "";
        try {
            const res = await fn();
            if (res && typeof res === "object" && res.puzzle) p = res;
            notice = done;
            onChanged();
        } catch (err) {
            error = errorText(err, "A művelet nem sikerült.");
        } finally {
            busy = "";
        }
    }

    /** A background action (202): wait for it, then reload. */
    async function runTask(/** @type {string} */ label, /** @type {string} */ path, /** @type {any} */ body, /** @type {string} */ done) {
        await run(
            label,
            async () => {
                const task = await sectionFetch("games", path, { method: "POST", body });
                const res = await waitForTask(() => sectionFetch("games", "tajszorejtveny/tasks"), task.key);
                if (res.error) throw new Error(res.error);
                if (!res.done) throw new Error("A művelet még fut; nézz vissza később.");
                await load();
                return null;
            },
            done,
        );
    }

    const approve = () =>
        run("approve", () => sectionFetch("games", `tajszorejtveny/puzzles/${id}/approve`, { method: "POST" }), "Jóváhagyva.");

    async function regenerate() {
        if (!(await confirmDialog("Új rácsot készítsünk erre a napra és szintre? A mostani elvész.", { yesLabel: "Újragenerálás" }))) return;
        await runTask("regenerate", `tajszorejtveny/puzzles/${id}/regenerate`, {}, "Új rács készült. Nézd át, majd hagyd jóvá.");
    }

    async function replace() {
        if (!spareId) return;
        if (!(await confirmDialog("Lecseréljük erre a tartalékra? A mostani rejtvény kikerül a hétből.", { yesLabel: "Csere" }))) return;
        const target = spareId;
        await run(
            "replace",
            () => sectionFetch("games", `tajszorejtveny/puzzles/${id}/replace`, { method: "POST", body: { spare_id: target } }),
            "A tartalék lett a nap rejtvénye.",
        );
        spareId = "";
        if (p?.id && p.id !== id) onOpen(p.id);
    }

    function startClue(/** @type {number} */ i) {
        editing = i;
        selected = i;
        clueDraft = puzzle.words[i].clue;
        keepClue = false;
    }

    async function saveClue() {
        if (editing === null) return;
        const word = editing;
        await run(
            "clue",
            () =>
                sectionFetch("games", `tajszorejtveny/puzzles/${id}/clue`, {
                    method: "POST",
                    body: { word, clue: clueDraft.trim(), save_override: keepClue },
                }),
            keepClue ? "A meghatározás mentve, a szó játékbeli meghatározásaként is." : "A meghatározás mentve.",
        );
        if (!error) editing = null;
    }

    async function saveProverb() {
        const text = proverbDraft.trim();
        if (!text || text === puzzle?.solution?.proverb) return;
        await run(
            "proverb",
            () => sectionFetch("games", `tajszorejtveny/puzzles/${id}/proverb`, { method: "POST", body: { text } }),
            "A mondás lecserélve.",
        );
    }

    async function block(/** @type {number} */ i) {
        const w = puzzle.words[i];
        const what = w.szekely ? "tájszót" : "töltelékszót";
        const then = live ? "Az élő rejtvény nem változik, a későbbiekben nem kerül elő." : "A rejtvény újragenerálódik nélküle.";
        if (!(await confirmDialog(`Letiltjuk a(z) „${w.answer}” ${what}? ${then}`, { yesLabel: "Tiltás", destructive: true }))) return;
        await runTask("block", `tajszorejtveny/puzzles/${id}/block`, { word: i }, `Letiltva: ${w.answer}.`);
    }

    const sourceLabel = (/** @type {string} */ s) => ({ szotar: "Szótár mondás", fallback: "saját lista", admin: "admin választotta" })[s] ?? s;
</script>

<section class="admin-subsection taj-puzzle" aria-labelledby="taj-puzzle-title">
    <div class="taj-puzzle-head">
        <h3 id="taj-puzzle-title">
            {#if p}
                {p.kind === "spare" ? "Tartalék" : `${formatHuDate(p.date)}, ${dayName(p.date)}`} · {LEVEL_LABELS[p.level]}
                <span class="taj-status taj-status--{p.status}">{STATUS_LABELS[p.status] ?? p.status}</span>
                {#if p.status === "live" && !p.reviewed}<span class="taj-status taj-status--needs_attention">átnézés nélkül</span>{/if}
            {:else}
                Rejtvény
            {/if}
        </h3>
        <button type="button" class="btn btn-sm" onclick={onClose}>Bezárás</button>
    </div>

    {#if error}
        <div class="info-box error" role="alert"><p>{error}</p></div>
    {/if}
    {#if notice}
        <div class="info-box success" role="status"><p>{notice}</p></div>
    {/if}

    {#if !p}
        {#if !error}<p>Betöltés…</p>{/if}
    {:else}
        {#if p.approved_by}
            <p class="taj-muted">Jóváhagyta: {p.approved_by}{p.approved_at ? `, ${formatHuDate(p.approved_at)}` : ""}</p>
        {/if}
        {#each puzzle.flags ?? [] as f (f)}
            <div class="info-box warning" role="status"><p>{FLAG_LABELS[f] ?? f}</p></div>
        {/each}
        <p class="taj-stats">
            {puzzle.stats.szekely} tájszó · {puzzle.stats.words} szó · {puzzle.stats.decorative} üres kocka ·
            {puzzle.stats.soft_repeats} ismétlés az elmúlt 4 hétből · mondás: {puzzle.stats.proverb_len} betű{puzzle.stats.proverb_used
                ? ", 8 héten belül már szerepelt"
                : ""}
        </p>

        <div class="admin-modal-actions taj-actions">
            {#if p.status === "draft" || p.status === "needs_attention"}
                <button type="button" class="admin-submit-btn" disabled={!!busy} onclick={approve}>
                    {busy === "approve" ? "Jóváhagyás…" : "Jóváhagyás"}
                </button>
            {/if}
            {#if !live && p.kind === "daily"}
                <button type="button" class="btn-update" disabled={!!busy} onclick={regenerate}>
                    {busy === "regenerate" ? "Generálás…" : "Újragenerálás"}
                </button>
                {#if sameLevelSpares.length}
                    <label class="taj-inline">
                        <span class="sr-only">Tartalék</span>
                        <select bind:value={spareId} disabled={!!busy}>
                            <option value="">Csere tartalékkal…</option>
                            {#each sameLevelSpares as s, i (s.id)}
                                <option value={s.id}>{i + 1}. tartalék ({s.stats.szekely} tájszó, {s.stats.decorative} üres)</option>
                            {/each}
                        </select>
                    </label>
                    <button type="button" class="btn-update" disabled={!!busy || !spareId} onclick={replace}>Csere</button>
                {/if}
            {/if}
        </div>

        <PuzzleGrid {puzzle} {selected} onselect={(w) => (selected = w)} />

        <div class="taj-proverb">
            <h4>Megfejtés</h4>
            <p>
                <strong>{puzzle.solution.proverb}</strong>
                <span class="taj-muted">({sourceLabel(puzzle.solution.source)})</span>
            </p>
            {#if !live}
                <form
                    class="taj-proverb-form"
                    onsubmit={(e) => {
                        e.preventDefault();
                        saveProverb();
                    }}
                >
                    <label for="taj-proverb-{id}">Másik mondás (a rács betűiből kell kijönnie)</label>
                    <input id="taj-proverb-{id}" class="admin-search-input" bind:value={proverbDraft} maxlength="120" />
                    <button type="submit" class="btn-update" disabled={!!busy || !proverbDraft.trim() || proverbDraft.trim() === puzzle.solution.proverb}
                        >{busy === "proverb" ? "Mentés…" : "Mondás cseréje"}</button
                    >
                </form>
            {/if}
        </div>

        <h4>Szavak</h4>
        <div class="admin-table-wrapper">
            <table class="admin-table admin-table--compact">
                <thead>
                    <tr>
                        <th>#</th>
                        <th>Szó</th>
                        <th>Meghatározás</th>
                        <th>Fajta</th>
                        <th class="admin-table-col--action">Szerk.</th>
                        <th class="admin-table-col--action">Tiltás</th>
                    </tr>
                </thead>
                <tbody>
                    {#each puzzle.words as w, i (w.id)}
                        <tr class:taj-selected={selected === i}>
                            <td>{i + 1}</td>
                            <td><button type="button" class="taj-link" onclick={() => (selected = i)}>{w.answer}</button></td>
                            <td>
                                {#if editing === i}
                                    <form
                                        class="taj-clue-form"
                                        onsubmit={(e) => {
                                            e.preventDefault();
                                            saveClue();
                                        }}
                                    >
                                        <input class="admin-search-input" bind:value={clueDraft} maxlength="60" aria-label="Meghatározás" />
                                        {#if w.szekely}
                                            <label class="taj-check"
                                                ><input type="checkbox" bind:checked={keepClue} /> A szó játékbeli meghatározása is legyen</label
                                            >
                                        {/if}
                                        <div class="taj-row-actions">
                                            <button type="button" class="btn btn-sm" onclick={() => (editing = null)}>Mégse</button>
                                            <button type="submit" class="btn-update" disabled={!!busy || !clueDraft.trim()}>Mentés</button>
                                        </div>
                                    </form>
                                {:else}
                                    {w.clue}
                                    {#if w.example}<div class="taj-muted">„{w.example}”</div>{/if}
                                {/if}
                            </td>
                            <td>
                                {#if w.szekely}
                                    tájszó{w.familiarity ? ` (${w.familiarity})` : ""}
                                    {#if w.szotar_id && /^\d+$/.test(w.szotar_id)}
                                        · <a href="/dictionary#szavak/{w.szotar_id}">Szótár</a>
                                    {/if}
                                {:else}
                                    töltelék
                                {/if}
                            </td>
                            <td><button type="button" class="btn-update" disabled={!!busy} onclick={() => startClue(i)}>Szerk.</button></td>
                            <td><button type="button" class="btn-delete" disabled={!!busy} onclick={() => block(i)}>Tiltás</button></td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</section>

<style>
    .taj-puzzle-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
    }

    .taj-puzzle-head h3 {
        margin: 0;
    }

    .taj-muted {
        color: var(--text-muted);
        font-size: var(--text-sm);
    }

    .taj-stats {
        margin: 0.5rem 0;
    }

    .taj-actions {
        flex-wrap: wrap;
        align-items: center;
    }

    .taj-inline select {
        padding: 0.4rem 0.5rem;
        border: 1px solid var(--border-color);
        border-radius: 4px;
        background: var(--card-bg);
        color: var(--text-primary);
        font: inherit;
    }

    .taj-proverb-form {
        display: flex;
        flex-wrap: wrap;
        align-items: flex-end;
        gap: 0.5rem;
    }

    .taj-proverb-form label {
        flex-basis: 100%;
        font-weight: 500;
    }

    .taj-proverb-form input {
        flex: 1 1 18rem;
    }

    .taj-clue-form {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
    }

    .taj-row-actions {
        display: flex;
        gap: 0.5rem;
        justify-content: flex-end;
    }

    .taj-check {
        display: flex;
        gap: 0.4rem;
        align-items: center;
        font-size: var(--text-sm);
    }

    .taj-link {
        padding: 0;
        border: none;
        background: none;
        color: var(--link-color, var(--szekely-green));
        font: inherit;
        font-weight: 700;
        cursor: pointer;
    }

    tr.taj-selected td {
        background: color-mix(in srgb, var(--szekely-green) 10%, transparent);
    }

    :global(.taj-status) {
        display: inline-block;
        margin-left: 0.4rem;
        padding: 0.05rem 0.45rem;
        border-radius: 999px;
        border: 1px solid var(--border-color);
        font-size: var(--text-xs);
        font-weight: 600;
        vertical-align: middle;
    }

    :global(.taj-status--approved),
    :global(.taj-status--live) {
        border-color: var(--szekely-green);
        color: var(--szekely-green);
    }

    :global(.taj-status--needs_attention) {
        border-color: var(--szekely-red);
        color: var(--szekely-red);
    }
</style>
