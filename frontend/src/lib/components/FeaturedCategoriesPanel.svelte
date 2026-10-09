<script>
    import { apiFetch } from "$lib/api.js";

    /**
     * The Lámsza home page's category chips (under the search box): at most
     * six categories, in order. The whole list is saved at once
     * (PUT /api/admin/entry_categories/featured), so the public page never
     * shows half an edit.
     * @type {{ categories?: Array<{ id: number, name: string, parent_id?: number | null }> }}
     */
    let { categories = [] } = $props();

    const max = 6;
    let ids = $state(/** @type {number[]} */ ([]));
    let saved = $state(/** @type {number[]} */ ([]));
    let pick = $state("");
    let status = $state("");
    let error = $state("");
    let busy = $state(false);

    let byId = $derived(new Map(categories.map((c) => [c.id, c])));
    let dirty = $derived(ids.join(",") !== saved.join(","));
    let options = $derived(
        categories
            .filter((c) => !ids.includes(c.id))
            .map((c) => ({ id: c.id, label: label(c) }))
            .sort((a, b) => a.label.localeCompare(b.label, "hu")),
    );

    /** @param {{ name: string, parent_id?: number | null }} c */
    function label(c) {
        const parent = c.parent_id ? byId.get(c.parent_id) : null;
        return parent ? `${parent.name} › ${c.name}` : c.name;
    }

    /** @param {number} id */
    function nameOf(id) {
        const c = byId.get(id);
        return c ? label(c) : `#${id}`;
    }

    async function load() {
        error = "";
        try {
            const list = await apiFetch("/api/admin/entry_categories/featured");
            ids = (list || []).map((/** @type {{ id: number }} */ c) => c.id);
            saved = [...ids];
        } catch (e) {
            error = "A kiemelt kategóriák nem töltődtek be.";
        }
    }

    function add() {
        const id = Number(pick);
        if (!id || ids.length >= max || ids.includes(id)) return;
        ids = [...ids, id];
        pick = "";
        status = "";
    }

    /** @param {number} i @param {number} by */
    function move(i, by) {
        const j = i + by;
        if (j < 0 || j >= ids.length) return;
        const next = [...ids];
        [next[i], next[j]] = [next[j], next[i]];
        ids = next;
        status = "";
    }

    /** @param {number} i */
    function remove(i) {
        ids = ids.filter((_, k) => k !== i);
        status = "";
    }

    async function save() {
        busy = true;
        error = "";
        status = "";
        try {
            const list = await apiFetch("/api/admin/entry_categories/featured", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ ids }),
            });
            ids = (list || []).map((/** @type {{ id: number }} */ c) => c.id);
            saved = [...ids];
            status = "Mentve.";
        } catch (e) {
            error = `A mentés nem sikerült: ${e instanceof Error ? e.message : e}`;
        } finally {
            busy = false;
        }
    }

    $effect(() => {
        load();
    });
</script>

<section class="featured-cats" aria-labelledby="featured-cats-title">
    <h3 id="featured-cats-title" class="featured-cats__title">Kiemelt kategóriák a kezdőlapon</h3>
    <p class="admin-info">
        A Lámsza kezdőlapján a keresőmező alatt, ebben a sorrendben. Legfeljebb {max}; alkategória is
        lehet.
    </p>

    {#if ids.length}
        <ol class="featured-cats__list">
            {#each ids as id, i (id)}
                <li class="featured-cats__row">
                    <span class="featured-cats__name">{nameOf(id)}</span>
                    <span class="featured-cats__buttons">
                        <button type="button" class="btn" onclick={() => move(i, -1)} disabled={i === 0} aria-label="Feljebb">↑</button>
                        <button type="button" class="btn" onclick={() => move(i, 1)} disabled={i === ids.length - 1} aria-label="Lejjebb">↓</button>
                        <button type="button" class="btn-delete" onclick={() => remove(i)}>Eltávolítás</button>
                    </span>
                </li>
            {/each}
        </ol>
    {:else}
        <p class="featured-cats__empty">Nincs kiemelt kategória: a kezdőlapon nem jelenik meg chip.</p>
    {/if}

    <div class="featured-cats__add">
        <label for="featured-cats-pick">Kategória hozzáadása</label>
        <select id="featured-cats-pick" bind:value={pick} disabled={ids.length >= max}>
            <option value="">Válassz...</option>
            {#each options as o (o.id)}
                <option value={o.id}>{o.label}</option>
            {/each}
        </select>
        <button type="button" class="btn" onclick={add} disabled={!pick || ids.length >= max}>Hozzáadás</button>
    </div>

    <div class="featured-cats__actions">
        <button type="button" class="admin-submit-btn" onclick={save} disabled={busy || !dirty}>Mentés</button>
        {#if status}<span class="featured-cats__status" role="status">{status}</span>{/if}
    </div>
    {#if error}
        <div class="info-box error" role="alert"><p>{error}</p></div>
    {/if}
</section>

<style>
    .featured-cats {
        display: grid;
        /* A long category name in the select must not widen the panel. */
        grid-template-columns: minmax(0, 1fr);
        gap: 0.75rem;
        margin-block: 1rem 1.5rem;
        padding: 1rem;
        border: 1px solid var(--border-color);
        border-radius: 12px;
    }
    .featured-cats__title {
        margin: 0;
    }
    .featured-cats .admin-info {
        margin: 0;
    }
    .featured-cats__list {
        display: grid;
        gap: 0.5rem;
        margin: 0;
        padding-left: 1.5rem;
    }
    .featured-cats__row {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.5rem;
    }
    .featured-cats__name {
        flex: 1 1 12rem;
        min-width: 0;
    }
    .featured-cats__buttons {
        display: flex;
        gap: 0.5rem;
        flex: none;
    }
    /* The admin form field look (admin.css .admin-form select). */
    .featured-cats__add select {
        flex: 1 1 14rem;
        min-width: 0;
        max-width: 24rem;
        box-sizing: border-box;
        padding: 0.6rem 0.8rem;
        border: 1px solid var(--border-color);
        border-radius: 4px;
        font-size: var(--text-base);
        font-family: inherit;
        background: var(--card-bg);
        color: var(--text-primary);
    }
    .featured-cats__empty {
        margin: 0;
        color: var(--text-muted);
    }
    .featured-cats__add,
    .featured-cats__actions {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.5rem;
    }
    .featured-cats__status {
        color: var(--text-muted);
    }
</style>
