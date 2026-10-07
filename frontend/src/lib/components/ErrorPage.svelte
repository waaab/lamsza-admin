<script>
    /**
     * The network's error page (UI_BASELINE "err-page"), shared by all four
     * apps through scripts/sync-shared-frontend.sh. Each app's +error.svelte
     * renders it inside its own shell:
     *
     *   <ErrorPage status={$page.status} appName="Székely szótár" />
     *
     * The text comes from the status code, in plain Hungarian; SvelteKit's own
     * message ("Not Found" and the like) is English and never shown.
     */
    import ErrorLantern from "$lib/icons/ErrorLantern.svelte";

    /** @type {{ status?: number, appName?: string, homeHref?: string }} */
    let { status = 500, appName = "Lámsza", homeHref = "/" } = $props();

    /** @type {Record<number, { title: string, text: string }>} */
    const MESSAGES = {
        400: {
            title: "Hibás kérés",
            text: "A kérés nem értelmezhető. Ellenőrizd a címet, és próbáld újra.",
        },
        401: {
            title: "Bejelentkezés szükséges",
            text: "Ezt az oldalt csak bejelentkezve láthatod.",
        },
        403: {
            title: "Nincs hozzáférés",
            text: "Ehhez az oldalhoz nincs jogosultságod.",
        },
        404: {
            title: "Az oldal nem található",
            text: "Lámpással is kerestük, de ez az oldal nincs meg. Lehet, hogy elköltözött, vagy sosem létezett.",
        },
        408: {
            title: "Lejárt a várakozási idő",
            text: "A kérés túl sokáig tartott. Próbáld újra.",
        },
        410: {
            title: "Az oldal megszűnt",
            text: "Ez az oldal már nem érhető el.",
        },
        429: {
            title: "Túl sok kérés",
            text: "Kicsit lassíts! Próbáld újra pár perc múlva.",
        },
    };

    const FALLBACK = {
        title: "Valami hiba történt",
        text: "Váratlan hiba történt. Próbáld újra kicsit később.",
    };

    const message = $derived(MESSAGES[status] ?? FALLBACK);
</script>

<svelte:head>
    <title>Hiba {status} - {appName}</title>
    <!--
      An unknown path is served the SPA fallback with HTTP 200, so a crawler
      would index this page as content. Keep it out of the index.
    -->
    <meta name="robots" content="noindex" />
</svelte:head>

<section class="error-page" aria-labelledby="error-page-title">
    <ErrorLantern size={240} />
    <h1 id="error-page-title" class="error-page-oops">Hoppácska!</h1>
    <p class="error-page-code">Hiba: {status}</p>
    <h2 class="error-page-title">{message.title}</h2>
    <p class="error-page-text">{message.text}</p>
    <a class="btn" href={homeHref}>Vissza a főoldalra</a>
</section>

<style>
    .error-page {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
        max-width: 36rem;
        margin: 1.5rem auto 3rem;
        padding: 0 1rem;
        text-align: center;
    }
    .error-page :global(.lmz-err-art) {
        margin-bottom: 0.75rem;
    }
    .error-page-oops {
        margin: 0;
        font-size: var(--text-2xl);
        color: var(--szekely-red);
    }
    .error-page-code {
        margin: 0;
        font-size: var(--text-sm);
        font-weight: 600;
        letter-spacing: 0.04em;
        color: var(--text-muted);
        font-variant-numeric: tabular-nums;
    }
    .error-page-title {
        margin: 0.75rem 0 0;
        font-size: var(--text-lg);
        color: var(--text-primary);
        text-wrap: balance;
    }
    .error-page-text {
        margin: 0 0 1.25rem;
        color: var(--text-secondary);
        line-height: 1.55;
        text-wrap: pretty;
    }
</style>
