<script>
    import GoogleSignIn from "$lib/components/GoogleSignIn.svelte";

    let {
        open = false,
        appName = "Lámsza",
        clientId = "",
        configReady = true,
        onClose = () => {},
        onSignedIn = () => {},
        policyHref = "/iranyelvek",
        signIn = null,
    } = $props();

    function articleName(name) {
        const word = String(name || "").trim();
        const first = word.charAt(0).toLocaleLowerCase("hu-HU");
        const vowel = "aáeéiíoóöőuúüű".includes(first);
        return `${vowel ? "az" : "a"} ${word}`;
    }

    function onKey(event) {
        if (open && event.key === "Escape") onClose();
    }
</script>

<svelte:window onkeydown={onKey} />

{#if open}
    <div class="signin-overlay" role="presentation" onclick={onClose} onkeydown={onKey}>
        <div
            class="signin-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="signin-title"
            aria-describedby="signin-lead"
            tabindex="-1"
            onclick={(event) => event.stopPropagation()}
            onkeydown={(event) => {
                if (event.key === "Escape") onClose();
                event.stopPropagation();
            }}
        >
            <h2 id="signin-title">Belépés</h2>
            <p id="signin-lead">
                Jelentkezz be a Google-fiókoddal, hogy ki tudd használni {articleName(appName)} teljes funkcióit és lehetőségeit.
            </p>
            <p class="signin-network">
                A Lámsza hálózat alkalmazása. A belépés csak ezen az oldalon jelentkeztet be.
                <a href={policyHref}>Irányelvek</a>.
            </p>
            <div class="signin-body">
                {#if clientId}
                    {#key clientId}
                        <GoogleSignIn {clientId} {onSignedIn} {signIn} />
                    {/key}
                {:else if configReady}
                    <p class="signin-note">A Google belépés nincs beállítva.</p>
                {:else}
                    <p class="signin-note">Betöltés…</p>
                {/if}
                <button type="button" class="signin-cancel" onclick={onClose}>Mégse</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .signin-overlay {
        position: fixed;
        inset: 0;
        z-index: 1000;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 1rem;
        background: rgba(0, 0, 0, 0.62);
    }

    .signin-dialog {
        width: min(26rem, 100%);
        box-sizing: border-box;
        padding: 1.75rem 1.5rem 1.35rem;
        border: 1px solid var(--border-color);
        border-radius: 16px;
        background: var(--card-bg);
        box-shadow: 0 18px 48px rgba(0, 0, 0, 0.35);
        text-align: center;
    }

    h2 {
        margin: 0;
        font-size: var(--text-lg, 1.25rem);
        font-weight: 700;
        color: var(--text-primary);
    }

    #signin-lead {
        margin: 0.85rem auto 0;
        max-width: 22rem;
        color: var(--text-muted);
        font-size: var(--text-base);
        line-height: 1.5;
    }

    .signin-network {
        margin: 0.75rem auto 0;
        max-width: 22rem;
        color: var(--text-muted);
        font-size: var(--text-sm, 0.9rem);
        line-height: 1.45;
    }

    .signin-network a {
        color: inherit;
    }

    .signin-body {
        width: min(280px, 100%);
        margin: 1.25rem auto 0;
        display: flex;
        flex-direction: column;
        align-items: stretch;
        gap: 0.85rem;
    }

    .signin-note {
        margin: 0;
        color: var(--text-muted);
        font-size: var(--text-sm, 0.9rem);
        line-height: 1.4;
    }

    .signin-cancel {
        width: 100%;
        margin: 0;
        padding: 0.55rem 1rem;
        border: 1px solid var(--border-color);
        border-radius: 8px;
        background: transparent;
        color: var(--text-muted);
        font: inherit;
        font-size: var(--text-base);
        cursor: pointer;
    }

    .signin-cancel:hover {
        color: var(--text-primary);
        border-color: var(--szekely-brown, var(--border-color));
        background: var(--hover-bg, rgba(255, 255, 255, 0.04));
    }
</style>
