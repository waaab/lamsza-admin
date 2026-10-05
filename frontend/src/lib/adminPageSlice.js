/** Rows per page for admin tables. */
export const ADMIN_PAGE_SIZE = 10;

/**
 * @param {unknown[]} items
 * @param {number} page 1-based
 * @param {number} [pageSize]
 */
export function adminPageSlice(items, page, pageSize = ADMIN_PAGE_SIZE) {
    const arr = Array.isArray(items) ? items : [];
    const total = arr.length;
    const totalPages = Math.max(1, Math.ceil(total / pageSize) || 1);
    const p = Math.min(Math.max(1, page | 0), totalPages);
    const start = (p - 1) * pageSize;
    const slice = arr.slice(start, start + pageSize);
    return {
        rows: slice,
        total,
        totalPages,
        page: p,
        from: total === 0 ? 0 : start + 1,
        to: Math.min(start + pageSize, total),
    };
}
