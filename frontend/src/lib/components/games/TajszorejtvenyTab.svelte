<script>
    /**
     * The Tájszórejtvény tab of /games (lamsza-jatszoter spec §10.3, §16.6.4;
     * WAYS_OF_WORKING R18): the weekly review, the players' reports, the word
     * metadata, the filler words and the closed periods that changed after
     * their winners were frozen, plus the game's on/off switch. Everything
     * goes through the relay to Játszótér's /internal/admin/tajszorejtveny/.
     *
     * @type {{ stats?: any, onChanged?: () => void }}
     */
    let { stats = null, onChanged = () => {} } = $props();

    import CommonWordsView from "$lib/components/games/tajszorejtveny/CommonWordsView.svelte";
    import ReportsView from "$lib/components/games/tajszorejtveny/ReportsView.svelte";
    import WeeksView from "$lib/components/games/tajszorejtveny/WeeksView.svelte";
    import WordsView from "$lib/components/games/tajszorejtveny/WordsView.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";
    import { LEVEL_LABELS, PERIOD_LABELS } from "$lib/tajszorejtvenyAdmin.js";

    let view = $state("weeks");
    let error = $state("");
    let switching = $state(false);
    /** @type {any[] | null} */
    let late = $state(null);

    const enabled = $derived(stats?.enabled ?? null);

    const VIEWS = $derived([
        { id: "weeks", label: "Hetek" },
        { id: "reports", label: stats?.open_reports ? `Jelzések (${stats.open_reports})` : "Jelzések" },
        { id: "words", label: "Tájszavak" },
        { id: "common", label: "Töltelékszavak" },
        { id: "late", label: stats?.late_periods ? `Lezárt időszakok (${stats.late_periods})` : "Lezárt időszakok" },
    ]);

    async function toggle() {
        if (enabled === null || switching) return;
        const next = !enabled;
        const ok = await confirmDialog(
            next
                ? "Bekapcsolod a Tájszórejtvényt? A játékosok azonnal látják a Játszótéren."
                : "Kikapcsolod a Tájszórejtvényt? A játékosok nem érik el, amíg újra be nem kapcsolod.",
            { yesLabel: next ? "Bekapcsolás" : "Kikapcsolás", destructive: !next },
        );
        if (!ok) return;
        switching = true;
        error = "";
        try {
            await sectionFetch("games", "tajszorejtveny/enabled", { method: "POST", body: { enabled: next } });
            onChanged();
        } catch (err) {
            error = errorText(err, "A kapcsoló nem állítható.");
        } finally {
            switching = false;
        }
    }

    async function loadLate() {
        try {
            const data = await sectionFetch("games", "tajszorejtveny/late-periods");
            late = data?.periods ?? [];
        } catch (err) {
            error = errorText(err, "A lista nem tölthető be.");
        }
    }

    $effect(() => {
        if (view === "late" && late === null) loadLate();
    });
</script>

<div class="taj-switch">
    {#if enabled === null}
        <span class="taj-muted">A játék állapota most nem érhető el.</span>
    {:else}
        <span>A játék <strong>{enabled ? "be van kapcsolva" : "ki van kapcsolva"}</strong>{enabled ? "." : ": a játékosok nem látják."}</span>
        <button type="button" class="btn btn-sm" disabled={switching} onclick={toggle}>{enabled ? "Kikapcsolás" : "Bekapcsolás"}</button>
    {/if}
</div>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

<div class="header-tabs taj-views" role="group" aria-label="Tájszórejtvény nézetek">
    {#each VIEWS as v (v.id)}
        <button type="button" class="btn" class:active={view === v.id} onclick={() => (view = v.id)}>{v.label}</button>
    {/each}
</div>

{#if view === "weeks"}
    <WeeksView {onChanged} />
{:else if view === "reports"}
    <ReportsView {onChanged} />
{:else if view === "words"}
    <WordsView />
{:else if view === "common"}
    <CommonWordsView />
{:else if view === "late"}
    <p class="admin-info">
        Ezekben az időszakokban a győztesek már ki voltak hirdetve, amikor új eredmény érkezett (egy vendég belépett, vagy egy napi
        rejtvényt később fejeztek be). A kihirdetett győztesek nem változnak; a lista azt mutatja, hány eredmény érkezett utólag.
    </p>
    <div class="admin-table-wrapper">
        <table class="admin-table">
            <thead>
                <tr><th>Időszak</th><th>Kezdete</th><th>Szint</th><th>Utólagos eredmény</th></tr>
            </thead>
            <tbody>
                {#each late ?? [] as l (`${l.period}-${l.start}-${l.level}`)}
                    <tr>
                        <td>{PERIOD_LABELS[l.period] ?? l.period}</td>
                        <td>{formatHuDate(l.start)}</td>
                        <td>{LEVEL_LABELS[l.level] ?? l.level}</td>
                        <td>{l.results}</td>
                    </tr>
                {:else}
                    <tr><td colspan="4">{late === null ? "Betöltés…" : "Nincs ilyen időszak."}</td></tr>
                {/each}
            </tbody>
        </table>
    </div>
{/if}

<style>
    .taj-switch {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 1rem;
    }

    .taj-muted {
        color: var(--text-muted);
    }

    .taj-views {
        margin-bottom: 1rem;
    }
</style>
