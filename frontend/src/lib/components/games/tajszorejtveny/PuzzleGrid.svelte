<script>
    /**
     * A Tájszórejtvény grid for the review screen, with its answers: letter
     * cells show the solution (given letters shaded, the proverb's cells
     * numbered), clue cells their clues with the arrow, the rest the plain
     * decorative cell. Clicking a clue or a letter selects a word.
     *
     * @type {{ puzzle: any, selected?: number, onselect?: (word: number) => void }}
     */
    let { puzzle, selected = -1, onselect = () => {} } = $props();

    /** @type {Record<string, string>} */
    const ARROWS = { r: "→", d: "↓", dr: "↳", rd: "↴" };

    const given = $derived(new Set((puzzle.given ?? []).map((/** @type {number[]} */ c) => `${c[0]},${c[1]}`)));
    const solution = $derived(
        new Map((puzzle.solution?.cells ?? []).map((/** @type {number[]} */ c, /** @type {number} */ i) => [`${c[0]},${c[1]}`, i + 1])),
    );
    const inWord = $derived(
        new Set(
            selected >= 0 ? (puzzle.words[selected]?.cells ?? []).map((/** @type {number[]} */ c) => `${c[0]},${c[1]}`) : [],
        ),
    );

    /** A letter cell selects its across word, else its down word; a second click on a crossing switches. */
    function pickLetter(/** @type {any} */ cell) {
        const a = cell.a ?? -1;
        const d = cell.d ?? -1;
        if (a >= 0 && d >= 0) onselect(selected === a ? d : a);
        else onselect(a >= 0 ? a : d);
    }
</script>

<div class="taj-grid-wrap">
    <div class="taj-grid" style:grid-template-columns="repeat({puzzle.w}, var(--taj-cell))" role="grid" aria-label="A rejtvény rácsa megoldással">
        {#each puzzle.grid as row, r (r)}
            {#each row as cell, c (c)}
                {@const k = `${r},${c}`}
                {#if cell.k === "L"}
                    <button
                        type="button"
                        class="cell letter"
                        class:given={given.has(k)}
                        class:in-word={inWord.has(k)}
                        aria-label="{r + 1}. sor, {c + 1}. oszlop: {cell.ch}"
                        onclick={() => pickLetter(cell)}
                    >
                        {#if solution.has(k)}<span class="sol">{solution.get(k)}</span>{/if}
                        <span class="ch">{cell.ch}</span>
                    </button>
                {:else if cell.k === "C"}
                    <div class="cell clue">
                        {#each cell.clues ?? [] as cl (cl.word)}
                            <button type="button" class="clue-line" class:current={cl.word === selected} onclick={() => onselect(cl.word)}>
                                <span class="clue-text">{puzzle.words[cl.word]?.clue}</span>
                                <span class="arrow" aria-hidden="true">{ARROWS[cl.arrow] ?? ""}</span>
                            </button>
                        {/each}
                    </div>
                {:else}
                    <div class="cell decor" aria-hidden="true"></div>
                {/if}
            {/each}
        {/each}
    </div>
</div>

<style>
    .taj-grid-wrap {
        overflow-x: auto;
        margin: 0.5rem 0 1rem;
    }

    .taj-grid {
        --taj-cell: 3.4rem;
        display: grid;
        grid-auto-rows: var(--taj-cell);
        gap: 1px;
        width: max-content;
        border: 1px solid var(--border-color);
        background: var(--border-color);
    }

    .cell {
        position: relative;
        box-sizing: border-box;
        width: 100%;
        height: 100%;
        margin: 0;
        padding: 0;
        border: none;
        border-radius: 0;
        background: var(--card-bg);
        color: var(--text-primary);
        font: inherit;
        overflow: hidden;
    }

    .letter {
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
    }

    .letter .ch {
        font-size: 1.25rem;
        font-weight: 700;
        text-transform: uppercase;
    }

    .letter.given {
        background: color-mix(in srgb, var(--szekely-brown) 14%, var(--card-bg));
    }

    .letter.in-word {
        outline: 2px solid var(--szekely-green);
        outline-offset: -2px;
    }

    .sol {
        position: absolute;
        top: 1px;
        right: 3px;
        font-size: 0.65rem;
        font-weight: 700;
        color: var(--szekely-red);
    }

    .clue {
        display: flex;
        flex-direction: column;
        background: var(--entry-category-bg, var(--card-bg));
    }

    .clue-line {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 2px;
        min-height: 0;
        padding: 1px 2px;
        border: none;
        background: transparent;
        color: inherit;
        font: inherit;
        font-size: 0.56rem;
        font-weight: 600;
        line-height: 1.1;
        text-align: center;
        cursor: pointer;
        overflow: hidden;
    }

    .clue-line + .clue-line {
        border-top: 1px solid var(--border-color);
    }

    .clue-line.current {
        background: color-mix(in srgb, var(--szekely-green) 18%, transparent);
    }

    .clue-text {
        overflow: hidden;
        display: -webkit-box;
        -webkit-line-clamp: 4;
        line-clamp: 4;
        -webkit-box-orient: vertical;
        hyphens: auto;
        overflow-wrap: break-word;
    }

    .arrow {
        flex: none;
        color: var(--text-muted);
    }

    .decor {
        background: color-mix(in srgb, var(--text-muted) 12%, var(--card-bg));
    }
</style>
