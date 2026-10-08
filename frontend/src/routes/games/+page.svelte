<script>
    /**
     * The Játszótér section of the admin app (lamsza WAYS_OF_WORKING R18). Every
     * call goes through this backend's relay to Játszótér's internal admin API
     * (src/lib/sectionApi.js); this app never touches Játszótér's database.
     */
    import AdminShell from "$lib/components/admin/AdminShell.svelte";
    import AdminWelcomeGrid from "$lib/components/admin/AdminWelcomeGrid.svelte";
    import ConfirmHost from "$lib/components/admin/ConfirmHost.svelte";
    import RovasfejtoTab from "$lib/components/games/RovasfejtoTab.svelte";
    import SzokeresoTab from "$lib/components/games/SzokeresoTab.svelte";
    import TajszorejtvenyTab from "$lib/components/games/TajszorejtvenyTab.svelte";
    import { APP_ORIGINS } from "$lib/adminApps.js";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";
    import { dashboardLines } from "$lib/tajszorejtvenyAdmin.js";

    const jatszoterOrigin = APP_ORIGINS.jatszoter;

    /** Sidebar, in order; the dashboard cards are the same entries below Vezérlőpult. */
    const NAV = [
        { id: "welcome", title: "Vezérlőpult", icon: "dashboard" },
        { sep: true },
        { id: "rovasfejto", title: "Rovásfejtő", icon: "rovasfejto", countTitle: "Aktív feladványok" },
        { id: "szokereso", title: "Szókereső", icon: "szokereso", countTitle: "Közzétett feladványok és piszkozatok" },
        { id: "tajszorejtveny", title: "Tájszórejtvény", icon: "tajszorejtveny", countTitle: "Jövő hét: jóváhagyott / elkészült rejtvények" },
    ];
    const CARDS = /** @type {{ id: string, title: string, icon: string, countTitle?: string }[]} */ (
        NAV.filter((n) => n.id && n.id !== "welcome")
    );

    /** @type {Record<string, { title: string, greeting: string }>} */
    const COPY = {
        welcome: {
            title: "Vezérlőpult Játszótér",
            greeting:
                "A Játszótér feladványainak adminisztrációja. A kártyákon a feladványok száma látható, a név pedig megegyezik az oldalsáv gombjaival.",
        },
        rovasfejto: { title: "Rovásfejtő", greeting: "Közmondások és más feladványok rovásírással." },
        szokereso: { title: "Szókereső", greeting: "A napi szókereső rácsok összeállítása." },
        tajszorejtveny: {
            title: "Tájszórejtvény",
            greeting: "A heti rejtvények átnézése és jóváhagyása, a játékosok hibajelzései és a játék szavai.",
        },
    };

    /** @type {AdminShell | null} */
    let shell = $state(null);
    let active = $state("welcome");
    /** @type {any} */
    let stats = $state(null);
    let statsError = $state("");

    const head = $derived(COPY[active] || COPY.welcome);
    const tajLines = $derived(dashboardLines(stats?.tajszorejtveny));
    const counts = $derived(
        stats
            ? {
                  rovasfejto: stats.rovasfejto.active,
                  szokereso: stats.szokereso.published + stats.szokereso.drafts,
                  tajszorejtveny: stats.tajszorejtveny
                      ? `${stats.tajszorejtveny.next_approved}/${stats.tajszorejtveny.next_total}`
                      : null,
              }
            : null,
    );

    async function loadStats() {
        try {
            stats = await sectionFetch("games", "stats");
            statsError = "";
        } catch (err) {
            statsError = errorText(err, "A Játszótér adatai most nem érhetők el.");
        }
    }

    function onSelect(/** @type {string} */ tab) {
        active = tab;
        if (tab === "welcome") loadStats();
    }
</script>

<svelte:head>
    <title>Játszótér - Adminisztráció</title>
</svelte:head>

<AdminShell
    bind:this={shell}
    title={head.title}
    greeting={head.greeting}
    nav={NAV}
    {active}
    external={{ href: jatszoterOrigin, title: "Játszótér megnyitása új lapon" }}
    {onSelect}
    onReady={loadStats}
>
    {#if active === "welcome"}
        <section class="admin-subsection" aria-labelledby="games-messages-title">
            <h3 id="games-messages-title">Üzenetek</h3>
            {#if statsError}
                <div class="info-box error" role="alert">
                    <p>{statsError}</p>
                    <p><button type="button" class="btn btn-sm" onclick={loadStats}>Újra</button></p>
                </div>
            {/if}
            {#if stats}
                {#if stats.dictionary.ok}
                    <div class="info-box info" role="status">
                        <p>Szótár adatok betöltve: {stats.dictionary.word_count} szó ({stats.dictionary.source}).</p>
                    </div>
                {:else}
                    <div class="info-box warning" role="status">
                        <p>A Szótár adatai most nem elérhetők, ezért a Szókereső szólistája üres lehet.</p>
                    </div>
                {/if}
                <div class="info-box" role="status">
                    <p>
                        Rovásfejtő: {stats.rovasfejto.active} aktív, {stats.rovasfejto.deleted} törölt feladvány.
                        Szókereső: {stats.szokereso.published} közzétett, {stats.szokereso.drafts} piszkozat.
                    </p>
                </div>
                {#each tajLines as line (line.text)}
                    <div class="info-box {line.level}" role="status"><p>{line.text}</p></div>
                {/each}
            {/if}
        </section>
        <AdminWelcomeGrid items={CARDS} {counts} onSelect={(id) => shell?.select(id)} />
    {:else if active === "rovasfejto"}
        <RovasfejtoTab onChanged={loadStats} />
    {:else if active === "szokereso"}
        <SzokeresoTab onChanged={loadStats} />
    {:else if active === "tajszorejtveny"}
        <TajszorejtvenyTab stats={stats?.tajszorejtveny ?? null} onChanged={loadStats} />
    {/if}

    {#snippet overlays()}
        <ConfirmHost />
    {/snippet}
</AdminShell>
