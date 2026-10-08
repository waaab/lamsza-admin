<script>
    /**
     * Tájszórejtvény weeks (spec §10.3, §16.6.4): this week and next week as a
     * 7 × 4 matrix (day × level) with each puzzle's status and flags, the
     * spares, "Az egész hét jóváhagyása" and "Generálás most". Next week is
     * generated on Wednesday at 22:00 and goes live on Monday at 0:00, approved
     * or not. A cell opens the puzzle below the matrix.
     *
     * @type {{ onChanged?: () => void }}
     */
    let { onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import PuzzleView from "$lib/components/games/tajszorejtveny/PuzzleView.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";
    import { FLAG_LABELS, LEVELS, LEVEL_LABELS, STATUS_LABELS, dayName, waitForTask, weekMatrix } from "$lib/tajszorejtvenyAdmin.js";

    /** @type {any} */
    let data = $state(null);
    let which = $state(/** @type {"this" | "next"} */ ("next"));
    let error = $state("");
    let notice = $state("");
    let busy = $state("");
    let openId = $state("");

    const week = $derived(data ? data[which] : null);
    const matrix = $derived(weekMatrix(week));
    const counts = $derived(week?.counts ?? {});
    const waiting = $derived((counts.draft ?? 0) + (counts.needs_attention ?? 0));
    const generating = $derived((data?.tasks ?? []).find((/** @type {any} */ t) => t.kind === "generate" && t.running) ?? null);

    async function load() {
        try {
            data = await sectionFetch("games", "tajszorejtveny/weeks");
            error = "";
        } catch (err) {
            error = errorText(err, "A hetek nem tölthetők be.");
        }
    }

    onMount(load);

    async function approveWeek() {
        if (!week || busy) return;
        const ok = await confirmDialog(
            `Jóváhagyod a hét összes piszkozatát (${counts.draft ?? 0})? Az átnézendőket (${counts.needs_attention ?? 0}) egyenként kell jóváhagyni.`,
            { yesLabel: "Jóváhagyás" },
        );
        if (!ok) return;
        busy = "approve";
        error = "";
        try {
            const res = await sectionFetch("games", `tajszorejtveny/weeks/${week.week_start}/approve`, { method: "POST" });
            notice = `${res.approved} rejtvény jóváhagyva.${res.needs_attention ? ` ${res.needs_attention} átnézendő maradt.` : ""}`;
            await load();
            onChanged();
        } catch (err) {
            error = errorText(err, "A jóváhagyás nem sikerült.");
        } finally {
            busy = "";
        }
    }

    async function generate() {
        if (!week || busy) return;
        const exists = (counts.total ?? 0) > 0;
        const ok = await confirmDialog(
            exists
                ? "Elkészítjük a hét hiányzó rejtvényeit? A meglévők nem változnak."
                : "Elkészítjük a hét rejtvényeit most? Néhány percig tart.",
            { yesLabel: "Generálás" },
        );
        if (!ok) return;
        busy = "generate";
        error = "";
        notice = "";
        try {
            const task = await sectionFetch("games", `tajszorejtveny/weeks/${week.week_start}/generate`, { method: "POST" });
            await load();
            const res = await waitForTask(() => sectionFetch("games", "tajszorejtveny/tasks"), task.key, { interval: 3000 });
            await load();
            if (res.error) error = res.error;
            else if (res.done) notice = "A hét rejtvényei elkészültek.";
            onChanged();
        } catch (err) {
            // 409: this week is already being generated (the scheduled job or another admin).
            if (/** @type {any} */ (err)?.status === 409) notice = "A hét generálása már fut. Néhány perc múlva töltsd újra a hetet.";
            else error = errorText(err, "A generálás nem indult el.");
            await load();
        } finally {
            busy = "";
        }
    }

    const cellTitle = (/** @type {any} */ c) =>
        [STATUS_LABELS[c.status] ?? c.status, ...(c.flags ?? []).map((/** @type {string} */ f) => FLAG_LABELS[f] ?? f)].join("\n");
</script>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}
{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}

