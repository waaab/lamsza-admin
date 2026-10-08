<script>
    /**
     * Tájszórejtvény word metadata (spec §11.1, §16.2.5, §16.2.7): every Szótár
     * word with what the game keeps about it. Familiarity (1 well known ... 3
     * rare; empty = the default rule), a game clue that replaces the
     * definition in the grid, an example sentence used only once reviewed
     * (Szótár's own example comes first), and a block. The definitions
     * themselves are fixed in /dictionary.
     */
    import { onMount } from "svelte";
    import AdminPaginationBar from "$lib/components/admin/AdminPaginationBar.svelte";
    import { adminPageSlice } from "$lib/adminPageSlice.js";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    /** @type {any[]} */
    let words = $state([]);
    let q = $state("");
    let filter = $state("pool");
    let loading = $state(true);
    let error = $state("");
    let notice = $state("");
    let page = $state(1);
    /** @type {any} */
    let editing = $state(null);
    let saving = $state(false);
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let searchTimer;

    const shown = $derived(
        words.filter((w) =>
            filter === "pool"
                ? w.in_pool
                : filter === "unset"
                  ? w.in_pool && !w.familiarity
                  : filter === "examples"
                    ? w.example && !w.example_reviewed
                    : filter === "blocked"
                      ? w.blocked
                      : true,
        ),
    );
    const pg = $derived(/** @type {ReturnType<typeof adminPageSlice> & { rows: any[] }} */ (adminPageSlice(shown, page)));

    async function load() {
        loading = true;
        try {
            const data = await sectionFetch("games", `tajszorejtveny/words${q.trim() ? `?q=${encodeURIComponent(q.trim())}` : ""}`);
            words = data?.words ?? [];
            error = "";
        } catch (err) {
            error = errorText(err, "A szavak nem tölthetők be.");
            words = [];
        } finally {
            loading = false;
        }
    }

    onMount(load);

    function onSearchInput() {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => {
            page = 1;
            load();
        }, 300);
    }

    function edit(/** @type {any} */ w) {
        editing = {
            szotar_id: w.szotar_id,
            headword: w.headword,
            familiarity: w.familiarity || 0,
            clue_override: w.clue_override ?? "",
            example: w.example ?? "",
            example_reviewed: !!w.example_reviewed,
            blocked: !!w.blocked,
            definition: w.definition,
            szotar_example: w.szotar_example ?? "",
        };
        notice = "";
    }

    async function save() {
        if (!editing || saving) return;
        saving = true;
        error = "";
        try {
            const { definition: _d, szotar_example: _e, ...body } = editing;
            await sectionFetch("games", "tajszorejtveny/words", { method: "PUT", body: { ...body, familiarity: Number(body.familiarity) || 0 } });
            notice = `Mentve: ${editing.headword}.`;
            editing = null;
            await load();
        } catch (err) {
            error = errorText(err, "A mentés nem sikerült.");
        } finally {
            saving = false;
        }
    }

    const famLabel = (/** @type {number} */ f) => ({ 1: "1 ismert", 2: "2 közepes", 3: "3 ritka" })[f] ?? "";
</script>

<p class="admin-info">
    Az ismertség dönti el, melyik szinten kerülhet elő a szó: az 1-es a Könnyű szinthez kell. Üresen az alapszabály dönt (a
    táblázatban halványan).
</p>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}
{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}

<div class="admin-table-toolbar">
    <label class="admin-search-label"
        >Keresés
        <input type="search" class="admin-search-input" bind:value={q} oninput={onSearchInput} placeholder="Címszó…" /></label
    >
    <label class="admin-search-label"
        >Szűrő
        <select class="admin-search-input" bind:value={filter} onchange={() => (page = 1)}>
            <option value="pool">A játékban használható</option>
            <option value="unset">Használható, ismertség nélkül</option>
            <option value="examples">Átnézendő példamondat</option>
            <option value="blocked">Letiltott</option>
            <option value="all">Mind</option>
        </select></label
    >
</div>
<AdminPaginationBar
    total={pg.total}
    page={pg.page}
    totalPages={pg.totalPages}
    from={pg.from}
    to={pg.to}
    on:prev={() => (page = Math.max(1, page - 1))}
    on:next={() => (page = Math.min(pg.totalPages, page + 1))}
