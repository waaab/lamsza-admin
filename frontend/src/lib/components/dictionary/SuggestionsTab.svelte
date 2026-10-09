<script>
    /**
     * Szótár's word suggestions (lamsza docs/PLAN_01_SUGGESTION_REVIEW.md).
     * Függőben lists the open ones with their warnings; each opens in
     * SuggestionReview, where the admin sees the per-field diff and decides.
     * There is no one-click accept from the table. Elbírált lists the latest
     * decisions with who decided, when and why. The acting admin is recorded in
     * Szótár's `decided_by_email`.
     *
     * @type {{ szotarOrigin: string, onChanged?: () => void }}
     */
    let { szotarOrigin, onChanged = () => {} } = $props();

    import { errorText, formatHuDate, formatHuDateTime, sectionFetch } from "$lib/sectionApi.js";
    import SuggestionReview from "$lib/components/dictionary/SuggestionReview.svelte";

    /**
     * @typedef {{
     *   id: number, kind: "new" | "edit", word_id: number | null, headword: string, user_name: string,
     *   created_at: string, status: string, word_deleted: boolean, stale: boolean,
     *   duplicates: { id: number, headword: string }[],
     *   decided_at: string | null, decided_by_email: string, reject_reason: string
     * }} Row
     */

    /** @type {"open" | "decided"} */
    let view = $state("open");
    /** @type {Row[]} */
    let rows = $state([]);
    let loaded = $state(false);
    let error = $state("");
    /** @type {{ text: string, wordId?: number } | null} */
    let notice = $state(null);
    /** @type {number | null} */
    let reviewing = $state(null);

    async function load() {
        loaded = false;
        error = "";
        try {
            const data = await sectionFetch("dictionary", view === "open" ? "suggestions" : "suggestions?status=decided");
            rows = Array.isArray(data) ? data : [];
        } catch (err) {
            error = errorText(err, "A javaslatok nem tölthetők be.");
            rows = [];
        } finally {
            loaded = true;
        }
    }

    $effect(() => {
        view;
        load();
    });

    /** @param {{ action: "accept" | "reject", headword: string, wordId?: number }} r */
    function decided(r) {
        reviewing = null;
        notice = r.action === "accept" ? { text: `Elfogadva: ${r.headword}.`, wordId: r.wordId } : { text: `Elutasítva: ${r.headword}.` };
        onChanged();
        load();
    }

    /** @param {Row} row */
    function typeLabel(row) {
        return row.kind === "new" ? "Új szó" : "Módosítás";
    }
</script>

{#if notice}
    <div class="info-box success" role="status">
        <p>
            {notice.text}
            {#if notice.wordId}
                <a href={`${szotarOrigin}/szo/${notice.wordId}`} target="_blank" rel="noopener noreferrer"
                    >Megnyitás a Szótárban</a
                >
            {/if}
        </p>
    </div>
{/if}
{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

<div class="suggestion-views" role="group" aria-label="Szójavaslatok szűrése">
    <button type="button" class="btn btn-sm" aria-pressed={view === "open"} onclick={() => (view = "open")}>Függőben</button>
    <button type="button" class="btn btn-sm" aria-pressed={view === "decided"} onclick={() => (view = "decided")}>Elbírált</button>
</div>

<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            {#if view === "open"}
                <tr>
                    <th>Típus</th>
                    <th>Címszó</th>
                    <th>Felhasználó</th>
                    <th>Beküldve</th>
                    <th class="admin-table-col--action">Művelet</th>
                </tr>
            {:else}
                <tr>
                    <th>Típus</th>
                    <th>Címszó</th>
                    <th>Döntés</th>
                    <th>Elbírálta</th>
                    <th>Mikor</th>
                    <th class="admin-table-col--action">Művelet</th>
                </tr>
            {/if}
        </thead>
        <tbody>
            {#each rows as row (row.id)}
                <tr>
                    <td>{typeLabel(row)}</td>
                    <td>
                        {#if row.word_id}
                            <a href={`${szotarOrigin}/szo/${row.word_id}`} target="_blank" rel="noopener noreferrer"
                                >{row.headword}</a
                            >
                        {:else}
                            {row.headword}
                        {/if}
                        {#if view === "open"}
                            {#if row.duplicates?.length}<span class="flag flag--warn">Már létezik</span>{/if}
                            {#if row.stale}<span class="flag flag--warn">Elavult</span>{/if}
                            {#if row.word_deleted}<span class="flag flag--error">Törölt szó</span>{/if}
                        {:else if row.reject_reason}
                            <div class="row-reason">Indoklás: {row.reject_reason}</div>
                        {/if}
                    </td>
                    {#if view === "open"}
                        <td>{row.user_name}</td>
                        <td>{formatHuDate(row.created_at)}</td>
                    {:else}
                        <td>{row.status === "accepted" ? "Elfogadva" : "Elutasítva"}</td>
                        <td>{row.decided_by_email || "-"}</td>
                        <td>{formatHuDateTime(row.decided_at)}</td>
                    {/if}
                    <td class="admin-table-col--action">
                        <button type="button" class={view === "open" ? "admin-submit-btn" : "btn btn-sm"} onclick={() => (reviewing = row.id)}
                            >{view === "open" ? "Megnyitás" : "Részletek"}</button
                        >
                    </td>
                </tr>
            {:else}
                <tr>
                    <td colspan={view === "open" ? 5 : 6}>
                        {#if !loaded}Betöltés…{:else if view === "open"}Nincs elbírálásra váró szójavaslat.{:else}Még nincs elbírált szójavaslat.{/if}
                    </td>
                </tr>
            {/each}
        </tbody>
    </table>
</div>

{#if reviewing}
    {#key reviewing}
        <SuggestionReview id={reviewing} {szotarOrigin} onDecided={decided} onClose={() => (reviewing = null)} />
    {/key}
{/if}

<style>
    .suggestion-views {
        display: flex;
        gap: 0.5rem;
        margin-bottom: 0.75rem;
    }
    .suggestion-views [aria-pressed="true"] {
        border-color: var(--primary);
        color: var(--primary);
        font-weight: 600;
    }
    .flag {
        display: inline-block;
        margin-left: 0.4rem;
        padding: 0.05rem 0.45rem;
        border-radius: 6px;
        font-size: var(--text-xs);
        font-weight: 600;
        border: 1px solid;
    }
    .flag--warn {
        border-color: var(--warning-note-border);
        background: var(--warning-note-bg);
    }
    .flag--error {
        border-color: var(--szekely-red);
        background: color-mix(in srgb, var(--szekely-red) 14%, transparent);
    }
    .row-reason {
        margin-top: 0.25rem;
        font-size: var(--text-sm);
        color: var(--text-muted);
    }
</style>
