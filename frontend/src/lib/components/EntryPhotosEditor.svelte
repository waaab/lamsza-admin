<script>
    import { getApiBase, apiCall } from "$lib/api.js";
    import { absoluteMediaUrl } from "$lib/eventImage.js";
    import { MAX_ENTRY_PHOTOS, normalizePhotos } from "$lib/entryPhotos.js";

    /**
     * @type {{
     *   photos?: Array<Record<string, unknown>>,
     *   entryId?: number | string | null,
     *   alertError?: (message: string) => void,
     * }}
     */
    let { photos = $bindable([]), entryId = null, alertError = () => {} } = $props();

    let list = $derived(normalizePhotos(photos));
    let remaining = $derived(Math.max(0, MAX_ENTRY_PHOTOS - list.length));
    let uploading = $state(false);
    let uploadId = $derived(`entry-photo-upload-${entryId ?? "new"}`);

    async function onFiles(event) {
        const input = event.currentTarget;
        const files = [...(input.files || [])];
        input.value = "";
        if (!files.length || remaining <= 0) return;
        uploading = true;
        try {
            for (const file of files.slice(0, remaining)) {
                const fd = new FormData();
                fd.append("file", file);
                if (entryId) fd.append("entry_id", String(entryId));
                const res = await apiCall("/api/admin/entry-images", {
                    method: "POST",
                    body: fd,
                });
                if (!res.ok) {
                    alertError((await res.text()) || "Feltöltés sikertelen.");
                    break;
                }
                const data = await res.json();
                const url = String(data.url || "").trim();
                if (!url) {
                    alertError("A feltöltés nem adott vissza kép URL-t.");
                    break;
                }
                photos = [
                    ...normalizePhotos(photos),
                    {
                        url,
                        alt: "",
                        title: "",
                        description: "",
                        copyright: "",
                        width: Number(data.width) || 1600,
                        height: Number(data.height) || 1200,
                    },
                ];
            }
        } catch (err) {
            alertError("Feltöltés hiba: " + (err?.message || err));
        } finally {
            uploading = false;
        }
    }

    function patch(i, field, value) {
        photos = normalizePhotos(photos).map((photo, idx) =>
            idx === i ? { ...photo, [field]: value } : photo,
        );
    }

    function removeAt(i) {
        photos = normalizePhotos(photos).filter((_, idx) => idx !== i);
    }

    function move(i, dir) {
        const next = [...normalizePhotos(photos)];
        const j = i + dir;
        if (j < 0 || j >= next.length) return;
        const tmp = next[i];
        next[i] = next[j];
        next[j] = tmp;
        photos = next;
    }
</script>

<div class="entry-photos-editor">
    <label class="entry-photos-editor__upload" for={uploadId}>
        Fotók feltöltése
        <input
            id={uploadId}
            type="file"
            accept="image/jpeg,image/png,image/webp,image/gif"
            multiple
            disabled={uploading || remaining <= 0}
            onchange={onFiles}
        />
    </label>
    <p class="admin-form-hint">
        JPEG, PNG, WebP vagy GIF, max. {MAX_ENTRY_PHOTOS} kép. Az alt a képernyőolvasóknak
        kell; a cím és a leírás a nyilvános diavetítésen látszik. Szélesség és magasság a
        feltöltéskor mentődik.
    </p>

    {#if !list.length}
        <p class="admin-form-hint">Még nincs feltöltött fotó.</p>
    {:else}
        <ul class="entry-photos-editor__list">
            {#each list as photo, i (photo.url + i)}
                <li class="entry-photos-editor__item">
                    <img
                        class="entry-photos-editor__preview"
                        src={absoluteMediaUrl(photo.url, getApiBase())}
                        alt={photo.alt || ""}
                        title={photo.title || photo.alt || ""}
                        width={photo.width}
                        height={photo.height}
                        loading="lazy"
                        decoding="async"
                    />
                    <div class="entry-photos-editor__fields">
                        <label>
                            Alt
                            <input
                                type="text"
                                name={`photo-alt-${i}`}
                                value={photo.alt}
                                oninput={(event) =>
                                    patch(i, "alt", event.currentTarget.value)}
                            />
                        </label>
                        <label>
                            Cím
                            <input
                                type="text"
                                name={`photo-title-${i}`}
                                value={photo.title}
                                oninput={(event) =>
                                    patch(i, "title", event.currentTarget.value)}
                            />
                        </label>
                        <label>
                            Leírás
                            <textarea
                                name={`photo-description-${i}`}
                                rows="2"
                                value={photo.description}
                                oninput={(event) =>
                                    patch(i, "description", event.currentTarget.value)}
                            ></textarea>
                        </label>
                        <label>
                            Szerzői jog
                            <input
                                type="text"
                                name={`photo-copyright-${i}`}
                                value={photo.copyright}
                                oninput={(event) =>
                                    patch(i, "copyright", event.currentTarget.value)}
                            />
                        </label>
                        <p class="entry-photos-editor__meta">
                            {photo.width}×{photo.height}
                        </p>
                        <div class="entry-photos-editor__actions">
                            <button
                                type="button"
                                class="btn-update"
                                disabled={i === 0}
                                onclick={() => move(i, -1)}>Fel</button
                            >
                            <button
                                type="button"
                                class="btn-update"
                                disabled={i === list.length - 1}
                                onclick={() => move(i, 1)}>Le</button
                            >
                            <button
                                type="button"
                                class="btn-delete"
                                onclick={() => removeAt(i)}>Törlés</button
                            >
                        </div>
                    </div>
                </li>
            {/each}
        </ul>
    {/if}
</div>

<style>
    .entry-photos-editor {
        display: flex;
        flex-direction: column;
        gap: 0.45rem;
        margin-bottom: 0.75rem;
    }
    .entry-photos-editor__upload {
        display: flex;
        flex-direction: column;
        gap: 0.3rem;
        font-weight: 600;
    }
    .entry-photos-editor__list {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
    }
    .entry-photos-editor__item {
        display: grid;
        grid-template-columns: 7.5rem minmax(0, 1fr);
        gap: 0.75rem;
        padding: 0.65rem;
        border: 1px solid var(--border-color);
        border-radius: 8px;
        background: var(--card-bg, #fff);
    }
    .entry-photos-editor__preview {
        width: 7.5rem;
        height: auto;
        max-height: 6rem;
        object-fit: cover;
        border-radius: 6px;
        border: 1px solid var(--border-color);
        background: var(--bg-body, #f4f4f4);
    }
    .entry-photos-editor__fields {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
        min-width: 0;
    }
    .entry-photos-editor__fields label {
        display: flex;
        flex-direction: column;
        gap: 0.15rem;
        font-size: var(--text-sm);
        font-weight: 600;
    }
    .entry-photos-editor__meta {
        margin: 0;
        color: var(--text-muted);
        font-size: var(--text-sm);
    }
    .entry-photos-editor__actions {
        display: flex;
        flex-wrap: wrap;
        gap: 0.35rem;
    }
    @media (max-width: 640px) {
        .entry-photos-editor__item {
            grid-template-columns: 1fr;
        }
        .entry-photos-editor__preview {
            width: 100%;
            max-height: 10rem;
        }
    }
</style>
