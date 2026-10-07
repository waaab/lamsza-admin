<script>
    /**
     * Open suggestions from Szótár's signed-in users: accept (a new word is
     * created, an edit replaces the word's senses) or reject (moved from
     * Szótár's SuggestionList). The acting admin is recorded in Szótár's
     * `decided_by_email`.
     *
     * @type {{ szotarOrigin: string, onChanged?: () => void }}
     */
    let { szotarOrigin, onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";

    /** @type {{ id: number, word_id: number | null, headword: string, user_name: string, created_at: string }[]} */
    let rows = $state([]);
    let loaded = $state(false);
    let error = $state("");
    /** @type {{ text: string, wordId?: number } | null} */
    let notice = $state(null);
    /** @type {Record<number, string>} */
    let rowError = $state({});
    /** @type {Record<number, boolean>} */
    let busy = $state({});

    onMount(async () => {
        try {
            const data = await sectionFetch("dictionary", "suggestions");
            rows = Array.isArray(data) ? data : [];
        } catch (err) {
            error = errorText(err, "A javaslatok nem tölthetők be.");
        } finally {
            loaded = true;
        }
    });

    async function decide(/** @type {typeof rows[number]} */ row, /** @type {"accept" | "reject"} */ action) {
        if (busy[row.id]) return;
        busy = { ...busy, [row.id]: true };
        rowError = { ...rowError, [row.id]: "" };
        try {
            const result = await sectionFetch("dictionary", `suggestions/${row.id}/${action}`, { method: "POST" });
            rows = rows.filter((r) => r.id !== row.id);
            notice =
                action === "accept"
                    ? { text: `Elfogadva: ${row.headword}.`, wordId: result?.id }
                    : { text: `Elutasítva: ${row.headword}.` };
            onChanged();
        } catch (err) {
            rowError = {
                ...rowError,
                [row.id]: errorText(err, action === "accept" ? "Az elfogadás nem sikerült." : "Az elutasítás nem sikerült."),
            };
        } finally {
            busy = { ...busy, [row.id]: false };
        }
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

<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            <tr>
                <th>Típus</th>
                <th>Címszó</th>
                <th>Felhasználó</th>
                <th>Beküldve</th>
                <th class="admin-table-col--action">Művelet</th>
            </tr>
        </thead>
        <tbody>
            {#each rows as row (row.id)}
                <tr>
                    <td>{row.word_id == null ? "Új szó" : "Módosítás"}</td>
                    <td>
                        {#if row.word_id}
                            <a href={`${szotarOrigin}/szo/${row.word_id}`} target="_blank" rel="noopener noreferrer"
                                >{row.headword}</a
                            >
                        {:else}
                            {row.headword}
                        {/if}
                        {#if rowError[row.id]}
                            <div class="info-box error row-error" role="alert"><p>{rowError[row.id]}</p></div>
                        {/if}
                    </td>
                    <td>{row.user_name}</td>
                    <td>{formatHuDate(row.created_at)}</td>
                    <td class="admin-table-col--action">
                        <button
                            type="button"
                            class="admin-submit-btn"
                            disabled={busy[row.id]}
                            onclick={() => decide(row, "accept")}>Elfogad</button
                        >
                        <button type="button" class="btn btn-sm" disabled={busy[row.id]} onclick={() => decide(row, "reject")}
                            >Elutasít</button
                        >
                    </td>
                </tr>
            {:else}
                <tr><td colspan="5">{loaded ? "Nincs elbírálásra váró szójavaslat." : "Betöltés…"}</td></tr>
            {/each}
        </tbody>
    </table>
</div>

<style>
    .row-error {
        margin-top: 0.5rem;
    }
</style>
