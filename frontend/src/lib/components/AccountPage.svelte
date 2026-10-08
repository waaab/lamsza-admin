<script>
    /**
     * The Fiók page shell in every public app (UI_BASELINE "acc-page"): the
     * tab bar with "Fiók" first and the app's own tabs after it, the open tab
     * in the URL hash (#fiok, #szojavaslataim…), and the sign-in prompt for a
     * signed-out visitor. Shared from lamsza by scripts/sync-shared-frontend.sh.
     * The page title stays each app's own, above this.
     *
     *   <AccountPage tabs={[{ id: "fiok", label: "Fiók" }, …]} signedIn={…}
     *       ready={…} onSignIn={openLogin}>
     *       {#snippet tab(id)}…{/snippet}
     *   </AccountPage>
     */
    import { onMount } from "svelte";

    /**
     * @type {{
     *   tabs: { id: string, label: string }[],
     *   signedIn: boolean,
     *   ready?: boolean,
     *   onSignIn: () => void,
     *   active?: string,
     *   tab: import("svelte").Snippet<[string]>,
     * }}
     */
    let { tabs, signedIn, ready = true, onSignIn, active = $bindable(""), tab } = $props();

    const ids = $derived(tabs.map((t) => t.id));

    function fromHash() {
        const id = decodeURIComponent(location.hash.replace(/^#/, ""));
        return ids.includes(id) ? id : "";
    }

    /** @param {string} id */
    function choose(id) {
        active = id;
        history.replaceState(history.state, "", `#${encodeURIComponent(id)}`);
    }

    onMount(() => {
        active = fromHash() || active || ids[0];
        const onHash = () => {
            const id = fromHash();
            if (id) active = id;
        };
        window.addEventListener("hashchange", onHash);
        return () => window.removeEventListener("hashchange", onHash);
    });

    const current = $derived(ids.includes(active) ? active : ids[0]);
</script>

{#if !ready}
    <div class="account-page" aria-busy="true">
        <nav class="header-tabs account-tabs" aria-hidden="true">
            {#each tabs as t (t.id)}
                <span class="btn skeleton">{t.label}</span>
            {/each}
        </nav>
    </div>
{:else if !signedIn}
    <div class="info-box account-login-prompt">
        <p>Jelentkezz be a fiókodhoz.</p>
        <button type="button" class="btn" onclick={onSignIn}>Belépés</button>
    </div>
{:else}
    <div class="account-page">
        <nav class="header-tabs account-tabs" aria-label="Fiók">
            {#each tabs as t (t.id)}
                <button
                    type="button"
                    class="btn"
                    class:active={current === t.id}
                    aria-current={current === t.id ? "page" : undefined}
                    onclick={() => choose(t.id)}
                >
                    {t.label}
                </button>
            {/each}
        </nav>
        {@render tab(current)}
    </div>
{/if}
