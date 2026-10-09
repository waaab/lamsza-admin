<script>
    /**
     * One Szótár suggestion, reviewed before deciding (lamsza
     * docs/PLAN_01_SUGGESTION_REVIEW.md): a per-field diff against the word as
     * it is now (a new word shows every field), the duplicate-headword, stale
     * and deleted-word warnings, and Elfogad / Elutasít with an optional reason.
     * A decided suggestion shows who decided, when and why, read-only.
     *
     * @type {{
     *   id: number,
     *   szotarOrigin: string,
     *   onDecided: (r: { action: "accept" | "reject", headword: string, wordId?: number }) => void,
     *   onClose: () => void,
     * }}
     */
    let { id, szotarOrigin, onDecided, onClose } = $props();

    import { onMount } from "svelte";
    import { CHANGE_LABEL, changedCount, suggestionDiff } from "$lib/suggestionDiff.js";
    import { errorText, formatHuDateTime, sectionFetch } from "$lib/sectionApi.js";

    /** @type {any} */
    let s = $state(null);
    let loadError = $state("");
    let actionError = $state("");
    let busy = $state(false);
    let rejecting = $state(false);
    let reason = $state("");
    let showSame = $state(false);

    const sections = $derived(s ? suggestionDiff(s.kind === "edit" ? s.current : null, s.payload) : []);
    const changes = $derived(changedCount(sections));
    const sameCount = $derived(sections.reduce((n, sec) => n + sec.rows.filter((r) => r.change === "same").length, 0));
    const isOpen = $derived(s?.status === "open");
    // A "Most" column only while there is a word to compare with.
    const compare = $derived(s?.kind === "edit" && !!s?.current);

    async function load() {
        loadError = "";
        try {
            s = await sectionFetch("dictionary", `suggestions/${id}`);
        } catch (err) {
            loadError = errorText(err, "A javaslat nem tölthető be.");
        }
    }
    onMount(load);

    async function accept() {
        if (busy) return;
        busy = true;
        actionError = "";
        try {
            const result = await sectionFetch("dictionary", `suggestions/${id}/accept`, {
                method: "POST",
                body: s.stale ? { confirm_stale: true } : {},
            });
            onDecided({ action: "accept", headword: s.headword, wordId: result?.id });
        } catch (err) {
            actionError = errorText(err, "Az elfogadás nem sikerült.");
            // The word changed or was deleted while this screen was open:
            // reload, so the warning and the diff show the new state.
            const code = /** @type {{ code?: string }} */ (err).code;
            if (code === "STALE_EDIT" || code === "WORD_DELETED") await load();
        } finally {
            busy = false;
        }
    }

    async function reject() {
        if (busy) return;
        busy = true;
        actionError = "";
        try {
            await sectionFetch("dictionary", `suggestions/${id}/reject`, {
                method: "POST",
                body: { reason: reason.trim() },
            });
            onDecided({ action: "reject", headword: s.headword });
        } catch (err) {
            actionError = errorText(err, "Az elutasítás nem sikerült.");
        } finally {
            busy = false;
        }
    }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
    class="link-dialog-overlay"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    aria-labelledby="suggestion-review-title"
    onclick={(e) => e.target === e.currentTarget && !busy && onClose()}
    onkeydown={(e) => e.key === "Escape" && !busy && onClose()}
