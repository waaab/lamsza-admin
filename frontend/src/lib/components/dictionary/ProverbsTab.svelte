<script>
    /**
     * Szótár's daily proverbs (Mondások), the network's only mondás store: one
     * per Bucharest calendar day, shown on Szótár's and Lámsza's home pages.
     * Everything the main admin's Mondások page did is here: add, edit, delete,
     * search, paging, the "Mai nap" button, the "ma" row and the warning when
     * today has none. "Today" is Szótár's (R19), never the browser's. A second
     * mondás on a taken day gets the backend's 409 message.
     *
     * @type {{ onChanged?: () => void }}
     */
    let { onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import AdminPaginationBar from "$lib/components/admin/AdminPaginationBar.svelte";
    import AdminPlusIcon from "$lib/components/admin/AdminPlusIcon.svelte";
    import { adminPageSlice } from "$lib/adminPageSlice.js";
    import HuDateInput from "$lib/components/HuDateInput.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";

    /** @typedef {{ id: number, text: string, meaning: string, display_date: string }} Proverb */
    /** @type {Proverb[]} */
    let proverbs = $state([]);
    let loaded = $state(false);
    let error = $state("");
    let notice = $state("");
    let createOpen = $state(false);
    /** Szótár's today, a Bucharest day (lamsza WAYS_OF_WORKING R19), never the browser's. */
    let today = $state("");
    let draft = $state({ text: "", meaning: "", display_date: "" });
    let createError = $state("");
    /** @type {Proverb | null} */
    let editing = $state(null);
    let editError = $state("");
    let saving = $state(false);
    let search = $state("");
    let page = $state(1);

    /** Search by text, meaning, ID or date, as the main admin's page did. */
    const filtered = $derived.by(() => {
        const q = search.trim().toLowerCase();
        if (!q) return proverbs;
        return proverbs.filter((p) =>
            [p.text, p.meaning, p.id, p.display_date].some((f) => String(f ?? "").toLowerCase().includes(q)),
        );
    });
    const pg = $derived(adminPageSlice(filtered, page));
    /** Newest display date first, as Szótár lists them. */
    const byDate = (/** @type {Proverb[]} */ rows) =>
        [...rows].sort((a, b) => b.display_date.localeCompare(a.display_date) || b.id - a.id);
    const todayCount = $derived(today ? proverbs.filter((p) => p.display_date === today).length : 1);

    async function load() {
        try {
            const [data, stats] = await Promise.all([
                sectionFetch("dictionary", "proverbs"),
                sectionFetch("dictionary", "stats"),
            ]);
            proverbs = data?.proverbs || [];
            today = stats?.today || "";
            if (!draft.display_date) draft.display_date = today;
            error = "";
        } catch (err) {
            error = errorText(err, "A mondások most nem érhetők el.");
        } finally {
            loaded = true;
        }
    }

    onMount(load);

    async function create(/** @type {SubmitEvent} */ e) {
        e.preventDefault();
        saving = true;
        createError = "";
        try {
            const item = await sectionFetch("dictionary", "proverbs", { method: "POST", body: draft });
            notice = "Mondás hozzáadva.";
            draft = { text: "", meaning: "", display_date: today };
            createOpen = false;
            proverbs = byDate([item, ...proverbs]);
            onChanged();
        } catch (err) {
            createError = errorText(err, "A mentés nem sikerült.");
        } finally {
            saving = false;
        }
    }

    async function update(/** @type {SubmitEvent} */ e) {
        e.preventDefault();
        if (!editing) return;
        saving = true;
        editError = "";
        try {
            const { id, text, meaning, display_date } = editing;
            const item = await sectionFetch("dictionary", `proverbs/${id}`, {
                method: "PUT",
                body: { text, meaning, display_date },
            });
            proverbs = byDate(proverbs.map((row) => (row.id === item.id ? item : row)));
            onChanged();
            notice = "Mondás mentve.";
            editing = null;
        } catch (err) {
            editError = errorText(err, "A mentés nem sikerült.");
        } finally {
            saving = false;
        }
    }

    async function remove(/** @type {Proverb} */ p) {
        if (!(await confirmDialog("Biztosan törlöd ezt a mondást?", { yesLabel: "Törlés", destructive: true }))) return;
        try {
            await sectionFetch("dictionary", `proverbs/${p.id}`, { method: "DELETE" });
            proverbs = proverbs.filter((item) => item.id !== p.id);
            notice = "Mondás törölve.";
            onChanged();
        } catch (err) {
            error = errorText(err, "A törlés nem sikerült.");
        }
    }
</script>

<p class="admin-info">
    A Szótár és a Lámsza kezdőlapján az adott <strong>naptári napra</strong> (bukaresti idő szerint) beütemezett
    mondás jelenik meg. Egy napra csak egy mondás állítható be. Ha a napra nincs mondás, a kezdőlapokon nem jelenik
    meg mondás-blokk.
</p>
{#if loaded && todayCount === 0}
    <div class="info-box warning" role="status">
        <p>
            Ma ({formatHuDate(today)}) nincs beütemezett mondás, ezért a kezdőlapokon a mondás-blokk rejtve marad.
            Állíts be egy mondást a mai napra.
        </p>
    </div>
{/if}

<details class="admin-create-panel" bind:open={createOpen}>
    <summary class="admin-create-summary"><span>Új mondás hozzáadása</span><AdminPlusIcon /></summary>
    <form class="admin-form admin-create-form" onsubmit={create}>
        <label for="dict-proverb-text">Mondás</label>
        <textarea id="dict-proverb-text" name="text" bind:value={draft.text} required rows="3"></textarea>
        <label for="dict-proverb-date">Megjelenés napja</label>
        <div class="admin-date-field">
            <HuDateInput id="dict-proverb-date" name="display_date" bind:value={draft.display_date} required />
            <button type="button" class="btn btn-sm" disabled={!today} onclick={() => (draft.display_date = today)}
                >Mai nap</button
            >
        </div>
        <label for="dict-proverb-meaning">Jelentés (opcionális)</label>
        <textarea id="dict-proverb-meaning" name="meaning" bind:value={draft.meaning} rows="3"></textarea>
        {#if createError}
            <div class="info-box error" role="alert"><p>{createError}</p></div>
        {/if}
        <div>
            <button type="submit" class="admin-submit-btn" disabled={saving || !draft.text.trim() || !draft.display_date}
                >Hozzáadás</button
            >
        </div>
    </form>
</details>

{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}
{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

<div class="admin-table-toolbar">
    <label class="admin-search-label"
        >Keresés
        <input
            id="dict-proverb-search"
            name="search"
            type="search"
            class="admin-search-input"
            bind:value={search}
            oninput={() => (page = 1)}
            placeholder="Szöveg vagy ID…"
        /></label
    >
</div>
{#snippet pager()}
    <AdminPaginationBar
        total={pg.total}
        page={pg.page}
        totalPages={pg.totalPages}
        from={pg.from}
        to={pg.to}
        on:prev={() => (page = Math.max(1, page - 1))}
        on:next={() => (page = Math.min(pg.totalPages, page + 1))}
    />
{/snippet}
{@render pager()}
<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            <tr>
                <th>ID</th>
                <th>Megjelenés napja</th>
                <th>Mondás</th>
                <th>Jelentés</th>
                <th class="admin-table-col--action">Szerk.</th>
                <th class="admin-table-col--action">Törlés</th>
            </tr>
        </thead>
        <tbody>
            {#each pg.rows as p (p.id)}
                {@const isToday = p.display_date === today}
                <tr class:admin-row-today={isToday}>
                    <td>{p.id}</td>
                    <td>
                        {formatHuDate(p.display_date)}
                        {#if isToday}<span class="admin-date-today-badge">ma</span>{/if}
                    </td>
                    <td>{p.text}</td>
                    <td>{p.meaning || "-"}</td>
                    <td><button type="button" class="btn-update" onclick={() => ((editing = { ...p }), (editError = ""))}>Szerk.</button></td>
                    <td><button type="button" class="btn-delete" onclick={() => remove(p)}>Törlés</button></td>
                </tr>
            {:else}
                <tr><td colspan="6">{!loaded ? "Betöltés…" : search.trim() ? "Nincs találat." : "Még nincs mondás."}</td></tr>
            {/each}
        </tbody>
    </table>
</div>
{@render pager()}

{#if editing}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
        class="link-dialog-overlay"
        role="dialog"
        tabindex="-1"
        aria-labelledby="dict-proverb-edit-title"
        onclick={(e) => e.target === e.currentTarget && (editing = null)}
        onkeydown={(e) => e.key === "Escape" && (editing = null)}
    >
        <div class="link-dialog admin-modal">
            <h3 id="dict-proverb-edit-title">Mondás szerkesztése</h3>
            <form class="admin-form" onsubmit={update}>
                <label for="dict-eproverb-text">Mondás</label>
                <textarea id="dict-eproverb-text" name="text" bind:value={editing.text} required rows="3"></textarea>
                <label for="dict-eproverb-date">Megjelenés napja</label>
                <div class="admin-date-field">
                    <HuDateInput id="dict-eproverb-date" name="display_date" bind:value={editing.display_date} required />
                    <button
                        type="button"
                        class="btn btn-sm"
                        disabled={!today}
                        onclick={() => editing && (editing.display_date = today)}>Mai nap</button
                    >
                </div>
                <label for="dict-eproverb-meaning">Jelentés (opcionális)</label>
                <textarea id="dict-eproverb-meaning" name="meaning" bind:value={editing.meaning} rows="3"></textarea>
                {#if editError}
                    <div class="info-box error" role="alert"><p>{editError}</p></div>
                {/if}
                <div class="admin-modal-actions">
                    <button type="submit" class="admin-submit-btn" disabled={saving}>Mentés</button>
                    <button type="button" class="btn-delete" onclick={() => (editing = null)}>Mégse</button>
                </div>
            </form>
        </div>
    </div>
{/if}

<style>
    .admin-info {
        color: var(--text-faint, #666);
        margin-bottom: 1rem;
    }
    /* The main admin's Mondások look: the date field with "Mai nap", the "ma"
       badge and the highlighted row. */
    .admin-date-field {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex-wrap: wrap;
    }
    .admin-date-field :global(input) {
        min-width: 12rem;
    }
    .admin-date-today-badge {
        display: inline-block;
        margin-left: 0.35rem;
        padding: 0.1rem 0.45rem;
        border-radius: 999px;
        background: var(--szekely-red, #c8102e);
        color: #fff;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.03em;
    }
    tr.admin-row-today {
        background: color-mix(in srgb, var(--szekely-green, #2f4f4f) 10%, transparent);
    }
</style>
