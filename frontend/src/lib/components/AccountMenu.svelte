<script>
    /**
     * The account button at the right of every public app's toolbar, before
     * the apps launcher (UI_BASELINE "tb-account-menu"). Signed out: Belépés.
     * Signed in: the user's photo (or the profile icon), opening a menu with
     * Fiók, Beállítások and Kijelentkezés. Shared from lamsza by
     * scripts/sync-shared-frontend.sh.
     *
     * Keyboard and pointer behave as in AppsLauncher: Enter or Space opens it
     * and focuses the first item, Escape closes it and returns focus to the
     * button, tabbing out or a pointerdown outside closes it.
     *
     *   <AccountMenu account={…or null} onLogin={openLogin} onLogout={logout}
     *       active={pathname === "/fiok" || pathname === "/beallitasok"} />
     */
    import { tick } from "svelte";
    import AppIcon from "$lib/icons/AppIcon.svelte";
    import { accountShownName } from "$lib/accountDetails.js";

    /**
     * @type {{
     *   account: Partial<import("$lib/accountDetails.js").Account> | null,
     *   onLogin: () => void,
     *   onLogout: () => void | Promise<void>,
     *   accountHref?: string,
     *   settingsHref?: string,
     *   active?: boolean,
     * }}
     */
    let { account, onLogin, onLogout, accountHref = "/fiok", settingsHref = "/beallitasok", active = false } = $props();

    const menuId = "account-menu-panel";
    const shownName = $derived(account ? accountShownName(account) : "");

    let open = $state(false);
    let photoFailed = $state(false);
    /** @type {HTMLDivElement | undefined} */
    let root = $state();
    /** @type {HTMLButtonElement | undefined} */
    let button = $state();

    $effect(() => {
        // A new photo gets a fresh try.
        void account?.picture;
        photoFailed = false;
    });

    /** @param {MouseEvent} event */
    async function toggle(event) {
        open = !open;
        // detail 0: opened with Enter or Space, not a pointer.
        if (open && event.detail === 0) {
            await tick();
            root?.querySelector(".account-menu-item")?.focus();
        }
    }

    function close() {
        open = false;
    }

    async function logout() {
        close();
        await onLogout();
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

{#if !account}
    <button type="button" class="btn nav-btn" title="Belépés" onclick={() => onLogin()}>
        <AppIcon name="login" size={16} />
        <span class="sr-only">Belépés</span>
    </button>
{:else}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="account-menu" bind:this={root} onkeydown={onKeydown} onfocusout={onFocusOut}>
        <button
            bind:this={button}
            type="button"
            class="btn nav-btn account-menu-button"
            class:active={open || active}
            class:account-menu-button--photo={!!account.picture && !photoFailed}
            title={shownName ? `Fiók: ${shownName}` : "Fiók"}
            aria-expanded={open}
            aria-controls={menuId}
            onclick={toggle}
        >
            {#if account.picture && !photoFailed}
                <img
                    class="account-menu-photo"
                    src={account.picture}
                    alt=""
                    referrerpolicy="no-referrer"
                    onerror={() => (photoFailed = true)}
                />
            {:else}
                <AppIcon name="profile" size={16} />
            {/if}
            <span class="sr-only">Fiók menü</span>
        </button>
        {#if open}
            <div id={menuId} class="account-menu-panel">
                <div class="account-menu-head">
                    <span class="account-menu-name">{shownName}</span>
                    {#if account.email && account.email !== shownName}
                        <span class="account-menu-email">{account.email}</span>
                    {/if}
                </div>
                <ul class="account-menu-list">
                    <li>
                        <a class="account-menu-item" href={accountHref} onclick={close}>
                            <AppIcon name="profile" size={16} />
                            <span>Fiók</span>
                        </a>
                    </li>
                    <li>
                        <a class="account-menu-item" href={settingsHref} onclick={close}>
                            <AppIcon name="settings" size={16} />
                            <span>Beállítások</span>
                        </a>
                    </li>
                    <li>
                        <button type="button" class="account-menu-item" onclick={logout}>
                            <AppIcon name="logout" size={16} />
                            <span>Kijelentkezés</span>
                        </button>
                    </li>
                </ul>
            </div>
        {/if}
    </div>
{/if}
