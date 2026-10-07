<script>
    /**
     * The intro text on each of Szótár's speech-type pages (moved from
     * Szótár's /admin/szofajok). A card saves only when its text changed; the
     * backend refuses more than 2000 characters.
     */
    import { onMount } from "svelte";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    /** @type {string[]} */
    let types = $state([]);
    /** @type {Record<string, string>} */
    let drafts = $state({});
    /** @type {Record<string, string>} */
    let saved = $state({});
    /** @type {Record<string, string>} */
    let status = $state({});
    let loaded = $state(false);
    let error = $state("");
    /** @type {string | null} */
    let saving = $state(null);

    onMount(async () => {
        try {
            const [list, data] = await Promise.all([
                sectionFetch("dictionary", "speech-types"),
                sectionFetch("dictionary", "speech-type-intros"),
            ]);
            types = list?.speech_types || [];
            /** @type {Record<string, string>} */
            const next = Object.fromEntries(types.map((name) => [name, ""]));
            for (const item of data?.intros || []) next[item.speech_type] = item.intro || "";
            drafts = { ...next };
            saved = { ...next };
        } catch (err) {
            error = errorText(err, "A bemutatkozó szövegek nem töltődtek be.");
        } finally {
            loaded = true;
        }
    });

    async function save(/** @type {string} */ name) {
        saving = name;
        status = { ...status, [name]: "" };
        try {
            const item = await sectionFetch("dictionary", "speech-type-intros", {
                method: "PUT",
                body: { speech_type: name, intro: drafts[name] || "" },
            });
            saved = { ...saved, [name]: item?.intro || "" };
            drafts = { ...drafts, [name]: item?.intro || "" };
            status = { ...status, [name]: "Mentve." };
        } catch (err) {
            status = { ...status, [name]: errorText(err, "A mentés nem sikerült.") };
        } finally {
            saving = null;
        }
    }
</script>

<p class="admin-info">
    Minden szófaj oldalán 2-4 mondatos bevezető jelenik meg. Üresen hagyva nincs bevezető bekezdés.
</p>

{#if error}
    <div class="info-box error" role="alert"><p>{error}</p></div>
{/if}

{#if !loaded}
    <p>Betöltés…</p>
{:else}
    <div class="intro-grid">
        {#each types as name (name)}
            <form
                class="admin-form intro-card"
                onsubmit={(e) => {
                    e.preventDefault();
                    save(name);
                }}
            >
                <label for="intro-{name}">{name}</label>
                <textarea
                    id="intro-{name}"
                    name="intro"
                    bind:value={drafts[name]}
                    rows="4"
                    maxlength="2000"
                    placeholder="2-4 mondatos bemutatkozó szöveg…"
                ></textarea>
                <div class="admin-modal-actions intro-actions">
                    <button
                        type="submit"
                        class="btn-update"
                        disabled={saving === name || (drafts[name] || "") === (saved[name] || "")}
                        >{saving === name ? "Mentés…" : "Mentés"}</button
                    >
                    {#if status[name]}
                        <span class="admin-status" role="status">{status[name]}</span>
                    {/if}
                </div>
            </form>
        {/each}
    </div>
{/if}

<style>
    .admin-info {
        color: var(--text-faint, #666);
        margin-bottom: 1rem;
    }
    .intro-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr));
        gap: 1rem;
    }
    .intro-card {
        margin: 0;
        padding: 1rem;
        gap: 0.75rem;
    }
    .intro-card label {
        font-weight: 600;
        margin: 0;
    }
    .intro-actions {
        align-items: center;
    }
    @media (max-width: 480px) {
        .intro-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
