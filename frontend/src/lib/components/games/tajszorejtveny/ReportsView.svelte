<script>
    /**
     * "Hibát találtam" reports from players (spec §10.5): the clue, the answer
     * (the admin may see it), the reason and the note; resolve, dismiss or
     * reopen with an optional note. A Székely word links to its entry in
     * /dictionary, where a wrong definition is fixed at the source.
     *
     * @type {{ onChanged?: () => void }}
     */
    let { onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import AdminPaginationBar from "$lib/components/admin/AdminPaginationBar.svelte";
    import { adminPageSlice } from "$lib/adminPageSlice.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";
    import { LEVEL_LABELS, REASON_LABELS, REPORT_STATUS_LABELS } from "$lib/tajszorejtvenyAdmin.js";

    /** @type {any[]} */
    let rows = $state([]);
    let status = $state("open");
    let loading = $state(true);
    let error = $state("");
    let notice = $state("");
    /** @type {Record<string, string>} */
    let notes = $state({});
    /** @type {Record<string, boolean>} */
    let busy = $state({});
    let page = $state(1);
    const pg = $derived(/** @type {ReturnType<typeof adminPageSlice> & { rows: any[] }} */ (adminPageSlice(rows, page)));

    async function load() {
        loading = true;
        try {
            const data = await sectionFetch("games", `tajszorejtveny/reports?status=${status}`);
            rows = data?.reports ?? [];
            notes = Object.fromEntries(rows.map((r) => [r.id, r.admin_note ?? ""]));
            error = "";
        } catch (err) {
            error = errorText(err, "A jelzések nem tölthetők be.");
            rows = [];
        } finally {
            loading = false;
        }
    }

    onMount(load);

    async function decide(/** @type {any} */ row, /** @type {"resolved" | "dismissed" | "open"} */ next) {
        if (busy[row.id]) return;
        busy = { ...busy, [row.id]: true };
        error = "";
        try {
            await sectionFetch("games", `tajszorejtveny/reports/${row.id}`, {
                method: "POST",
                body: { status: next, note: notes[row.id] ?? "" },
            });
            notice = `${row.answer}: ${REPORT_STATUS_LABELS[next].toLowerCase()}.`;
            await load();
            onChanged();
        } catch (err) {
            error = errorText(err, "A jelzés nem menthető.");
        } finally {
            busy = { ...busy, [row.id]: false };
        }
    }
</script>

<div class="header-tabs" role="group" aria-label="Jelzések állapota">
    {#each [["open", "Nyitott"], ["resolved", "Megoldva"], ["dismissed", "Elvetve"], ["all", "Mind"]] as [id, label] (id)}
        <button type="button" class="btn" class:active={status === id} onclick={() => ((status = id), (page = 1), load())}>{label}</button>
    {/each}
</div>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}
{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}

<AdminPaginationBar
    total={pg.total}
    page={pg.page}
    totalPages={pg.totalPages}
    from={pg.from}
    to={pg.to}
    on:prev={() => (page = Math.max(1, page - 1))}
    on:next={() => (page = Math.min(pg.totalPages, page + 1))}
/>
<div class="admin-table-wrapper">
    <table class="admin-table">
        <thead>
            <tr>
                <th>Beküldve</th>
                <th>Rejtvény</th>
                <th>Szó és meghatározás</th>
                <th>Ok</th>
                <th>Admin megjegyzés</th>
                <th class="admin-table-col--action">Művelet</th>
            </tr>
        </thead>
        <tbody>
            {#each pg.rows as r (r.id)}
                <tr>
                    <td>{formatHuDate(r.created_at)}</td>
                    <td>{r.date ? formatHuDate(r.date) : "gyakorló"}<div class="taj-muted">{LEVEL_LABELS[r.level] ?? r.level}</div></td>
                    <td>
                        <strong>{r.answer}</strong>
                        {#if r.szotar_id && /^\d+$/.test(r.szotar_id)}
                            · <a href="/dictionary#szavak/{r.szotar_id}">Szótár</a>
                        {/if}
                        <div>„{r.clue}”</div>
                    </td>
                    <td>
                        {REASON_LABELS[r.reason] ?? r.reason}
                        {#if r.note}<div class="taj-muted">{r.note}</div>{/if}
                        {#if status === "all"}<div class="taj-muted">{REPORT_STATUS_LABELS[r.status] ?? r.status}</div>{/if}
                    </td>
                    <td>
                        <input
                            class="admin-search-input"
                            aria-label="Admin megjegyzés"
                            maxlength="500"
                            bind:value={notes[r.id]}
                            disabled={busy[r.id]}
                        />
                    </td>
                    <td class="taj-row-actions">
                        {#if r.status === "open"}
                            <button type="button" class="btn-update" disabled={busy[r.id]} onclick={() => decide(r, "resolved")}>Megoldva</button>
                            <button type="button" class="btn btn-sm" disabled={busy[r.id]} onclick={() => decide(r, "dismissed")}>Elvetés</button>
                        {:else}
                            <button type="button" class="btn btn-sm" disabled={busy[r.id]} onclick={() => decide(r, "open")}>Újranyitás</button>
                        {/if}
                    </td>
                </tr>
            {:else}
                <tr><td colspan="6">{loading ? "Betöltés…" : "Nincs jelzés."}</td></tr>
            {/each}
        </tbody>
    </table>
</div>

<style>
    .taj-muted {
        color: var(--text-muted);
        font-size: var(--text-sm);
    }

    .taj-row-actions {
        white-space: nowrap;
    }

    .taj-row-actions button + button {
        margin-left: 0.35rem;
    }
</style>
