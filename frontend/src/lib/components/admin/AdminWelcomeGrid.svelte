<script>
    import AdminNavIcon from "$lib/components/admin/AdminNavIcon.svelte";

    /**
     * A section dashboard's cards: the same entries, icons and titles as its
     * sidebar (below Vezérlőpult), each with its count. A card opens its tab.
     *
     * @type {{
     *   items: { id: string, title: string, icon?: string, countTitle?: string }[],
     *   counts: Record<string, number | string | null | undefined> | null,
     *   countTitle?: string,
     *   onSelect: (id: string) => void,
     * }}
     */
    let { items, counts, countTitle = "Rekordok száma az adatbázisban", onSelect } = $props();
</script>

<div class="admin-welcome" role="navigation" aria-label="Admin részlegek">
    <div class="admin-welcome-grid">
        {#each items as item (item.id)}
            {@const count = counts ? counts[item.id] : null}
            <button
                type="button"
                class="admin-welcome-card"
                onclick={() => onSelect(item.id)}
                title={item.title}
                aria-label={item.title}
            >
                <div class="admin-welcome-card-body">
                    <AdminNavIcon name={item.icon || item.id} size={56} />
                </div>
                <div class="admin-welcome-card-footer">
                    <span class="admin-welcome-card-footer-left" title={count != null ? item.countTitle || countTitle : "Betöltés…"}
                        >{count != null ? count : "…"}</span
                    >
                    <span class="admin-welcome-card-footer-right" title={item.title}>{item.title}</span>
                </div>
            </button>
        {/each}
    </div>
</div>
