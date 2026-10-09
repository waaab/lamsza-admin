<script>
    import AppIcon from "$lib/icons/AppIcon.svelte";
    /**
     * Szótár word editor, live mode (moved from Szótár's WordForm; Szótár keeps
     * its suggest-mode form, and both go through the same backend Validate).
     * A save replaces every sense of the word.
     *
     * @type {{
     *   wordId?: number | null,
     *   idPrefix?: string,
     *   onSaved?: (word: { id: number, headword: string }) => void,
     *   onCancel?: (() => void) | null,
     * }}
     */
    let { wordId = null, idPrefix = "word", onSaved = () => {}, onCancel = null } = $props();

    import { onMount } from "svelte";
    import { errorText, sectionFetch } from "$lib/sectionApi.js";

    /** @type {string[]} */
    let speechTypes = $state([]);
    let error = $state("");
    let saving = $state(false);
    let ready = $state(false);

    let headword = $state("");
    let pronunciation = $state("");
    let audioUrl = $state("");
    let videoUrl = $state("");
    let imgUrl = $state("");
    /** @type {any[]} */
    let definitions = $state([blankDef()]);

    function blankDef() {
        return {
            definition_hu: "",
            description_hu: "",
            example_sentences: "",
            example_proverbs: "",
            example_source: "",
            history: "",
            origin: "",
            conjugation: "",
            speech_types: [],
            locations: "",
            tags: "",
            synonyms: "",
            antonyms: "",
            variations: "",
        };
    }

    /** Comma-separated list, trimmed, without case-insensitive duplicates. */
    function labels(/** @type {string} */ value) {
        return value
            .split(",")
            .map((part) => part.trim())
            .filter(
                (part, index, all) =>
                    part && all.findIndex((item) => item.toLowerCase() === part.toLowerCase()) === index,
            );
    }

    onMount(async () => {
        try {
            const types = await sectionFetch("dictionary", "speech-types");
            speechTypes = types?.speech_types || [];
            if (wordId) {
                const word = await sectionFetch("dictionary", `words/${wordId}`);
                headword = word.headword || "";
                pronunciation = word.pronunciation || "";
                audioUrl = word.audio_url || "";
                videoUrl = word.video_url || "";
                imgUrl = word.img_url || "";
                definitions = (word.definitions?.length ? word.definitions : [blankDef()]).map((/** @type {any} */ def) => ({
                    definition_hu: def.definition_hu || "",
                    description_hu: def.description_hu || "",
                    example_sentences: def.example_sentences || "",
                    example_proverbs: def.example_proverbs || "",
                    example_source: def.example_source || "",
                    history: def.history || "",
                    origin: def.origin || "",
                    conjugation: def.conjugation || "",
                    speech_types: [...(def.speech_types || [])],
                    locations: (def.locations || []).join(", "),
                    tags: (def.tags || []).join(", "),
                    synonyms: (def.synonyms || []).join(", "),
                    antonyms: (def.antonyms || []).join(", "),
                    variations: (def.variations || []).join(", "),
                }));
            }
            ready = true;
        } catch (err) {
            error = errorText(err, "Az űrlap nem töltődött be.");
        }
    });

    function toggleSpeech(/** @type {any} */ def, /** @type {string} */ name) {
        def.speech_types = def.speech_types.includes(name)
            ? def.speech_types.filter((/** @type {string} */ item) => item !== name)
            : [...def.speech_types, name];
    }

    function removeDefinition(/** @type {number} */ index) {
        definitions = definitions.filter((_, i) => i !== index);
        if (definitions.length === 0) definitions = [blankDef()];
    }

    async function save(/** @type {SubmitEvent} */ event) {
        event.preventDefault();
        saving = true;
        error = "";
        const body = {
            headword,
            pronunciation,
            audio_url: audioUrl,
            video_url: videoUrl,
            img_url: imgUrl,
            definitions: definitions.map((def) => ({
                definition_hu: def.definition_hu,
                description_hu: def.description_hu,
                example_sentences: def.example_sentences,
                example_proverbs: def.example_proverbs,
                example_source: def.example_source,
                history: def.history,
                origin: def.origin,
                conjugation: def.conjugation,
                speech_types: def.speech_types,
                locations: labels(def.locations),
                tags: labels(def.tags),
                synonyms: labels(def.synonyms),
                antonyms: labels(def.antonyms),
                variations: labels(def.variations),
            })),
        };
        try {
            const saved = wordId
                ? await sectionFetch("dictionary", `words/${wordId}`, { method: "PUT", body })
                : await sectionFetch("dictionary", "words", { method: "POST", body });
            if (!wordId) {
                headword = pronunciation = audioUrl = videoUrl = imgUrl = "";
                definitions = [blankDef()];
            }
            onSaved(saved);
        } catch (err) {
            error = errorText(err, "A mentés nem sikerült.");
        } finally {
            saving = false;
        }
    }

    const id = (/** @type {string} */ name) => `${idPrefix}-${name}`;
</script>

