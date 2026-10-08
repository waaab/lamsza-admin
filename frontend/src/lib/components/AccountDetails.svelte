<script>
    /**
     * The user details on the Fiók tab, the same in every public app
     * (UI_BASELINE "acc-page"). Shared from lamsza by
     * scripts/sync-shared-frontend.sh. `before` renders above the list: the
     * display-name editor in Lámsza, a read-only note with a link elsewhere.
     *
     *   <AccountDetails account={accountFromMe(me)}>
     *       {#snippet before()}…{/snippet}
     *   </AccountDetails>
     */
    import { accountDetailRows } from "$lib/accountDetails.js";

    /** @type {{ account: import("$lib/accountDetails.js").Account, before?: import("svelte").Snippet }} */
    let { account, before } = $props();

    const rows = $derived(accountDetailRows(account));
</script>

<div class="account-details">
    {@render before?.()}
    <dl class="account-fields">
        {#each rows as row (row.key)}
            <dt>{row.label}</dt>
            {#if row.kind === "photo"}
                <dd><img src={row.value} alt="" class="account-photo" referrerpolicy="no-referrer" /></dd>
            {:else}
                <dd>{row.value}</dd>
            {/if}
        {/each}
    </dl>
</div>
