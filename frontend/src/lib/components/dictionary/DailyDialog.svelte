<script>
    /**
     * Pin Szótár's daily word until a date, or hand it back to the automatic
     * rotation (moved from Szótár's DailyDialog). Only words with audio can be
     * picked; the backend refuses a past date and a word without audio.
     *
     * @type {{ onClose: () => void, onSaved?: (day: any) => void }}
     */
    let { onClose, onSaved = () => {} } = $props();

    import { onMount } from "svelte";
    import HuDateInput from "$lib/components/HuDateInput.svelte";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    /** @type {{ id: number, headword: string }[]} */
    let choices = $state([]);
    let selected = $state("");
    let until = $state("");
    let today = $state("");
    let pinned = $state(false);
    let error = $state("");
    let saving = $state(false);

    onMount(async () => {
        try {
            const [list, day] = await Promise.all([
                sectionFetch("dictionary", "daily/choices"),
                sectionFetch("dictionary", "daily"),
            ]);
            choices = list?.words || [];
            today = day?.today || "";
            pinned = !!day?.pinned;
            until = day?.pinned && day?.until ? day.until : day?.today || "";
            selected = day?.word ? String(day.word.id) : choices[0] ? String(choices[0].id) : "";
        } catch (err) {
            error = errorText(err, "A lista nem töltődött be.");
        }
    });

    async function run(/** @type {"POST" | "DELETE"} */ method) {
        saving = true;
        error = "";
        try {
            const day = await sectionFetch("dictionary", "daily", {
                method,
                body: method === "POST" ? { word_id: Number(selected), until } : undefined,
            });
            onSaved(day);
            onClose();
        } catch (err) {
            error = errorText(err, method === "POST" ? "A mentés nem sikerült." : "A visszaállítás nem sikerült.");
        } finally {
            saving = false;
        }
    }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
    class="link-dialog-overlay"
    role="dialog"
    tabindex="-1"
    aria-labelledby="dict-daily-title"
    onclick={(e) => e.target === e.currentTarget && onClose()}
    onkeydown={(e) => e.key === "Escape" && onClose()}
>
    <div class="link-dialog admin-modal">
        <h3 id="dict-daily-title">Napi Székely Szó</h3>
        <p class="admin-status">
            Csak hangfelvételes szó választható. A megadott nap végéig ez marad a napi szó, utána újra a sor
            következik.
        </p>
        <form
            class="admin-form"
            onsubmit={(e) => {
                e.preventDefault();
                run("POST");
            }}
        >
            <label for="dict-daily-word">Szó</label>
            <select id="dict-daily-word" name="word_id" bind:value={selected} required>
                <option value="">Válassz...</option>
                {#each choices as item (item.id)}
                    <option value={String(item.id)}>{item.headword}</option>
                {/each}
            </select>
            <label for="dict-daily-until">Meddig</label>
            <HuDateInput id="dict-daily-until" name="until" bind:value={until} min={today} required />
            {#if error}
                <div class="info-box error" role="alert"><p>{error}</p></div>
            {/if}
            <div class="admin-modal-actions">
                <button type="submit" class="admin-submit-btn" disabled={saving || !selected || !until}>Mentés</button>
                {#if pinned}
                    <button type="button" class="btn-update" disabled={saving} onclick={() => run("DELETE")}
                        >Automatikus</button
                    >
                {/if}
                <button type="button" class="btn-delete" onclick={onClose}>Mégse</button>
            </div>
        </form>
    </div>
</div>
