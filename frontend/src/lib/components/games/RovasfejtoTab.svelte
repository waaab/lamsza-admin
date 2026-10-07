<script>
    /**
     * Rovásfejtő puzzles: create, edit, soft-delete (a deleted puzzle stays in
     * the list, marked "törölve"). Moved from Játszótér's /admin/rovasfejto.
     *
     * @type {{ onChanged?: () => void }}
     */
    let { onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import AdminPlusIcon from "$lib/components/admin/AdminPlusIcon.svelte";
    import RovasfejtoForm from "$lib/components/games/RovasfejtoForm.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    /** @type {any[]} */
    let puzzles = $state([]);
    /** @type {{ id: string, label_hu: string }[]} */
    let categories = $state([]);
    /** @type {{ level: number, label_hu: string }[]} */
    let difficulties = $state([]);
    let loaded = $state(false);
    let error = $state("");
    let notice = $state("");
    let createOpen = $state(false);
    /** @type {any} */
    let editing = $state(null);

    async function load() {
        try {
            const [meta, list] = await Promise.all([
                sectionFetch("games", "rovasfejto/meta"),
                sectionFetch("games", "rovasfejto/puzzles"),
            ]);
            categories = meta?.categories ?? [];
            difficulties = meta?.difficulties ?? [];
            puzzles = list?.puzzles ?? [];
            error = "";
        } catch (err) {
            error = errorText(err, "A feladványok nem tölthetők be.");
        } finally {
            loaded = true;
        }
    }

    onMount(load);

    function saved(/** @type {string} */ id, /** @type {boolean} */ created) {
        notice = created ? `Létrehozva: ${id}.` : `Mentve: ${id}.`;
        editing = null;
        if (created) createOpen = false;
        load();
        onChanged();
    }

    async function remove(/** @type {any} */ p) {
        if (!(await confirmDialog(`Töröljük a(z) ${p.id} feladványt?`, { yesLabel: "Törlés", destructive: true }))) return;
        try {
            await sectionFetch("games", `rovasfejto/puzzles/${p.id}`, { method: "DELETE" });
            notice = `Törölve: ${p.id}.`;
            await load();
            onChanged();
        } catch (err) {
            error = errorText(err, "A törlés nem sikerült.");
        }
    }

    const difficultyLabel = (/** @type {number} */ level) =>
        difficulties.find((d) => d.level === level)?.label_hu ?? `Szint ${level}`;
    const active = $derived(puzzles.filter((p) => p.enabled).length);
</script>

<p class="admin-info">A nehézség és a típus kézzel állítható, nem a szöveg hosszából számolódik.</p>

{#if loaded && categories.length}
    <details class="admin-create-panel" bind:open={createOpen}>
        <summary class="admin-create-summary"><span>Új feladvány hozzáadása</span><AdminPlusIcon /></summary>
        <div class="create-wrap">
            <RovasfejtoForm {categories} {difficulties} idPrefix="rovas-new" onSaved={saved} />
        </div>
    </details>
{/if}

{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}
{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

<p class="admin-status">{loaded ? `${active} aktív, ${puzzles.length - active} törölt feladvány.` : "Betöltés…"}</p>
<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            <tr>
                <th>ID</th>
                <th>Szöveg</th>
                <th>Típus</th>
                <th>Nehézség</th>
                <th class="admin-table-col--action">Szerk.</th>
                <th class="admin-table-col--action">Törlés</th>
            </tr>
        </thead>
        <tbody>
            {#each puzzles as p (p.id)}
                <tr class:admin-row-muted={!p.enabled}>
                    <td><code>{p.id}</code></td>
                    <td>{p.text}</td>
                    <td>{p.category_label}</td>
                    <td>{difficultyLabel(p.difficulty)}</td>
                    {#if p.enabled}
                        <td><button type="button" class="btn-update" onclick={() => (editing = p)}>Szerk.</button></td>
                        <td><button type="button" class="btn-delete" onclick={() => remove(p)}>Törlés</button></td>
                    {:else}
                        <td class="admin-table-col--action admin-table-col--action--muted" colspan="2">törölve</td>
                    {/if}
                </tr>
            {:else}
                <tr><td colspan="6">{loaded ? "Még nincs feladvány." : "Betöltés…"}</td></tr>
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
        aria-labelledby="rovas-edit-title"
        onclick={(e) => e.target === e.currentTarget && (editing = null)}
        onkeydown={(e) => e.key === "Escape" && (editing = null)}
    >
        <div class="link-dialog admin-modal">
            <h3 id="rovas-edit-title">Feladvány szerkesztése: {editing.id}</h3>
            {#key editing.id}
                <RovasfejtoForm
                    puzzle={editing}
                    {categories}
                    {difficulties}
                    idPrefix="rovas-edit"
                    onSaved={saved}
                    onCancel={() => (editing = null)}
                />
            {/key}
        </div>
    </div>
{/if}

<style>
    .admin-info {
        color: var(--text-faint, #666);
        margin-bottom: 1rem;
    }
    .create-wrap :global(.admin-form) {
        margin: 0;
        border: none;
        border-radius: 0;
    }
</style>
