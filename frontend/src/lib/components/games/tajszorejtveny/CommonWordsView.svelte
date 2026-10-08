<script>
    /**
     * Tájszórejtvény filler words (spec §11.2): the everyday words that fill
     * the grid around the Székely ones, each with its clue. Add one, edit its
     * clue, switch it off; a word is never deleted, so old puzzles stay as
     * they were. The clue must not contain the word.
     */
    import { onMount } from "svelte";
    import AdminPaginationBar from "$lib/components/admin/AdminPaginationBar.svelte";
    import AdminPlusIcon from "$lib/components/admin/AdminPlusIcon.svelte";
    import { adminPageSlice } from "$lib/adminPageSlice.js";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    /** @type {{ id: number, word: string, clue: string, enabled: boolean }[]} */
    let words = $state([]);
    let q = $state("");
    let loading = $state(true);
    let error = $state("");
    let notice = $state("");
    let page = $state(1);
    let createOpen = $state(false);
    let newWord = $state("");
    let newClue = $state("");
    /** @type {number | null} */
    let editingId = $state(null);
    let clueDraft = $state("");
    let busy = $state(false);

    const shown = $derived(q.trim() ? words.filter((w) => w.word.toLowerCase().includes(q.trim().toLowerCase())) : words);
    const pg = $derived(/** @type {ReturnType<typeof adminPageSlice> & { rows: any[] }} */ (adminPageSlice(shown, page)));
    const enabledCount = $derived(words.filter((w) => w.enabled).length);

    async function load() {
        loading = true;
        try {
            const data = await sectionFetch("games", "tajszorejtveny/common-words");
            words = data?.words ?? [];
            error = "";
        } catch (err) {
            error = errorText(err, "A töltelékszavak nem tölthetők be.");
        } finally {
            loading = false;
        }
    }

    onMount(load);

    /** @param {() => Promise<unknown>} fn @param {string} done */
    async function act(fn, done) {
        if (busy) return false;
        busy = true;
        error = "";
        notice = "";
        try {
            await fn();
            notice = done;
            await load();
            return true;
        } catch (err) {
            error = errorText(err, "A mentés nem sikerült.");
            return false;
        } finally {
            busy = false;
        }
    }

    async function add() {
        const word = newWord.trim();
        const ok = await act(
            () => sectionFetch("games", "tajszorejtveny/common-words", { method: "POST", body: { word, clue: newClue.trim() } }),
            `Hozzáadva: ${word.toUpperCase()}.`,
        );
        if (ok) {
            newWord = "";
            newClue = "";
        }
    }

    /** @param {{ id: number, word: string, clue: string, enabled: boolean }} w @param {{ clue?: string, enabled?: boolean }} change */
    async function update(w, change) {
        const ok = await act(
            () =>
                sectionFetch("games", `tajszorejtveny/common-words/${w.id}`, {
                    method: "PUT",
                    body: { clue: change.clue ?? w.clue, enabled: change.enabled ?? w.enabled },
                }),
            change.enabled === false ? `Kikapcsolva: ${w.word}.` : change.enabled === true ? `Bekapcsolva: ${w.word}.` : `Mentve: ${w.word}.`,
        );
        if (ok) editingId = null;
    }
</script>

<p class="admin-info">{enabledCount} bekapcsolt töltelékszó ({words.length} összesen). A 7-8 betűs szavak segítik a legjobban a rácsot.</p>

<details class="admin-create-panel" bind:open={createOpen}>
    <summary class="admin-create-summary"><span>Új töltelékszó</span><AdminPlusIcon /></summary>
    <form
        class="taj-add-form"
        onsubmit={(e) => {
            e.preventDefault();
            add();
        }}
    >
        <label
            >Szó
            <input class="admin-search-input" bind:value={newWord} maxlength="20" required /></label
        >
        <label
            >Meghatározás
            <input class="admin-search-input" bind:value={newClue} maxlength="60" required /></label
        >
        <button type="submit" class="admin-submit-btn" disabled={busy || !newWord.trim() || !newClue.trim()}>Hozzáadás</button>
    </form>
</details>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}
{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}

<div class="admin-table-toolbar">
    <label class="admin-search-label"
        >Keresés
        <input type="search" class="admin-search-input" bind:value={q} oninput={() => (page = 1)} placeholder="Szó…" /></label
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
                <th class="admin-table-col--action">Szerk.</th>
                <th class="admin-table-col--action">Állapot</th>
            </tr>
        </thead>
        <tbody>
            {#each pg.rows as w (w.id)}
                <tr class:admin-row-muted={!w.enabled}>
                    <td><strong>{w.word}</strong></td>
                    <td>
                        {#if editingId === w.id}
                            <form
                                class="taj-inline-form"
                                onsubmit={(e) => {
                                    e.preventDefault();
                                    update(w, { clue: clueDraft.trim() });
                                }}
                            >
                                <input class="admin-search-input" bind:value={clueDraft} maxlength="60" aria-label="Meghatározás" />
                                <button type="button" class="btn btn-sm" onclick={() => (editingId = null)}>Mégse</button>
                                <button type="submit" class="btn-update" disabled={busy || !clueDraft.trim()}>Mentés</button>
                            </form>
                        {:else}
                            {w.clue}
                        {/if}
                    </td>
                    <td>
                        <button type="button" class="btn-update" disabled={busy} onclick={() => ((editingId = w.id), (clueDraft = w.clue))}
                            >Szerk.</button
                        >
                    </td>
                    <td>
                        <button type="button" class="btn btn-sm" disabled={busy} onclick={() => update(w, { enabled: !w.enabled })}
                            >{w.enabled ? "Kikapcsolás" : "Bekapcsolás"}</button
                        >
                    </td>
                </tr>
            {:else}
                <tr><td colspan="4">{loading ? "Betöltés…" : "Nincs találat."}</td></tr>
            {/each}
        </tbody>
    </table>
</div>

<style>
    .taj-add-form {
        display: flex;
        flex-wrap: wrap;
        align-items: flex-end;
        gap: 0.75rem;
        padding: 1rem 0;
    }

    .taj-add-form label {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
        flex: 1 1 12rem;
        font-weight: 500;
    }

    .taj-inline-form {
        display: flex;
        flex-wrap: wrap;
        gap: 0.4rem;
        align-items: center;
    }

    .taj-inline-form input {
        flex: 1 1 14rem;
    }
</style>
