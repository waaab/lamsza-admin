<script>
    import { createEventDispatcher } from "svelte";
    import { ADMIN_PAGE_SIZE } from "$lib/adminPageSlice.js";

    const dispatch = createEventDispatcher();

    /** @type {number} */
    export let total = 0;
    export let page = 1;
    export let totalPages = 1;
    export let from = 0;
    export let to = 0;
    export let pageSize = ADMIN_PAGE_SIZE;
</script>

<div class="admin-pagination" role="navigation" aria-label="Táblázat lapozás">
    <span class="admin-pagination-info">
        {#if total === 0}
            Nincs megjeleníthető sor.
        {:else}
            Összesen <strong>{total}</strong> sor · ezen az oldalon
            <strong>{from}</strong>–<strong>{to}</strong>. ·
            <strong>{page}</strong>. / <strong>{totalPages}</strong> oldal
            ({pageSize} / oldal)
        {/if}
    </span>
    <div class="admin-pagination-nav">
        <button
            type="button"
            class="btn-update"
            disabled={page <= 1 || total === 0}
            on:click={() => dispatch("prev")}>‹ Előző</button
        >
        <button
            type="button"
            class="btn-update"
            disabled={page >= totalPages || total === 0}
            on:click={() => dispatch("next")}>Következő ›</button
        >
    </div>
</div>