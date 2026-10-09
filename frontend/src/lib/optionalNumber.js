/**
 * A number typed into an admin form, or null when the field is empty: an
 * optional fact such as an attraction's elevation, area or depth (lamsza
 * UI_BASELINE "szf-pages"). A decimal comma counts as a point ("0,22"), and
 * so do spaces between thousands ("1 234"). Anything else that is not a
 * number is null too, and the form says so before saving.
 *
 * @param {unknown} text
 * @returns {number | null}
 */
export function optionalNumber(text) {
    const s = String(text ?? "").trim().replace(/\s+/g, "").replace(",", ".");
    if (s === "") return null;
    if (!/^-?\d+(\.\d+)?$/.test(s)) return null;
    return Number(s);
}

/**
 * True when the field is empty or holds a number `optionalNumber` reads.
 *
 * @param {unknown} text
 */
export function isOptionalNumber(text) {
    return String(text ?? "").trim() === "" || optionalNumber(text) !== null;
}
