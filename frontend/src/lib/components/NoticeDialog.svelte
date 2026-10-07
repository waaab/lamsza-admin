<script>
    /** @type {{ title?: string, message?: string, closeLabel?: string, onClose?: () => void }} */
    let {
        title = "Javaslat elküldve",
        message = "A kérésedet elküldtük az adminnak. A módosítások az admin ellenőrzése és jóváhagyása után jelennek meg az oldalon.",
        closeLabel = "Bezárás",
        onClose = () => {},
    } = $props();
</script>

<div
    class="link-dialog-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="notice-dialog-title"
    tabindex="-1"
    onclick={(e) => e.target === e.currentTarget && onClose()}
    onkeydown={(e) => e.key === "Escape" && onClose()}
>
    <div class="link-dialog notice-dialog" role="presentation" onclick={(e) => e.stopPropagation()}>
        <h3 id="notice-dialog-title">{title}</h3>
        <p class="create-form-note">{message}</p>
        <div class="link-dialog-actions">
            <button type="button" class="link-dialog-submit" onclick={onClose}>{closeLabel}</button>
        </div>
    </div>
</div>

<style>
    .notice-dialog {
        width: min(28rem, calc(100vw - 2rem));
        max-width: min(28rem, calc(100vw - 2rem));
        min-width: 0;
    }
</style>
