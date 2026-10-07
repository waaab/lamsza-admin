<script>
    /**
     * One Rovásfejtő puzzle: the Latin-letter text, its type and difficulty
     * (set by hand, not derived from the length), with a rune preview (moved
     * from Játszótér's /admin/rovasfejto). A text that has no rovás encoding
     * gets the backend's ENCODE_ERROR message.
     *
     * @type {{
     *   puzzle?: { id: string, text: string, category: string, difficulty: number } | null,
     *   categories: { id: string, label_hu: string }[],
     *   difficulties: { level: number, label_hu: string }[],
     *   idPrefix?: string,
     *   onSaved: (id: string, created: boolean) => void,
     *   onCancel?: (() => void) | null,
     * }}
     */
    let { puzzle = null, categories, difficulties, idPrefix = "rovas", onSaved, onCancel = null } = $props();

    import { untrack } from "svelte";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    let text = $state(untrack(() => puzzle?.text ?? ""));
    let category = $state(untrack(() => puzzle?.category ?? categories.find((c) => c.id === "kozmondas")?.id ?? categories[0]?.id ?? ""));
    let difficulty = $state(untrack(() => puzzle?.difficulty ?? 2));
    /** @type {{ tokens: { kind: string, symbol: string }[] } | null} */
    let preview = $state(null);
    let error = $state("");
    let saving = $state(false);

    async function runPreview() {
        if (!text.trim()) return;
        error = "";
        try {
            preview = await sectionFetch("games", "rovasfejto/preview", { method: "POST", body: { text } });
        } catch (err) {
            preview = null;
            error = errorText(err, "Az előnézet nem sikerült.");
        }
    }

    async function save(/** @type {SubmitEvent} */ e) {
        e.preventDefault();
        saving = true;
        error = "";
        const body = { text, category, difficulty: Number(difficulty) };
        try {
            if (puzzle) {
                await sectionFetch("games", `rovasfejto/puzzles/${puzzle.id}`, { method: "PUT", body });
                onSaved(puzzle.id, false);
            } else {
                const res = await sectionFetch("games", "rovasfejto/puzzles", { method: "POST", body });
                text = "";
                preview = null;
                onSaved(res?.puzzle?.id ?? "", true);
            }
        } catch (err) {
            error = errorText(err, "A mentés nem sikerült.");
        } finally {
            saving = false;
        }
    }
</script>

<form class="admin-form" onsubmit={save}>
    <label for="{idPrefix}-text">Szöveg (latin betűk)</label>
    <textarea id="{idPrefix}-text" name="text" bind:value={text} rows="3" required placeholder="Írd be a közmondást…"
    ></textarea>
    <label for="{idPrefix}-category">Típus</label>
    <select id="{idPrefix}-category" name="category" bind:value={category} required>
        <option value="">Válassz...</option>
        {#each categories as cat (cat.id)}
            <option value={cat.id}>{cat.label_hu}</option>
        {/each}
    </select>
    <label for="{idPrefix}-difficulty">Nehézség</label>
    <select id="{idPrefix}-difficulty" name="difficulty" bind:value={difficulty} required>
        {#each difficulties as d (d.level)}
            <option value={d.level}>{d.label_hu}</option>
        {/each}
    </select>
    {#if preview}
        <div class="rovas-preview">
            <p class="rovas-preview-label">Rovás előnézet</p>
            <p class="rovas-preview-runes">
                {#each preview.tokens as tok, i (i)}{tok.kind === "space" ? " " : tok.symbol}{/each}
            </p>
        </div>
    {/if}
    {#if error}
        <div class="info-box error" role="alert"><p>{error}</p></div>
    {/if}
    <div class="admin-modal-actions">
        <button type="button" class="btn-update" onclick={runPreview} disabled={!text.trim()}>Rovás előnézet</button>
        <button type="submit" class="admin-submit-btn" disabled={saving || !text.trim()}
            >{saving ? "Mentés…" : puzzle ? "Mentés" : "Létrehozás"}</button
        >
        {#if onCancel}
            <button type="button" class="btn-delete" onclick={onCancel}>Mégse</button>
        {/if}
    </div>
</form>

<style>
    .rovas-preview {
        padding: 0.85rem;
        border: 1px solid var(--border-color);
        border-radius: 6px;
        background: var(--card-bg);
    }
    .rovas-preview-label {
        margin: 0 0 0.5rem;
        font-size: var(--text-xs);
        text-transform: uppercase;
        letter-spacing: 0.1em;
        color: var(--text-muted);
    }
    /* Játszótér's --font-rovas stack: system fonts only, no web font. */
    .rovas-preview-runes {
        margin: 0;
        font-family: "Noto Sans Old Hungarian", "Segoe UI Symbol", sans-serif;
        font-size: var(--text-xl);
        line-height: 1.5;
        word-break: break-word;
    }
</style>