{#if error && !ready}
    <div class="info-box error" role="alert"><p>{error}</p></div>
    {#if onCancel}
        <div class="admin-modal-actions">
            <button type="button" class="btn-delete" onclick={onCancel}>Bezárás</button>
        </div>
    {/if}
{:else if ready}
    <form class="admin-form word-form" onsubmit={save}>
        <label for={id("headword")}>Címszó</label>
        <input id={id("headword")} name="headword" bind:value={headword} required />
        <label for={id("pronunciation")}>Kiejtés</label>
        <input id={id("pronunciation")} name="pronunciation" bind:value={pronunciation} />
        <label for={id("audio")}>Hang (URL)</label>
        <input id={id("audio")} name="audio_url" bind:value={audioUrl} />
        <label for={id("video")}>Videó (URL)</label>
        <input id={id("video")} name="video_url" bind:value={videoUrl} />
        <label for={id("image")}>Kép (URL)</label>
        <input id={id("image")} name="img_url" bind:value={imgUrl} />

        {#each definitions as def, index (index)}
            {@const d = (/** @type {string} */ name) => id(`def-${index}-${name}`)}
            <fieldset class="word-sense">
                <legend>Jelentés {index + 1}</legend>
                <label for={d("hu")}>Magyar jelentés</label>
                <textarea id={d("hu")} name="definition_hu_{index}" rows="2" bind:value={def.definition_hu}></textarea>
                <fieldset class="admin-choices">
                    <legend>Szófaj</legend>
                    {#each speechTypes as name (name)}
                        <label for={d(`speech-${name}`)}>
                            <input
                                id={d(`speech-${name}`)}
                                name="speech_types_{index}"
                                type="checkbox"
                                checked={def.speech_types.includes(name)}
                                onchange={() => toggleSpeech(def, name)}
                            />
                            {name}
                        </label>
                    {/each}
                </fieldset>
                <label for={d("description")}>Leírás</label>
                <textarea id={d("description")} name="description_hu_{index}" rows="2" bind:value={def.description_hu}></textarea>
                <label for={d("examples")}>Példamondatok</label>
                <textarea id={d("examples")} name="example_sentences_{index}" rows="2" bind:value={def.example_sentences}></textarea>
                <label for={d("proverbs")}>Székely mondás ezzel a szóval</label>
                <textarea id={d("proverbs")} name="example_proverbs_{index}" rows="2" bind:value={def.example_proverbs}></textarea>
                <label for={d("source")}>Forrás</label>
                <input id={d("source")} name="example_source_{index}" bind:value={def.example_source} />
                <label for={d("origin")}>Eredet</label>
                <input id={d("origin")} name="origin_{index}" bind:value={def.origin} />
                <label for={d("history")}>Történet</label>
                <textarea id={d("history")} name="history_{index}" rows="2" bind:value={def.history}></textarea>
                <label for={d("conjugation")}>Ragozás</label>
                <input id={d("conjugation")} name="conjugation_{index}" bind:value={def.conjugation} />
                <label for={d("locations")}>Helyek, vesszővel</label>
                <input id={d("locations")} name="locations_{index}" bind:value={def.locations} />
                <label for={d("tags")}>Címkék, vesszővel</label>
                <input id={d("tags")} name="tags_{index}" bind:value={def.tags} />
                <label for={d("synonyms")}>Szinonimák, vesszővel</label>
                <input id={d("synonyms")} name="synonyms_{index}" bind:value={def.synonyms} />
                <label for={d("antonyms")}>Ellentétek, vesszővel</label>
                <input id={d("antonyms")} name="antonyms_{index}" bind:value={def.antonyms} />
                <label for={d("variations")}>Változatok, vesszővel</label>
                <input id={d("variations")} name="variations_{index}" bind:value={def.variations} />
                {#if definitions.length > 1}
                    <div>
                        <button type="button" class="btn btn-sm" onclick={() => removeDefinition(index)}
                            >Jelentés törlése</button
                        >
                    </div>
                {/if}
            </fieldset>
        {/each}

        <div>
            <button type="button" class="btn btn-sm" onclick={() => (definitions = [...definitions, blankDef()])}
                ><AppIcon name="plus" size={14} />Új jelentés</button
            >
        </div>
        {#if error}
            <div class="info-box error" role="alert"><p>{error}</p></div>
        {/if}
        <div class="admin-modal-actions">
            <button type="submit" class="admin-submit-btn" disabled={saving}>{saving ? "Mentés…" : "Mentés"}</button>
            {#if onCancel}
                <button type="button" class="btn-delete" onclick={onCancel}>Mégse</button>
            {/if}
        </div>
    </form>
{:else}
    <p class="admin-status">Betöltés…</p>
{/if}

<style>
    .word-sense {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        min-width: 0;
        margin: 0;
        padding: 1rem;
        border: 1px solid var(--border-color);
        border-radius: 6px;
    }
    .word-sense > legend {
        padding: 0 0.35rem;
        font-weight: 600;
    }
    .word-sense > label {
        font-weight: 500;
        margin-bottom: -0.5rem;
    }
</style>