{#if !data}
    {#if !error}<p>Betöltés…</p>{/if}
{:else}
    <div class="taj-week-bar">
        <div class="header-tabs" role="group" aria-label="Hét">
            <button type="button" class="btn" class:active={which === "this"} onclick={() => ((which = "this"), (openId = ""))}
                >Ez a hét ({formatHuDate(data.this_week)})</button
            >
            <button type="button" class="btn" class:active={which === "next"} onclick={() => ((which = "next"), (openId = ""))}
                >Jövő hét ({formatHuDate(data.next_week)})</button
            >
        </div>
        <div class="taj-week-actions">
            {#if which === "next" && waiting > 0}
                <button type="button" class="admin-submit-btn" disabled={!!busy} onclick={approveWeek}
                    >{busy === "approve" ? "Jóváhagyás…" : "Az egész hét jóváhagyása"}</button
                >
            {/if}
            <button type="button" class="btn-update" disabled={!!busy || !!generating} onclick={generate}
                >{busy === "generate" || generating ? "Generálás folyamatban…" : "Generálás most"}</button
            >
        </div>
    </div>

    <p class="taj-counts">
        {counts.total ?? 0}/28 rejtvény · {counts.approved ?? 0} jóváhagyva · {counts.draft ?? 0} piszkozat ·
        {counts.needs_attention ?? 0} átnézendő · {counts.live ?? 0} élő{counts.unreviewed_live
            ? ` (${counts.unreviewed_live} átnézés nélkül)`
            : ""}
        {#if week.generated_at}<span class="taj-muted"> · elkészült: {formatHuDate(week.generated_at)}</span>{/if}
    </p>
    {#if (counts.total ?? 0) === 0}
        <div class="info-box info" role="status">
            <p>Erre a hétre még nincs rejtvény. A jövő hetiek szerdán 22:00-kor készülnek el; a Generálás most azonnal elindítja.</p>
        </div>
    {/if}

    <div class="admin-table-wrapper">
        <table class="admin-table taj-matrix">
            <thead>
                <tr>
                    <th>Nap</th>
                    {#each LEVELS as level (level)}
                        <th>{LEVEL_LABELS[level]}</th>
                    {/each}
                </tr>
            </thead>
            <tbody>
                {#each matrix as row (row.date)}
                    <tr class:taj-today={row.date === data.today}>
                        <td>
                            <strong>{dayName(row.date)}</strong>
                            <div class="taj-muted">{formatHuDate(row.date)}</div>
                        </td>
                        {#each row.cells as c, i (i)}
                            <td>
                                {#if c}
                                    <button
                                        type="button"
                                        class="taj-cell taj-cell--{c.status}"
                                        class:open={openId === c.id}
                                        title={cellTitle(c)}
                                        onclick={() => (openId = openId === c.id ? "" : c.id)}
                                    >
                                        <span class="taj-cell-status">{STATUS_LABELS[c.status] ?? c.status}</span>
                                        <span class="taj-cell-meta"
                                            >{c.stats.szekely} tájszó · {c.stats.decorative} üres{c.stats.soft_repeats
                                                ? ` · ${c.stats.soft_repeats} ism.`
                                                : ""}</span
                                        >
                                        {#if c.flags.length}<span class="taj-cell-flag">⚠ {c.flags.length}</span>{/if}
                                    </button>
                                {:else}
                                    <span class="taj-muted">nincs</span>
                                {/if}
                            </td>
                        {/each}
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>

    {#if week.spares.length}
        <p class="taj-spares">
            Tartalékok:
            {#each LEVELS as level (level)}
                {@const list = week.spares.filter((/** @type {any} */ s) => s.level === level)}
                {#if list.length}
                    <span class="taj-spare-group"
                        >{LEVEL_LABELS[level]}:
                        {#each list as s, i (s.id)}
                            <button type="button" class="taj-link" onclick={() => (openId = s.id)}>{i + 1}.</button>
                        {/each}
                    </span>
                {/if}
            {/each}
        </p>
    {/if}

    {#if openId}
        <PuzzleView
            id={openId}
            spares={week.spares}
            onChanged={() => {
                load();
                onChanged();
            }}
            onClose={() => (openId = "")}
            onOpen={(id) => (openId = id)}
        />
    {/if}
{/if}

<style>
    .taj-week-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        margin-bottom: 0.75rem;
    }

    .taj-week-actions {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
    }

    .taj-counts {
        margin: 0 0 0.75rem;
    }

    .taj-muted {
        color: var(--text-muted);
        font-size: var(--text-sm);
    }

    .taj-matrix td {
        vertical-align: top;
    }

    .taj-matrix tr.taj-today td:first-child {
        box-shadow: inset 3px 0 0 var(--szekely-green);
    }

    .taj-cell {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 0.15rem;
        width: 100%;
        min-width: 8.5rem;
        padding: 0.4rem 0.5rem;
        border: 1px solid var(--border-color);
        border-radius: 6px;
        background: var(--card-bg);
        color: var(--text-primary);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .taj-cell.open {
        outline: 2px solid var(--szekely-green);
    }

    .taj-cell--approved,
    .taj-cell--live {
        border-color: color-mix(in srgb, var(--szekely-green) 60%, var(--border-color));
    }

    .taj-cell--needs_attention {
        border-color: var(--szekely-red);
    }

    .taj-cell-status {
        font-weight: 700;
    }

    .taj-cell-meta {
        font-size: var(--text-xs);
        color: var(--text-muted);
    }

    .taj-cell-flag {
        font-size: var(--text-xs);
        color: var(--szekely-red);
        font-weight: 600;
    }

    .taj-spares {
        display: flex;
        flex-wrap: wrap;
        gap: 0.75rem;
    }

    .taj-link {
        padding: 0 0.15rem;
        border: none;
        background: none;
        color: var(--link-color, var(--szekely-green));
        font: inherit;
        font-weight: 700;
        cursor: pointer;
    }
</style>
