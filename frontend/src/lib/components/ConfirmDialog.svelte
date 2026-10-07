<script>
    /**
     * Confirm dialog, network baseline (UI_BASELINE.md, "dlg-confirm"): the
     * Szótár / Játszótér AppDialog look - a native <dialog> with a blurred
     * backdrop, a card panel, and Mégse before the confirm button, aligned
     * right. The props stay as they were, so call sites do not change.
     *
     * @type {{
     *   open?: boolean,
     *   title?: string,
     *   message?: string,
     *   yesLabel?: string,
     *   noLabel?: string,
     *   destructive?: boolean,
     *   onYes?: () => void,
     *   onNo?: () => void,
     * }}
     */
    let {
        open = false,
        title = "Biztos vagy benne?",
        message = "",
        yesLabel = "Igen",
        noLabel = "Mégse",
        destructive = false,
        onYes = () => {},
        onNo = () => {},
    } = $props();

    let dialogEl = $state(null);

    $effect(() => {
        const el = dialogEl;
        if (!el) return;
        if (open && !el.open) el.showModal();
        else if (!open && el.open) el.close();
    });

    /** @param {MouseEvent} event */
    function onBackdropClick(event) {
        if (event.target === dialogEl) onNo();
    }

    /** @param {Event} event */
    function onCancel(event) {
        event.preventDefault();
        onNo();
    }
</script>

{#if open}
    <dialog
        bind:this={dialogEl}
        class="confirm-dialog"
        aria-labelledby="confirm-dialog-title"
        aria-describedby="confirm-dialog-message"
        onclick={onBackdropClick}
        oncancel={onCancel}
    >
        <div class="confirm-panel card">
            <h2 id="confirm-dialog-title" class="confirm-title">{title}</h2>
            <p id="confirm-dialog-message" class="confirm-message">{message}</p>
            <div class="confirm-actions">
                <button type="button" class="btn" onclick={onNo}>{noLabel}</button>
                <button type="button" class="btn" class:btn-danger={destructive} onclick={onYes}>{yesLabel}</button>
            </div>
        </div>
    </dialog>
{/if}

<style>
    .confirm-dialog {
        border: none;
        padding: 0;
        max-width: min(28rem, calc(100vw - 2rem));
        background: transparent;
        color: var(--text-primary);
    }

    .confirm-dialog::backdrop {
        background: rgba(0, 0, 0, 0.55);
        backdrop-filter: blur(2px);
    }

    .confirm-panel {
        padding: 1.5rem 1.35rem 1.25rem;
        margin: 0;
    }

    .confirm-title {
        margin: 0 0 0.75rem;
        font-size: var(--text-md);
        font-weight: 700;
        line-height: 1.3;
    }

    .confirm-message {
        margin: 0 0 1.25rem;
        color: var(--text-secondary);
        line-height: 1.5;
        font-size: var(--text-sm);
        white-space: pre-line;
    }

    .confirm-actions {
        display: flex;
        justify-content: flex-end;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .btn-danger {
        background: var(--btn-danger-bg);
        border-color: var(--btn-danger-bg);
        color: var(--white);
    }

    .btn-danger:hover {
        background: var(--btn-danger-hover);
        border-color: var(--btn-danger-hover);
    }
</style>
