/**
 * One confirm question at a time for the /dictionary and /games sections: a
 * tab asks with `await confirmDialog("…")`, the page renders <ConfirmHost />
 * once (UI_BASELINE "dlg-confirm": Mégse next to the action).
 */
export const confirmState = $state({ open: false, message: "", yesLabel: "Igen", destructive: false });

/** @type {((yes: boolean) => void) | null} */
let resolver = null;

/**
 * @param {string} message
 * @param {{ yesLabel?: string, destructive?: boolean }} [opts]
 * @returns {Promise<boolean>}
 */
export function confirmDialog(message, opts = {}) {
    resolver?.(false);
    confirmState.message = message;
    confirmState.yesLabel = opts.yesLabel || "Igen";
    confirmState.destructive = !!opts.destructive;
    confirmState.open = true;
    return new Promise((resolve) => (resolver = resolve));
}

/** @param {boolean} yes */
export function answerConfirm(yes) {
    confirmState.open = false;
    const r = resolver;
    resolver = null;
    r?.(yes);
}
