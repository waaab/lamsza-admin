<script>
    import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
    import { answerConfirm, confirmState } from "$lib/confirm.svelte.js";
</script>

<ConfirmDialog
    open={confirmState.open}
    message={confirmState.message}
    yesLabel={confirmState.yesLabel}
    destructive={confirmState.destructive}
    onYes={() => answerConfirm(true)}
    onNo={() => answerConfirm(false)}
/>
