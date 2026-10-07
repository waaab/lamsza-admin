<script>
    /**
     * Szótár's daily proverbs (Mondások): one per calendar day (moved from
     * Szótár's /admin/mondasok). A second one on a taken day gets the
     * backend's 409 message.
     *
     * @type {{ onChanged?: () => void }}
     */
    let { onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import AdminPlusIcon from "$lib/components/admin/AdminPlusIcon.svelte";
    import HuDateInput from "$lib/components/HuDateInput.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { errorText, formatHuDate, localISODate, sectionFetch } from "$lib/sectionApi.js";

    /** @typedef {{ id: number, text: string, meaning: string, display_date: string }} Proverb */
    /** @type {Proverb[]} */
    let proverbs = $state([]);
    let loaded = $state(false);
    let error = $state("");
    let notice = $state("");
    let createOpen = $state(false);
    let draft = $state({ text: "", meaning: "", display_date: localISODate() });
    let createError = $state("");
    /** @type {Proverb | null} */
    let editing = $state(null);
    let editError = $state("");
    let saving = $state(false);

    async function load() {
        try {
            const data = await sectionFetch("dictionary", "proverbs");
            proverbs = data?.proverbs || [];
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
            draft = { text: "", meaning: "", display_date: localISODate() };
            createOpen = false;
            proverbs = [item, ...proverbs];
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
            proverbs = proverbs.map((row) => (row.id === item.id ? item : row));
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
    A Szótár kezdőlapján az adott naptári napra beütemezett mondás jelenik meg. Egy napra csak egy mondás
    állítható be.
</p>

<details class="admin-create-panel" bind:open={createOpen}>
    <summary class="admin-create-summary"><span>Új mondás hozzáadása</span><AdminPlusIcon /></summary>
    <form class="admin-form admin-create-form" onsubmit={create}>
        <label for="dict-proverb-text">Mondás</label>
        <textarea id="dict-proverb-text" name="text" bind:value={draft.text} required rows="3"></textarea>
        <label for="dict-proverb-date">Megjelenés napja</label>
        <HuDateInput id="dict-proverb-date" name="display_date" bind:value={draft.display_date} required />
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

<p class="admin-status">{loaded ? `${proverbs.length} mondás.` : "Betöltés…"}</p>
<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            <tr>
                <th>Megjelenés napja</th>
                <th>Mondás</th>
                <th>Jelentés</th>
                <th class="admin-table-col--action">Szerk.</th>
                <th class="admin-table-col--action">Törlés</th>
            </tr>
        </thead>
        <tbody>
            {#each proverbs as p (p.id)}
                <tr>
                    <td>{formatHuDate(p.display_date)}</td>
                    <td>{p.text}</td>
                    <td>{p.meaning || "-"}</td>
                    <td><button type="button" class="btn-update" onclick={() => ((editing = { ...p }), (editError = ""))}>Szerk.</button></td>
                    <td><button type="button" class="btn-delete" onclick={() => remove(p)}>Törlés</button></td>
                </tr>
            {:else}
                <tr><td colspan="5">{loaded ? "Még nincs mondás." : "Betöltés…"}</td></tr>
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
                <HuDateInput id="dict-eproverb-date" name="display_date" bind:value={editing.display_date} required />
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
</style>
