<script>
    import { SvelteMap, SvelteSet } from "svelte/reactivity";

    /**
     * @type {{
     *   parents?: Array<{ id: number, name: string }>,
     *   children?: Array<{ id: number, name: string, parent_id?: number | null }>,
     *   primary?: number | string | null,
     *   extra?: Array<number | string>,
     *   primaryInputId?: string,
     * }}
     */
    let {
        parents = [],
        children = [],
        primary = $bindable(0),
        extra = $bindable([]),
        primaryInputId = "category-primary",
    } = $props();

    const maxTotal = 5;
    let pickValue = $state("");

    let filled = $derived(
        (Array.isArray(extra) ? extra : [])
            .map((id) => Number(id))
            .filter((id) => id > 0 && id !== Number(primary)),
    );
    let selectedIds = $derived(
        [Number(primary), ...filled].filter((id) => id > 0),
    );
    let nameById = $derived.by(() => {
        /** @type {SvelteMap<number, string>} */
        const map = new SvelteMap();
        for (const row of children) map.set(Number(row.id), String(row.name));
        return map;
    });
    let selectedRows = $derived(
        selectedIds.map((id, index) => ({
            id,
            name: nameById.get(id) || String(id),
            primary: index === 0,
        })),
    );
    let availableGroups = $derived.by(() => {
        const skip = new SvelteSet(selectedIds);
        return parents
            .map((parent) => ({
                parent,
                children: children.filter(
                    (row) =>
                        Number(row.parent_id) === Number(parent.id) &&
                        !skip.has(Number(row.id)),
                ),
            }))
            .filter((group) => group.children.length > 0);
    });
    let canAdd = $derived(selectedIds.length < maxTotal && availableGroups.length > 0);

    /** @param {string} value */
    function addCategory(value) {
        const id = Number(value) || 0;
        pickValue = "";
        if (id <= 0 || selectedIds.includes(id) || selectedIds.length >= maxTotal) return;
        if (!Number(primary)) {
            primary = id;
            return;
        }
        extra = [...filled, id];
    }

    /** @param {number} id */
    function removeCategory(id) {
        if (Number(primary) === id) {
            const [next, ...rest] = filled;
            primary = next || 0;
            extra = rest;
            return;
        }
        extra = filled.filter((item) => item !== id);
    }

    /** @param {number} id */
    function makePrimary(id) {
        if (!id || Number(primary) === id) return;
        const others = selectedIds.filter((item) => item !== id);
        primary = id;
        extra = others;
    }
</script>

<label for={primaryInputId}>Kategória <span class="field-required" aria-hidden="true">*</span></label>
<select
    id={primaryInputId}
    required={selectedIds.length === 0}
    disabled={!canAdd}
    value={pickValue}
    onchange={(event) => addCategory(event.currentTarget.value)}
>
    <option value="">Válassz...</option>
    {#each availableGroups as group (group.parent.id)}
        <optgroup label={group.parent.name}>
            {#each group.children as child (child.id)}
                <option value={child.id}>{child.name}</option>
            {/each}
        </optgroup>
    {/each}
</select>

<p class="category-multi-select__summary">
    {selectedIds.length}/{maxTotal} kategória
    {#if selectedIds.length === 0}
        <span class="category-multi-select__hint"> - az első választás az elsődleges.</span>
    {/if}
</p>

{#if selectedRows.length > 0}
    <ul class="category-multi-select__list">
        {#each selectedRows as row (row.id)}
            <li class="category-multi-select__item">
                <span class="category-multi-select__name">
                    {row.name}
                    {#if row.primary}
                        <span class="category-multi-select__primary">elsődleges</span>
                    {/if}
                </span>
                <span class="category-multi-select__actions">
                    {#if !row.primary}
                        <button
                            type="button"
                            class="btn btn-xs"
                            onclick={() => makePrimary(row.id)}
                        >Elsődleges</button>
                    {/if}
                    <button
                        type="button"
                        class="btn btn-xs"
                        onclick={() => removeCategory(row.id)}
                    >Eltávolítás</button>
                </span>
            </li>
        {/each}
    </ul>
{/if}

<style>
    .category-multi-select__summary {
        margin: 0.35rem 0 0;
        font-size: var(--text-sm);
        color: var(--text-muted);
    }

    .category-multi-select__hint {
        color: var(--text-muted);
    }

    .category-multi-select__list {
        list-style: none;
        margin: 0.5rem 0 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
    }

    .category-multi-select__item {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: space-between;
        gap: 0.5rem;
        padding: 0.35rem 0.5rem;
        border: 1px solid var(--border-color);
        border-radius: 6px;
        background: var(--bg-body);
        color: var(--text-primary);
    }

    .category-multi-select__name {
        font-size: var(--text-sm);
        color: var(--text-primary);
    }

    .category-multi-select__primary {
        margin-left: 0.35rem;
        font-size: var(--text-xs);
        text-transform: uppercase;
        letter-spacing: 0.02em;
        color: var(--szekely-red);
    }

    .category-multi-select__actions {
        display: flex;
        flex-wrap: wrap;
        gap: 0.35rem;
    }
</style>