>
    <div class="link-dialog admin-modal review">
        {#if loadError}
            <h3 id="suggestion-review-title">Szójavaslat</h3>
            <div class="info-box error" role="alert"><p>{loadError}</p></div>
            <div class="review-actions"><button type="button" class="btn" onclick={onClose}>Bezárás</button></div>
        {:else if !s}
            <h3 id="suggestion-review-title">Szójavaslat</h3>
            <p aria-busy="true">Betöltés…</p>
        {:else}
            <h3 id="suggestion-review-title">
                {s.kind === "new" ? "Új szó" : "Módosítás"}: {s.headword}
            </h3>
            <p class="review-meta">
                Beküldte: {s.user_name || "ismeretlen"}, {formatHuDateTime(s.created_at)}
                {#if s.kind === "edit" && s.word_id}
                    ·
                    <a href={`${szotarOrigin}/szo/${s.word_id}`} target="_blank" rel="noopener noreferrer">A szó a Szótárban</a>
                {/if}
            </p>

            {#if s.word_deleted}
                <div class="info-box error" role="alert">
                    <p>A szót időközben törölték. Ezt a javaslatot csak elutasítani lehet.</p>
                </div>
            {/if}
            {#if s.stale}
                <div class="info-box warning" role="alert">
                    <p>
                        A szót a javaslat beküldése ({formatHuDateTime(s.base_updated_at)}) óta módosították
                        ({formatHuDateTime(s.current_updated_at)}). Az összevetés a mostani állapothoz méri a javaslatot, és az
                        elfogadás felülírja a későbbi módosításokat is.
                    </p>
                </div>
            {/if}
            {#if s.duplicates?.length}
                <div class="info-box warning" role="status">
                    <p>
                        <span>
                            Ez a címszó már szerepel a szótárban:
                            {#each s.duplicates as d, i (d.id)}{#if i > 0}, {/if}<a
                                    href={`${szotarOrigin}/szo/${d.id}`}
                                    target="_blank"
                                    rel="noopener noreferrer">{d.headword}</a
                                > (<a href={`#szavak/${d.id}`} onclick={onClose}>szerkesztés</a>){/each}. Az elfogadás egy új,
                            külön szót hoz létre mellette.
                        </span>
                    </p>
                </div>
            {/if}

            {#if s.kind === "edit" && s.current && changes === 0}
                <div class="info-box info" role="status"><p>A javaslat semmin nem változtat.</p></div>
            {/if}

            {#each sections as sec (sec.title)}
                {@const rows = sec.rows.filter((r) => showSame || r.change !== "same")}
                {#if rows.length}
                    <h4 class="review-section">{sec.title}</h4>
                    <div class="admin-table-wrapper">
                        <table class="admin-table review-diff">
                            <colgroup>
                                <col class="col-field" />
                                {#if compare}<col />{/if}
                                <col />
                            </colgroup>
                            <thead>
                                <tr>
                                    <th scope="col">Mező</th>
                                    {#if compare}<th scope="col">Most</th>{/if}
                                    <th scope="col">Javasolt</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each rows as r (r.key)}
                                    <tr class={`change-${r.change}`}>
                                        <th scope="row">
                                            {r.label}
                                            {#if r.change !== "same"}<span class="change-tag">{CHANGE_LABEL[r.change]}</span>{/if}
                                        </th>
                                        {#if compare}<td class="before">{r.before || "-"}</td>{/if}
                                        <td class="after">{r.after || "-"}</td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            {/each}
            {#if compare && sameCount > 0}
                <button type="button" class="btn btn-sm review-toggle" onclick={() => (showSame = !showSame)}>
                    {showSame ? "Változatlan mezők elrejtése" : `Változatlan mezők megjelenítése (${sameCount})`}
                </button>
            {/if}

            {#if !isOpen}
                <div class="info-box {s.status === 'accepted' ? 'success' : 'brown'}" role="status">
                    <p>
                        {s.status === "accepted" ? "Elfogadva" : "Elutasítva"}.
                        Elbírálta: {s.decided_by_email || "ismeretlen"}, {formatHuDateTime(s.decided_at)}.
                        {#if s.reject_reason}Indoklás: {s.reject_reason}{/if}
                    </p>
                </div>
            {/if}

            {#if actionError}
                <div class="info-box error" role="alert"><p>{actionError}</p></div>
            {/if}

            {#if isOpen && rejecting}
                <label class="review-reason" for="suggestion-reject-reason">
                    Indoklás a beküldőnek (nem kötelező)
                    <textarea id="suggestion-reject-reason" rows="3" maxlength="500" bind:value={reason}></textarea>
                </label>
            {/if}
            <div class="review-actions">
                {#if isOpen && rejecting}
                    <button type="button" class="btn-delete" disabled={busy} onclick={reject}>Elutasítás megerősítése</button>
                    <button type="button" class="btn" disabled={busy} onclick={() => (rejecting = false)}>Mégse</button>
                {:else if isOpen}
                    {#if !s.word_deleted}
                        <button type="button" class="admin-submit-btn" disabled={busy} onclick={accept}>
                            {s.stale ? "Elfogadom így is" : "Elfogad"}
                        </button>
                    {/if}
                    <button type="button" class="btn" disabled={busy} onclick={() => (rejecting = true)}>Elutasít</button>
                    <button type="button" class="btn" disabled={busy} onclick={onClose}>Bezárás</button>
                {:else}
                    <button type="button" class="btn" onclick={onClose}>Bezárás</button>
                {/if}
            </div>
        {/if}
    </div>
</div>

<style>
    .review {
        max-width: 56rem;
        width: 100%;
        max-height: 90vh;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
    }
    /* The modal scrolls as a whole; its parts keep their height instead of
       shrinking to fit (they would spill out of their boxes on a phone). */
    .review > :global(*) {
        flex-shrink: 0;
    }
    .review h3 {
        margin: 0;
    }
    .review-meta {
        margin: 0;
        color: var(--text-muted);
        font-size: var(--text-sm);
    }
    .review-section {
        margin: 0.5rem 0 0;
        font-size: var(--text-base);
    }
    /* Fixed layout: the field names keep their column and "Most" and
       "Javasolt" share the rest evenly, whatever their content. */
    .review-diff {
        table-layout: fixed;
        width: 100%;
        /* The admin table's 720px minimum would push "Javasolt" off a phone. */
        min-width: 0;
    }
    .review-diff th {
        white-space: normal;
    }
    .review-diff .col-field {
        width: 11rem;
    }
    .review-diff th[scope="row"] {
        font-weight: 600;
        vertical-align: top;
        overflow-wrap: anywhere;
    }
    @media (max-width: 640px) {
        .review-diff .col-field {
            width: 6.5rem;
        }
    }
    .review-diff td {
        white-space: pre-wrap;
        overflow-wrap: anywhere;
        vertical-align: top;
    }
    .review-diff .change-added .after,
    .review-diff .change-changed .after {
        background: color-mix(in srgb, var(--szekely-green) 16%, transparent);
    }
    .review-diff .change-removed .before,
    .review-diff .change-changed .before {
        background: color-mix(in srgb, var(--szekely-red) 12%, transparent);
    }
    .review-diff .change-removed .before {
        text-decoration: line-through;
    }
    .change-tag {
        display: block;
        font-weight: 400;
        font-size: var(--text-xs);
        color: var(--text-muted);
    }
    .review-toggle {
        align-self: flex-start;
    }
    .review-reason {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
        font-size: var(--text-sm);
    }
    .review-reason textarea {
        width: 100%;
        box-sizing: border-box;
        font: inherit;
    }
    .review-actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
        justify-content: flex-end;
    }
</style>
