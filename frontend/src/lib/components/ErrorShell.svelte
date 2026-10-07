<script>
    /**
     * The network's minimal error shell (UI_BASELINE "err-page"), shared by all
     * four apps through scripts/sync-shared-frontend.sh. Every app's root
     * +error.svelte renders it, outside the app's own shell: no footer, no
     * sidebar, no right-side toolbar icons, and nothing that fetches data, so
     * it still works when the backend or the database is down.
     *
     * At the top: the "Lámsza" button (the network home) and, in a sub-app,
     * that app's own button. "Vissza a főoldalra" goes to the app's home.
     *
     *   <ErrorShell status={$page.status} appName="Székely szótár"
     *       lamszaHref={lamszaUrl("", $page.url.hostname)}
     *       app={{ label: "Szótár", icon: "szotar" }} />
     */
    import ErrorPage from "$lib/components/ErrorPage.svelte";
    import AppIcon from "$lib/icons/AppIcon.svelte";

    /**
     * @type {{
     *   status?: number,
     *   appName?: string,
     *   lamszaHref?: string,
     *   app?: { label: string, icon: string } | null,
     * }}
     */
    let { status = 500, appName = "Lámsza", lamszaHref = "/", app = null } = $props();
</script>

<div class="layout-bg error-layout">
    <header class="toolbar error-toolbar">
        <div class="nav">
            <a
                href={lamszaHref}
                class="btn nav-btn"
                title={app ? "Lámsza" : "Vissza a főoldalra"}
            >
                <AppIcon name="home" size={16} />
                <span>Lámsza</span>
            </a>
            {#if app}
                <a href="/" class="btn nav-btn" title={app.label}>
                    <AppIcon name={app.icon} size={16} />
                    <span>{app.label}</span>
                </a>
            {/if}
        </div>
    </header>

    <main class="container error-main">
        <ErrorPage {status} {appName} homeHref="/" />
    </main>
</div>

<style>
    .error-layout {
        display: flex;
        flex-direction: column;
        min-height: 100vh;
    }
    .error-toolbar {
        padding: 1rem;
    }
    .error-main {
        flex: 1;
    }
</style>
