<script>
    /**
     * The Szótár section of the admin app (lamsza WAYS_OF_WORKING R18). Every
     * call goes through this backend's relay to Szótár's internal admin API
     * (src/lib/sectionApi.js); this app never touches Szótár's database.
     */
    import AdminShell from "$lib/components/admin/AdminShell.svelte";
    import AdminWelcomeGrid from "$lib/components/admin/AdminWelcomeGrid.svelte";
    import ConfirmHost from "$lib/components/admin/ConfirmHost.svelte";
    import DailyTab from "$lib/components/dictionary/DailyTab.svelte";
    import ProverbsTab from "$lib/components/dictionary/ProverbsTab.svelte";
    import SpeechTypesTab from "$lib/components/dictionary/SpeechTypesTab.svelte";
    import SuggestionsTab from "$lib/components/dictionary/SuggestionsTab.svelte";
    import WordsTab from "$lib/components/dictionary/WordsTab.svelte";
    import { APP_ORIGINS } from "$lib/adminApps.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";

    const szotarOrigin = APP_ORIGINS.szotar;

    /** Sidebar, in order; the dashboard cards are the same entries below Vezérlőpult. */
    const NAV = [
        { id: "welcome", title: "Vezérlőpult", icon: "dashboard" },
        { id: "napi", title: "Napi szó", icon: "events", countTitle: "Hangfelvételes, választható szavak" },
        { sep: true },
        { id: "szavak", title: "Szavak", icon: "list" },
        { id: "javaslatok", title: "Szójavaslatok", icon: "word-suggestions", countTitle: "Elbírálásra váró szójavaslatok" },
        { id: "szofajok", title: "Szófajok", icon: "tags" },
        { id: "mondasok", title: "Mondások", icon: "mondasok" },
    ];
    const CARDS = /** @type {{ id: string, title: string, icon: string, countTitle?: string }[]} */ (
        NAV.filter((n) => n.id && n.id !== "welcome")
    );

    /** @type {Record<string, { title: string, greeting: string }>} */
    const COPY = {
        welcome: {
            title: "Vezérlőpult Szótár",
            greeting:
                "A Székely szótár adminisztrációja. A kártyákon a rekordok száma látható, a név pedig megegyezik az oldalsáv gombjaival.",
        },
        napi: { title: "Napi szó", greeting: "A Szótár kezdőlapján megjelenő napi székely szó." },
        szavak: { title: "Szavak", greeting: "Címszavak és jelentések." },
        szofajok: { title: "Szófajok", greeting: "Bemutatkozó szövegek a szófaj oldalakon." },
        mondasok: { title: "Mondások", greeting: "A napi mondások ütemezése." },
        javaslatok: {
            title: "Szójavaslatok",
            greeting:
                "A felhasználók által beküldött szójavaslatok elbírálása: új szavak és meglévő szavak módosításai (új jelentés, alak és hasonlók).",
        },
    };

    /** @type {AdminShell | null} */
    let shell = $state(null);
    let active = $state("welcome");
    /** @type {number | null} word to open in the editor (#szavak/<id>) */
    let openWordId = $state(null);
    /** @type {Record<string, number> | null} */
    let counts = $state(null);
    /** @type {any} */
    let daily = $state(null);
    let statsError = $state("");

    const head = $derived(COPY[active] || COPY.welcome);

    async function loadStats() {
        try {
            const [stats, day, choices] = await Promise.all([
                sectionFetch("dictionary", "stats"),
                sectionFetch("dictionary", "daily"),
                sectionFetch("dictionary", "daily/choices"),
            ]);
            counts = {
                napi: (choices?.words || []).length,
                szavak: stats.words,
                szofajok: stats.speech_types,
                mondasok: stats.proverbs,
                javaslatok: stats.open_suggestions,
            };
            daily = day;
            statsError = "";
        } catch (err) {
            statsError = errorText(err, "A Szótár adatai most nem érhetők el.");
        }
    }

    function onSelect(/** @type {string} */ tab, /** @type {string} */ rest) {
        active = tab;
        openWordId = tab === "szavak" && /^\d+$/.test(rest) ? Number(rest) : null;
        if (tab === "welcome") loadStats();
    }
</script>

<svelte:head>
    <title>Szótár - Adminisztráció</title>
</svelte:head>

<AdminShell
    bind:this={shell}
    title={head.title}
    greeting={head.greeting}
    nav={NAV}
    {active}
    external={{ href: szotarOrigin, title: "Szótár megnyitása új lapon" }}
    {onSelect}
    onReady={loadStats}
>
    {#if active === "welcome"}
        <section class="admin-subsection" aria-labelledby="dict-messages-title">
            <h3 id="dict-messages-title">Üzenetek</h3>
            {#if statsError}
                <div class="info-box error" role="alert">
                    <p>{statsError}</p>
                    <p><button type="button" class="btn btn-sm" onclick={loadStats}>Újra</button></p>
                </div>
            {/if}
            {#if daily}
                <div class="info-box info" role="status">
                    <p>
                        {#if daily.word}
                            Napi szó: <strong>{daily.word.headword}</strong>
                            · {daily.pinned && daily.until
                                ? `Beállítva eddig: ${formatHuDate(daily.until)}`
                                : "Automatikus"}
                        {:else}
                            Ma nincs hangfelvételes napi szó.
                        {/if}
                    </p>
                    <p><button type="button" class="btn btn-sm" onclick={() => shell?.select("napi")}>Megnyitás</button></p>
                </div>
            {/if}
            {#if counts}
                {#if counts.javaslatok > 0}
                    <div class="info-box warning" role="status">
                        <p>{counts.javaslatok} szójavaslat vár elbírálásra.</p>
                        <p>
                            <button type="button" class="btn btn-sm" onclick={() => shell?.select("javaslatok")}
                                >Megnyitás</button
                            >
                        </p>
                    </div>
                {:else}
                    <div class="info-box" role="status"><p>Nincs elbírálásra váró szójavaslat.</p></div>
                {/if}
            {/if}
        </section>
        <AdminWelcomeGrid items={CARDS} {counts} onSelect={(id) => shell?.select(id)} />
    {:else if active === "napi"}
        <DailyTab {szotarOrigin} onChanged={loadStats} />
    {:else if active === "szavak"}
        <WordsTab {szotarOrigin} openId={openWordId} onChanged={loadStats} />
    {:else if active === "szofajok"}
        <SpeechTypesTab />
    {:else if active === "mondasok"}
        <ProverbsTab onChanged={loadStats} />
    {:else if active === "javaslatok"}
        <SuggestionsTab {szotarOrigin} onChanged={loadStats} />
    {/if}

    {#snippet overlays()}
        <ConfirmHost />
    {/snippet}
</AdminShell>
