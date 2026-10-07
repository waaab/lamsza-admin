<script>
    /**
     * Szótár's daily word: what shows today, whether it is pinned, and the
     * dialog to pin another one (moved from Szótár's /admin hub).
     *
     * @type {{ szotarOrigin: string, onChanged?: () => void }}
     */
    let { szotarOrigin, onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import DailyDialog from "$lib/components/dictionary/DailyDialog.svelte";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";

    /** @type {any} */
    let day = $state(null);
    let choices = $state(/** @type {number | null} */ (null));
    let error = $state("");
    let loaded = $state(false);
    let dialogOpen = $state(false);

    async function load() {
        try {
            const [d, list] = await Promise.all([
                sectionFetch("dictionary", "daily"),
                sectionFetch("dictionary", "daily/choices"),
            ]);
            day = d;
            choices = (list?.words || []).length;
            error = "";
        } catch (err) {
            error = errorText(err, "A napi szó most nem elérhető.");
        } finally {
            loaded = true;
        }
    }

    onMount(load);
</script>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

<section class="admin-subsection" aria-labelledby="dict-daily-now">
    <h3 id="dict-daily-now">Jelenleg</h3>
    {#if !loaded}
        <p>Betöltés…</p>
    {:else if day?.word}
        <p class="daily-now">
            <a href={`${szotarOrigin}/szo/${day.word.id}`} target="_blank" rel="noopener noreferrer"
                ><strong>{day.word.headword}</strong></a
            >
            <span class="admin-status">
                · {day.pinned && day.until ? `Beállítva eddig: ${formatHuDate(day.until)}` : "Automatikus"}</span
            >
        </p>
    {:else}
        <p>Ma nincs hangfelvételes szó.</p>
    {/if}
    {#if choices != null}
        <p>{choices} hangfelvételes szó választható.</p>
    {/if}
    <button type="button" class="admin-submit-btn" onclick={() => (dialogOpen = true)}>Beállítás</button>
</section>

{#if dialogOpen}
    <DailyDialog
        onClose={() => (dialogOpen = false)}
        onSaved={() => {
            load();
            onChanged();
        }}
    />
{/if}

<style>
    .daily-now {
        font-size: var(--text-lg);
    }
</style>
