<script>
    /**
     * The Téma panel of a Beállítások page (UI_BASELINE "set-theme-ui"):
     * three buttons that preview the theme at once, and Mentés, which saves
     * it to the signed-in user's account. Shared from lamsza by
     * scripts/sync-shared-frontend.sh; Szótár and Játszótér use it, Lámsza's
     * own Beállítások has the same look.
     *
     *   <ThemeSettings current={$theme} labels={LABELS} onPreview={applyTheme}
     *       onSave={saveTheme} />
     */

    /**
     * @type {{
     *   current: string,
     *   labels: Record<string, string>,
     *   onPreview: (theme: string) => void,
     *   onSave: (theme: string) => Promise<void>,
     * }}
     */
    let { current, labels, onPreview, onSave } = $props();

    let selected = $state("");
    let saving = $state(false);
    let ok = $state("");
    let error = $state("");

    const shown = $derived(selected || current);

    /** @param {string} theme */
    function choose(theme) {
        selected = theme;
        ok = "";
        error = "";
        onPreview(theme);
    }

    async function save() {
        saving = true;
        ok = "";
        error = "";
        try {
            await onSave(shown);
            ok = "Elmentve.";
        } catch (err) {
            error = /** @type {Error} */ (err)?.message || "Nem sikerült menteni.";
        } finally {
            saving = false;
        }
    }
</script>

<div class="theme-settings">
    <h3>Téma</h3>
    <p class="theme-settings-hint">Kijelentkezve az eszközöd témája érvényes.</p>
    <div class="theme-settings-buttons" role="group" aria-label="Téma">
        {#each ["light", "dark", "system"] as theme (theme)}
            <button type="button" class="btn" class:active={shown === theme} aria-pressed={shown === theme} onclick={() => choose(theme)}>
                {labels[theme]}
            </button>
        {/each}
    </div>
    {#if error}
        <p class="theme-settings-error" role="alert">{error}</p>
    {/if}
    {#if ok}
        <p class="theme-settings-ok" role="status">{ok}</p>
    {/if}
    <button type="button" class="btn" disabled={saving} onclick={save}>{saving ? "Mentés…" : "Mentés"}</button>
</div>
