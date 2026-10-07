<script>
    /**
     * The admin app's shell, shared by its three sections (lamsza
     * WAYS_OF_WORKING R18): `/` (Lámsza), `/dictionary` (Szótár) and `/games`
     * (Játszótér).
     *
     * It owns the sign-in gate (UI_BASELINE "adm-gate": the shared sign-in
     * dialog opens by itself, a signed-in non-admin is sent on to Lámsza), the
     * icon sidebar, the header with the section switcher and sign-out, the
     * back-to-top button, and the tab deep links: a bare `#tab` (or
     * `#tab/rest`, e.g. `#szavak/42`) opens that tab.
     *
     * The page owns its tabs and data: `onReady` runs once the person is in,
     * `onSelect(tab, rest)` runs for every tab change.
     */
    import { onMount } from "svelte";
    import { fade } from "svelte/transition";
    import { replaceState } from "$app/navigation";
    import { auth } from "$lib/stores/auth";
    import { apiCall } from "$lib/api.js";
    import { adminAppLinks, APP_ORIGINS } from "$lib/adminApps.js";
    import AdminNavIcon from "$lib/components/admin/AdminNavIcon.svelte";
    import SignInDialog from "$lib/components/SignInDialog.svelte";
    import AppIcon from "$lib/icons/AppIcon.svelte";

    /**
     * `nav` entries are tabs ({ id, title, icon? }) or separators ({ sep: true }).
     *
     * @type {{
     *   title: string,
     *   greeting?: string,
     *   nav: { id?: string, title?: string, icon?: string, sep?: boolean }[],
     *   active: string,
     *   home?: string,
     *   external: { href: string, title: string },
     *   onSelect: (tab: string, rest: string) => void,
     *   onReady?: (me: { offline: boolean }) => void,
     *   children: import("svelte").Snippet,
     *   overlays?: import("svelte").Snippet,
     * }}
     */
    let {
        title,
        greeting = "",
        nav,
        active,
        home = "welcome",
        external,
        onSelect,
        onReady = () => {},
        children,
        overlays,
    } = $props();

    const lamszaOrigin = APP_ORIGINS.lamsza;
    const apps = adminAppLinks(typeof location !== "undefined" ? location.pathname : "/");

    let authenticated = $state(false);
    let authReady = $state(false);
    let authDenied = $state(false);
    let googleClientId = $state("");
    let configUnreachable = $state(false);
    let loginOpen = $state(false);
    /** The shell scrolls inside <main>, not the window (back-to-top). */
    /** @type {HTMLElement | null} */
    let mainEl = $state(null);
    let scrollTop = $state(0);

    const tabIds = $derived(new Set(nav.map((n) => n.id).filter(Boolean)));

    /** Opens a tab from the sidebar or a dashboard card, and keeps the hash in step. */
    export function select(/** @type {string} */ tab, rest = "") {
        onSelect(tab, rest);
        const hash = tab === home ? "" : `#${tab}${rest ? `/${rest}` : ""}`;
        try {
            replaceState(`${location.pathname}${location.search}${hash}`, {});
        } catch {
            /* before the router is ready; the hash is only a convenience */
        }
    }

    /** Applies `#tab` or `#tab/rest` from the address bar, if it names a tab. */
    function openFromHash() {
        const raw = decodeURIComponent(location.hash.replace(/^#/, ""));
        if (!raw) return;
        const [tab, ...rest] = raw.split("/");
        if (tabIds.has(tab)) onSelect(tab, rest.join("/"));
    }

    function enter(/** @type {{ offline?: boolean }} */ me) {
        authenticated = true;
        authDenied = false;
        onReady({ offline: !!me.offline });
        openFromHash();
    }

    onMount(() => {
        (async () => {
            try {
                const res = await apiCall("/api/config/public");
                if (res.ok) {
                    const data = await res.json();
                    googleClientId = data.google_client_id || "";
                    configUnreachable = false;
                } else {
                    configUnreachable = true;
                }
            } catch (e) {
                configUnreachable = true;
                console.error(e);
            }
            const me = await auth.refresh();
            authReady = true;
            if (me.isAdmin) {
                enter(me);
            } else if (me.loggedIn) {
                sendNonAdminAway();
            } else {
                // Sign-in gate (UI_BASELINE "adm-gate"): open the dialog straight away.
                loginOpen = true;
            }
        })();
        const onHash = () => authenticated && openFromHash();
        window.addEventListener("hashchange", onHash);
        return () => window.removeEventListener("hashchange", onHash);
    });

    async function onGoogleSignedIn() {
        const me = await auth.refresh();
        loginOpen = false;
        if (me.isAdmin) enter(me);
        else sendNonAdminAway();
    }

    /**
     * Sign in through admin's own API, for the shared GoogleSignIn. A Google
     * account that is not on the admin allowlist gets 403 from
     * /api/auth/google; that visitor is sent on to Lámsza (UI_BASELINE
     * "adm-gate") instead of seeing only "Belépés sikertelen.". Any other
     * failure throws, and the button shows that message.
     * @param {string} credential
     */
    async function adminSignIn(credential) {
        const res = await apiCall("/api/auth/google", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ credential }),
        });
        if (res.status === 403) {
            sendNonAdminAway();
            return new Promise(() => {}); // the page is leaving
        }
        if (!res.ok) {
            console.warn("admin sign-in refused:", res.status, await res.text());
            throw new Error("refused");
        }
    }

    /** A signed-in non-admin has nothing to do here: on to Lámsza (UI_BASELINE "adm-gate"). */
    function sendNonAdminAway() {
        authDenied = true;
        window.location.assign(lamszaOrigin);
    }

    async function logout() {
        authenticated = false;
        await auth.logout();
        window.location.href = "/";
    }