/>
<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            <tr>
                <th>Szó</th>
                <th>Meghatározás</th>
                <th>Ismertség</th>
                <th>Játékbeli meghatározás</th>
                <th>Példamondat</th>
                <th class="admin-table-col--action">Szerk.</th>
            </tr>
        </thead>
        <tbody>
            {#each pg.rows as w (w.szotar_id || w.headword)}
                <tr class:admin-row-muted={w.blocked}>
                    <td>
                        <strong>{w.headword}</strong>
                        {#if /^\d+$/.test(w.szotar_id)}<div><a href="/dictionary#szavak/{w.szotar_id}">Szótár</a></div>{/if}
                        {#if w.blocked}<div class="taj-muted">letiltva</div>{:else if !w.in_pool}<div class="taj-muted">nem használható</div>{/if}
                    </td>
                    <td>{w.definition}</td>
                    <td>
                        {#if w.familiarity}{famLabel(w.familiarity)}{:else if w.effective}<span class="taj-muted">{famLabel(w.effective)}</span>{/if}
                    </td>
                    <td>{w.clue_override}</td>
                    <td>
                        {#if w.szotar_example}<div>„{w.szotar_example}” <span class="taj-muted">(Szótár)</span></div>{/if}
                        {#if w.example}<div>„{w.example}” <span class="taj-muted">({w.example_reviewed ? "átnézve" : "nincs átnézve"})</span></div>{/if}
                    </td>
                    <td><button type="button" class="btn-update" onclick={() => edit(w)}>Szerk.</button></td>
                </tr>
            {:else}
                <tr><td colspan="6">{loading ? "Betöltés…" : "Nincs találat."}</td></tr>
            {/each}
        </tbody>
    </table>
</div>

{#if editing}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
        class="link-dialog-overlay"
        role="dialog"
        tabindex="-1"
        aria-labelledby="taj-word-edit-title"
        onclick={(e) => e.target === e.currentTarget && (editing = null)}
        onkeydown={(e) => e.key === "Escape" && (editing = null)}
    >
        <form
            class="link-dialog admin-modal taj-word-form"
            onsubmit={(e) => {
                e.preventDefault();
                save();
            }}
        >
            <h3 id="taj-word-edit-title">{editing.headword}</h3>
            <p class="taj-muted">{editing.definition}</p>
            <label for="taj-w-fam"
                >Ismertség
                <select id="taj-w-fam" bind:value={editing.familiarity}>
                    <option value={0}>Alapszabály szerint</option>
                    <option value={1}>1: ismert</option>
                    <option value={2}>2: közepes</option>
                    <option value={3}>3: ritka</option>
                </select></label
            >
            <label for="taj-w-clue"
                >Játékbeli meghatározás (üresen a Szótár meghatározása)
                <input id="taj-w-clue" bind:value={editing.clue_override} maxlength="60" /></label
            >
            {#if editing.szotar_example}
                <p class="taj-muted">A Szótár példamondata: „{editing.szotar_example}” (ez kerül a játékba).</p>
            {/if}
            <label for="taj-w-example"
                >Saját példamondat
                <textarea id="taj-w-example" rows="2" bind:value={editing.example} maxlength="300"></textarea></label
            >
            <label class="taj-check"><input type="checkbox" bind:checked={editing.example_reviewed} /> A példamondat átnézve, használható</label>
            <label class="taj-check"><input type="checkbox" bind:checked={editing.blocked} /> Letiltva: nem kerül rejtvénybe</label>
            <div class="link-dialog-actions">
                <button type="button" class="link-dialog-cancel" onclick={() => (editing = null)}>Mégse</button>
                <button type="submit" class="link-dialog-submit" disabled={saving}>{saving ? "Mentés…" : "Mentés"}</button>
            </div>
        </form>
    </div>
{/if}

<style>
    .taj-muted {
        color: var(--text-muted);
        font-size: var(--text-sm);
    }

    .taj-word-form label {
        display: block;
        margin-bottom: 0.75rem;
        font-weight: 500;
    }

    .taj-word-form select,
    .taj-word-form input:not([type="checkbox"]),
    .taj-word-form textarea {
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

    .taj-word-form .taj-check {
        display: flex;
        gap: 0.5rem;
        align-items: center;
        font-weight: 400;
    }
</style>
