<script>
    import { WEEKDAYS, emptyWeekHours, normalizeHours } from "$lib/entryHours.js";
    import HuTimeInput from "$lib/components/HuTimeInput.svelte";

    /** @type {{ hours?: Record<string, { open: string, close: string, closed: boolean }> }} */
    let { hours = $bindable(emptyWeekHours()) } = $props();

    let rows = $derived(
        WEEKDAYS.map(({ key, label }) => {
            const week = normalizeHours(hours);
            return { key, label, slot: week[key] };
        }),
    );

    function patchDay(key, patch) {
        const week = normalizeHours(hours);
        hours = { ...week, [key]: { ...week[key], ...patch } };
    }
</script>

<ul class="entry-hours-editor">
    {#each rows as row (row.key)}
        <li>
            <div class="entry-hours-editor__head">
                <span class="entry-hours-editor__day">{row.label}</span>
                <label class="entry-hours-editor__closed">
                    <input
                        type="checkbox"
                        name={`hours-${row.key}-closed`}
                        checked={row.slot.closed}
                        onchange={(event) =>
                            patchDay(row.key, { closed: event.currentTarget.checked })}
                    />
                    Zárva
                </label>
            </div>
            <div class="entry-hours-editor__times">
                <HuTimeInput
                    name={`hours-${row.key}-open`}
                    ariaLabel="{row.label} nyitás"
                    placeholder="08:00"
                    value={row.slot.open}
                    disabled={row.slot.closed}
                    onchange={(next) => patchDay(row.key, { open: next })}
                />
                <HuTimeInput
                    name={`hours-${row.key}-close`}
                    ariaLabel="{row.label} zárás"
                    placeholder="20:00"
                    value={row.slot.close}
                    disabled={row.slot.closed}
                    onchange={(next) => patchDay(row.key, { close: next })}
                />
            </div>
        </li>
    {/each}
</ul>

<style>
    .entry-hours-editor {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        min-width: 0;
    }
    .entry-hours-editor li {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
        min-width: 0;
    }
    .entry-hours-editor__head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.5rem;
    }
    .entry-hours-editor__day {
        font-weight: 500;
    }
    .entry-hours-editor__times {
        display: grid;
        grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
        gap: 0.4rem;
        min-width: 0;
    }
    .entry-hours-editor__times :global(input) {
        width: 100%;
        min-width: 0;
        box-sizing: border-box;
        margin: 0;
    }
    .entry-hours-editor__closed {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        margin: 0;
        font-weight: 400;
        white-space: nowrap;
    }
    .entry-hours-editor__closed input {
        width: auto;
        margin: 0;
    }
</style>
