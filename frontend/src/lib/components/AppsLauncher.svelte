<script>
    /**
     * The apps launcher (UI_BASELINE "tb-apps-launcher"): the last button on
     * the right of every public app's toolbar, opening a small panel with the
     * network's apps. Shared from lamsza by scripts/sync-shared-frontend.sh;
     * the list lives in $lib/networkApps.js. Not used in the admin app.
     *
     *   <AppsLauncher current="szotar" hostname={$page.url.hostname} />
     *
     * Keyboard: Enter or Space opens it and focuses the first app; Escape
     * closes it and returns focus to the button; tabbing out or a click
     * outside closes it. Outside clicks are caught on pointerdown, so a
     * toolbar dropdown that stops click propagation still closes it.
     */
    import { tick } from "svelte";
    import AppIcon from "$lib/icons/AppIcon.svelte";
    import { launcherApps } from "$lib/networkApps.js";

    /** @type {{ current: string, hostname?: string }} */
    let { current, hostname = "" } = $props();

    const apps = $derived(launcherApps(current, hostname));
    const panelId = "apps-launcher-panel";

    let open = $state(false);
    /** @type {HTMLDivElement | undefined} */
    let root = $state();
    /** @type {HTMLButtonElement | undefined} */
    let button = $state();

    /** @param {MouseEvent} event */
    async function toggle(event) {
        open = !open;
        // detail 0: opened with Enter or Space, not a pointer.
        if (open && event.detail === 0) {
            await tick();
            root?.querySelector(".apps-launcher-item")?.focus();
        }
    }

    function close() {
        open = false;
    }

    /** @param {KeyboardEvent} event */
    function onKeydown(event) {
        if (open && event.key === "Escape") {
            event.stopPropagation();
            close();
            button?.focus();
        }
    }

    /** @param {FocusEvent} event */
    function onFocusOut(event) {
        const next = /** @type {Node | null} */ (event.relatedTarget);
        if (open && next && !root?.contains(next)) close();
    }

    /** @param {PointerEvent} event */
    function onWindowPointerDown(event) {
        if (open && !root?.contains(/** @type {Node} */ (event.target))) close();
    }
</script>

<svelte:window onpointerdown={onWindowPointerDown} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="apps-launcher" bind:this={root} onkeydown={onKeydown} onfocusout={onFocusOut}>
    <button
        bind:this={button}
        type="button"
        class="btn nav-btn"
        class:active={open}
        title="Alkalmazások"
        aria-expanded={open}
        aria-controls={panelId}
        onclick={toggle}
    >
        <AppIcon name="apps" size={16} />
        <span class="sr-only">Alkalmazások</span>
    </button>
    {#if open}
        <nav id={panelId} class="apps-launcher-panel" aria-label="Lámsza alkalmazások">
            <div class="dropdown-header">Lámsza alkalmazások</div>
            <ul class="apps-launcher-list">
                {#each apps as app (app.id)}
                    <li>
                        <a
                            href={app.href}
                            class="apps-launcher-item"
                            class:current={app.current}
                            aria-current={app.current ? "true" : undefined}
                            onclick={close}
                        >
                            <AppIcon name={app.icon} size={24} />
                            <span>{app.name}</span>
                        </a>
                    </li>
                {/each}
            </ul>
        </nav>
    {/if}
</div>
