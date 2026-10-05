<script>
    import { onMount } from "svelte";
    import { getApiBase } from "$lib/api.js";

    let { clientId = "", onSignedIn = () => {} } = $props();

    let host = $state(null);
    let error = $state("");
    let loading = $state(false);

    onMount(() => {
        if (!clientId) return undefined;
        let cancelled = false;
        loadGis()
            .then(() => {
                if (!cancelled && host && window.google?.accounts?.id) {
                    window.google.accounts.id.initialize({
                        client_id: clientId,
                        callback: handleCredential,
                    });
                    window.google.accounts.id.renderButton(host, {
                        theme: "outline",
                        size: "large",
                        text: "signin_with",
                        locale: "hu",
                        width: 280,
                    });
                }
            })
            .catch(() => {
                if (!cancelled) {
                    error = "A Google belépés betöltése sikertelen.";
                }
            });
        return () => {
            cancelled = true;
        };
    });

    function loadGis() {
        if (typeof window !== "undefined" && window.google?.accounts?.id) {
            return Promise.resolve();
        }
        return new Promise((resolve, reject) => {
            const existing = document.querySelector(
                'script[src="https://accounts.google.com/gsi/client?hl=hu"]',
            );
            if (existing) {
                if (window.google?.accounts?.id) {
                    resolve();
                    return;
                }
                existing.addEventListener("load", () => resolve(), { once: true });
                existing.addEventListener(
                    "error",
                    () => reject(new Error("gis")),
                    { once: true },
                );
                return;
            }
            const s = document.createElement("script");
            s.src = "https://accounts.google.com/gsi/client?hl=hu";
            s.async = true;
            s.defer = true;
            s.onload = () => resolve();
            s.onerror = () => reject(new Error("gis"));
            document.head.appendChild(s);
        });
    }

    async function handleCredential(response) {
        loading = true;
        error = "";
        try {
            const res = await fetch(`${getApiBase()}/api/auth/google`, {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ credential: response.credential }),
            });
            if (!res.ok) {
                const t = await res.text();
                throw new Error(t || "Belépés sikertelen");
            }
            await onSignedIn();
        } catch (e) {
            error = e.message || "Belépés sikertelen";
        } finally {
            loading = false;
        }
    }
</script>

<div class="google-sign-in">
    {#if !clientId}
        <p class="google-sign-in-missing">
            A Google belépés nincs beállítva.
        </p>
    {:else}
        <div class="google-sign-in-button" bind:this={host}></div>
        {#if loading}
            <p class="google-sign-in-status">Belépés…</p>
        {/if}
    {/if}
    {#if error}
        <p class="login-error">{error}</p>
    {/if}
</div>

<style>
    .google-sign-in {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.75rem;
        min-height: 2.5rem;
    }
    .google-sign-in-missing,
    .google-sign-in-status {
        margin: 0;
        color: var(--text-muted, #666);
        font-size: 0.95rem;
        text-align: center;
    }
    .login-error {
        margin: 0;
        color: #b00020;
        text-align: center;
    }
</style>
