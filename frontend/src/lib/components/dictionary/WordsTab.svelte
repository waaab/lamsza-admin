<script>
    /**
     * Szótár words: search, letter filter, create, live edit, delete (moved
     * from Szótár's /admin/szavak). `openId` opens that word's editor, for the
     * `#szavak/<id>` deep link from Szótár's entry page.
     *
     * @type {{ szotarOrigin: string, openId?: number | null, onChanged?: () => void }}
     */
    let { szotarOrigin, openId = null, onChanged = () => {} } = $props();

    import { onMount } from "svelte";
    import AdminPaginationBar from "$lib/components/admin/AdminPaginationBar.svelte";
    import AdminPlusIcon from "$lib/components/admin/AdminPlusIcon.svelte";
    import { adminPageSlice } from "$lib/adminPageSlice.js";
    import WordForm from "$lib/components/dictionary/WordForm.svelte";
    import { confirmDialog } from "$lib/confirm.svelte.js";
    import { ABC } from "$lib/dictionaryLetters.js";
    import { errorText, formatHuDate, sectionFetch } from "$lib/sectionApi.js";

    /** @type {any[]} */
    let words = $state([]);
    let q = $state("");
    let letter = $state("");
    let loading = $state(true);
    let error = $state("");
    let notice = $state("");
    /** @type {number | null} */
    let editingId = $state(null);
    let createOpen = $state(false);
    let page = $state(1);
    const pg = $derived(adminPageSlice(words, page));
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let searchTimer;

    async function load() {
        loading = true;
        error = "";
        try {
            const params = new URLSearchParams();
            if (q.trim()) params.set("q", q.trim());
            if (letter) params.set("letter", letter);
            const qs = params.toString();
            const data = await sectionFetch("dictionary", `words${qs ? `?${qs}` : ""}`);
            words = data?.words || [];
        } catch (err) {
            error = errorText(err, "A lista nem tölthető be.");
            words = [];
        } finally {
            loading = false;
        }
    }

    function onSearchInput() {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => {
            page = 1;
            load();
        }, 300);
    }

    onMount(load);

    $effect(() => {
        if (openId) editingId = openId;
    });

    async function remove(/** @type {{ id: number, headword: string }} */ word) {
        const ok = await confirmDialog(
            `Biztosan törlöd a(z) „${word.headword}” szót? A hozzá érkezett javaslatok is törlődnek.`,
            { yesLabel: "Törlés", destructive: true },
        );
        if (!ok) return;
        try {
            await sectionFetch("dictionary", `words/${word.id}`, { method: "DELETE" });
            notice = `Törölve: ${word.headword}.`;
            await load();
            onChanged();
        } catch (err) {
            error = errorText(err, "A törlés nem sikerült.");
        }
    }

    function saved(/** @type {{ id: number, headword: string }} */ word, /** @type {boolean} */ created) {
        notice = `${created ? "Létrehozva" : "Mentve"}: ${word.headword}.`;
        editingId = null;
        if (created) createOpen = false;
        load();
        onChanged();
    }

    function updatedAt(/** @type {string | null | undefined} */ iso) {
        const year = Number(String(iso ?? "").slice(0, 4));
        if (!year || year <= 1) return "-";
        return formatHuDate(iso) || "-";
    }
</script>

<details class="admin-create-panel" bind:open={createOpen}>
    <summary class="admin-create-summary"><span>Új szó hozzáadása</span><AdminPlusIcon /></summary>
    {#if createOpen}
        <div class="admin-create-form-wrap">
            <WordForm idPrefix="new-word" onSaved={(w) => saved(w, true)} />
        </div>
    {/if}
</details>

{#if notice}
    <div class="info-box success" role="status"><p>{notice}</p></div>
{/if}
{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

<div class="admin-table-toolbar">
    <label class="admin-search-label words-search"
        >Keresés
        <input
            id="dict-words-q"
            name="q"
            type="search"
            class="admin-search-input"
            bind:value={q}
            oninput={onSearchInput}
            placeholder="Címszó…"
        /></label
    >
    <label class="admin-search-label words-letter"
        >Betű
        <select id="dict-words-letter" name="letter" class="admin-search-input" bind:value={letter} onchange={() => ((page = 1), load())}>
            <option value="">Mind</option>
            {#each ABC as item (item)}
                <option value={item}>{item}</option>
            {/each}
        </select></label
    >
</div>
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
                <th>Címszó</th>
                <th>Szófaj</th>
                <th>Jelentések</th>
                <th>Frissítve</th>
                <th class="admin-table-col--action">Szerk.</th>
                <th class="admin-table-col--action">Törlés</th>
            </tr>
        </thead>
        <tbody>
            {#each pg.rows as word (word.id)}
                <tr>
                    <td>
                        <a href={`${szotarOrigin}/szo/${word.id}`} target="_blank" rel="noopener noreferrer"
                            >{word.headword}</a
                        >
                    </td>
                    <td>{(word.speech_types || []).join(", ") || "-"}</td>
                    <td>{word.definitions_count ?? "-"}</td>
                    <td>{updatedAt(word.updated_at)}</td>
                    <td><button type="button" class="btn-update" onclick={() => (editingId = word.id)}>Szerk.</button></td>
                    <td><button type="button" class="btn-delete" onclick={() => remove(word)}>Törlés</button></td>
                </tr>
            {:else}
                <tr><td colspan="6">{loading ? "Betöltés…" : "Nincs találat."}</td></tr>
            {/each}
        </tbody>
    </table>
</div>

{#if editingId}
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
        class="link-dialog-overlay"
        role="dialog"
        tabindex="-1"
        aria-labelledby="dict-word-edit-title"
        onclick={(e) => e.target === e.currentTarget && (editingId = null)}
        onkeydown={(e) => e.key === "Escape" && (editingId = null)}
    >
        <div class="link-dialog admin-modal">
            <h3 id="dict-word-edit-title">Szó szerkesztése</h3>
            {#key editingId}
                <WordForm
                    wordId={editingId}
                    idPrefix="edit-word"
                    onSaved={(w) => saved(w, false)}
                    onCancel={() => (editingId = null)}
                />
            {/key}
        </div>
    </div>
{/if}

<style>
    .words-search,
    .words-letter {
        width: auto;
    }
    .admin-create-form-wrap :global(.admin-form) {
        margin: 0;
        border: none;
        border-radius: 0;
    }
</style>