</script>

{#if !authenticated}
    <!-- Sign-in gate, network baseline (UI_BASELINE "adm-gate", Játszótér's): the
         shared sign-in dialog opens by itself; a signed-in non-admin is sent on
         to Lámsza. The admin app keeps its own shell, so no toolbar or footer. -->
    <div class="container">
        <div class="admin-login-wrapper">
            <div class="card login-box">
                {#if !authReady}
                    <p class="greeting">Betöltés…</p>
                {:else if authDenied}
                    <p class="greeting">Ehhez admin jogosultság kell.</p>
                {:else}
                    <p class="greeting">Az admin felülethez lépj be.</p>
                    {#if configUnreachable}
                        <p>Az API nem elérhető, ezért a belépés most nem lehetséges.</p>
                    {/if}
                    <button type="button" class="btn" onclick={() => (loginOpen = true)}>Belépés</button>
                {/if}
            </div>
        </div>
    </div>
    <SignInDialog
        open={loginOpen}
        appName="Lámsza admin"
        clientId={googleClientId}
        configReady={authReady}
        onClose={() => (loginOpen = false)}
        onSignedIn={onGoogleSignedIn}
        signIn={adminSignIn}
        policyHref={`${lamszaOrigin}/iranyelvek`}
    />
{:else}
    <div class="admin-layout">
        <aside class="admin-sidebar">
            <div class="admin-sidebar-inner" class:admin-sidebar-inner--short={nav.length < 12}>
                <a
                    href={external.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="admin-sidebar-btn admin-sidebar-btn--external"
                    title={external.title}
                    aria-label={external.title}
                >
                    <AdminNavIcon name="external" />
                </a>
                {#each nav as item, i (item.id ?? `sep-${i}`)}
                    {#if item.id}
                        {@const id = item.id}
                        <button
                            type="button"
                            class="admin-sidebar-btn {active === id ? 'active' : ''}"
                            onclick={() => select(id)}
                            title={item.title}
                            aria-label={item.title}
                            aria-current={active === id ? "page" : undefined}
                        >
                            <AdminNavIcon name={item.icon || id} />
                        </button>
                    {:else}
                        <hr class="admin-sidebar-sep" aria-hidden="true" />
                    {/if}
                {/each}
            </div>
        </aside>

        <main class="admin-main" bind:this={mainEl} onscroll={() => (scrollTop = mainEl?.scrollTop ?? 0)}>
            <header class="admin-header">
                <div class="admin-header-text">
                    <h1 class="admin-page-title">{title}</h1>
                    {#if greeting}
                        <p class="admin-page-greeting">{greeting}</p>
                    {/if}
                </div>
                <nav class="admin-header-actions" aria-label="Lámsza admin részlegek">
                    {#each apps as app (app.id)}
                        <a
                            href={app.href}
                            class="btn nav-btn admin-header-icon-btn"
                            title={app.label}
                            aria-label={app.label}
                            aria-current={app.current ? "page" : undefined}
                        >
                            <AppIcon name={app.icon} size={20} />
                        </a>
                    {/each}
                    <button
                        class="btn nav-btn admin-header-icon-btn"
                        type="button"
                        onclick={logout}
                        title="Kijelentkezés"
                        aria-label="Kijelentkezés"
                    >
                        <AppIcon name="logout" size={20} />
                    </button>
                </nav>
            </header>

            <div class="admin-container w-full">
                {@render children()}
            </div>
        </main>
        <!-- Back-to-top, as on the other apps (UI_BASELINE "tb-backtotop"). -->
        {#if scrollTop > 500}
            <button
                type="button"
                class="btn back-to-top"
                onclick={() => mainEl?.scrollTo({ top: 0, behavior: "smooth" })}
                aria-label="Ugrás az oldal tetejére"
                transition:fade={{ duration: 200 }}>↑</button
            >
        {/if}
    </div>

    {@render overlays?.()}
{/if}

<style>
    .login-box {
        max-width: 400px;
        text-align: center;
        width: 100%;
    }
    .w-full {
        width: 100%;
    }
</style>
