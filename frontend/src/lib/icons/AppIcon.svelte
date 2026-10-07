<script>
    /**
     * Shared icons for admin (sidebar + welcome). Public toolbar markup lives in
     * (public)/+layout.svelte - copy paths here when an entity should match the site.
     */
    export let name = "entries";
    /** Rendered box size (px). Toolbar uses 24; admin sidebar 28; welcome grid 56. */
    export let size = 24;
    let className = "";
    export { className as class };
</script>

<span
    class="app-icon-wrap {className}"
    style:width="{size}px"
    style:height="{size}px"
    aria-hidden="true"
>
    {#if name === "home"}
        <!-- Same magnifying glass as the Lámsza mark in Szótár and Játszótér -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"></circle>
            <path d="m21 21-4.3-4.3"></path>
        </svg>
    {:else if name === "profile"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"></path>
            <circle cx="12" cy="7" r="4"></circle>
        </svg>
    {:else if name === "dashboard"}
        <!-- Four-up board: three tiles and a gear -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="7.5" height="7.5" rx="1.6"></rect>
            <rect x="13.5" y="3" width="7.5" height="7.5" rx="1.6"></rect>
            <rect x="3" y="13.5" width="7.5" height="7.5" rx="1.6"></rect>
            <circle cx="17.25" cy="17.25" r="2.05"></circle>
            <path d="M17.25 13.35v1.15M17.25 20v1.15M13.35 17.25h1.15M20 17.25h1.15M14.55 14.55l.8.8M19.15 19.15l.8.8M19.95 14.55l-.8.8M15.35 19.15l-.8.8"></path>
        </svg>
    {:else if name === "entries"}
        <!-- Same as homepage “Indexelünk” -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="10" x2="21" y1="6" y2="6"></line>
            <line x1="10" x2="21" y1="12" y2="12"></line>
            <line x1="10" x2="21" y1="18" y2="18"></line>
            <path d="M4 6h1v4"></path>
            <path d="M4 10h2"></path>
            <path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1"></path>
        </svg>
    {:else if name === "websites"}
        <!-- Same list as Index, with W W instead of 1 and 2 -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="10" x2="21" y1="6" y2="6"></line>
            <line x1="10" x2="21" y1="12" y2="12"></line>
            <line x1="10" x2="21" y1="18" y2="18"></line>
            <path d="M2.3 3.5 3.8 8.7 5.3 5.5 6.8 8.7 8.3 3.5"></path>
            <path d="M2.3 14.2 3.8 19.4 5.3 16.2 6.8 19.4 8.3 14.2"></path>
        </svg>
    {:else if name === "mondasok"}
        <!-- Hungarian „ - scaled to match visual weight of 24×24 stroke icons -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" overflow="visible" class="app-icon-quote">
            <g transform="translate(12 17.5) scale(1.62) translate(-12 -17.5)">
                <text
                    x="12"
                    y="18"
                    text-anchor="middle"
                    font-size="24"
                    font-weight="800"
                    font-family="Georgia, 'Times New Roman', Times, serif">„</text>
            </g>
        </svg>
    {:else if name === "quicklinks"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"
            ><path
                d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"
            ></path><path
                d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"
            ></path></svg
        >
    {:else if name === "newsfeeds"}
        <!-- Admin: newspaper (matches public /hirek toolbar art in +layout.svelte) -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a4 4 0 0 1-4-4V6"></path>
            <path d="M18 14h-8"></path>
            <path d="M15 18h-5"></path>
            <path d="M10 6h8v4h-8V6Z"></path>
        </svg>
    {:else if name === "counties"}
        <!-- Same shield as homepage “A megyétől” -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
        </svg>
    {:else if name === "locations"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"
            ><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path><circle cx="12" cy="10" r="3"></circle></svg
        >
    {:else if name === "my_location"}
        <!-- Target: outer ring, inner ring, center, four ticks -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="6.25"></circle>
            <circle cx="12" cy="12" r="2.35"></circle>
            <circle cx="12" cy="12" r="0.9" fill="currentColor" stroke="none"></circle>
            <line x1="12" y1="1.25" x2="12" y2="3.75"></line>
            <line x1="12" y1="20.25" x2="12" y2="22.75"></line>
            <line x1="1.25" y1="12" x2="3.75" y2="12"></line>
            <line x1="20.25" y1="12" x2="22.75" y2="12"></line>
        </svg>
    {:else if name === "venues"}
        <!-- Stadium: rim, bowl, entrance notch, pennants (.app-icon-solid) -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon class="app-icon-solid" points="6.5,2.5 8,5 5,5" />
            <polygon class="app-icon-solid" points="12,1.5 13.5,4 10.5,4" />
            <polygon class="app-icon-solid" points="17.5,2.5 19,5 16,5" />
            <path d="M 5 9 Q 12 5.6 19 9"></path>
            <path d="M 5 9 L 5 15.5 Q 5 18.6 8.6 18.6"></path>
            <path d="M 19 9 L 19 15.5 Q 19 18.6 15.4 18.6"></path>
        </svg>
    {:else if name === "attractions"}
        <!-- View Apps–style triptych: taller outer edges, visible gaps -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M 2 3.5 L 8.5 5.2 L 8.5 18.8 L 2 20.5 Z"></path>
            <path d="M 9.5 5.2 H 14.5 V 18.8 H 9.5 Z"></path>
            <path d="M 15.5 5.2 L 22 3.5 L 22 20.5 L 15.5 18.8 Z"></path>
        </svg>
    {:else if name === "events"}
        <!-- Same calendar as public toolbar (+layout.svelte) -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M8 2v4"></path>
            <path d="M16 2v4"></path>
            <rect width="18" height="18" x="3" y="4" rx="2"></rect>
            <path d="M3 10h18"></path>
        </svg>
    {:else if name === "entry_categories"}
        <!-- Category: triangle, square, circle -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="12 2.2 6.15 10.8 17.85 10.8"></polygon>
            <rect x="2.5" y="13.6" width="7.5" height="7.5"></rect>
            <circle cx="17.65" cy="17.35" r="3.75"></circle>
        </svg>
    {:else if name === "category_search"}
        <!-- Category search: triangle, square, magnifying glass -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="12 2.2 6.15 10.8 17.85 10.8"></polygon>
            <rect x="2.5" y="13.6" width="7.5" height="7.5"></rect>
            <circle cx="16.45" cy="16.35" r="3.25"></circle>
            <line x1="18.85" y1="18.75" x2="21.2" y2="21.1"></line>
        </svg>
    {:else if name === "entry_types"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
            <polyline points="2 17 12 22 22 17"></polyline>
            <polyline points="2 12 12 17 22 12"></polyline>
        </svg>
    {:else if name === "tags"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="4" y1="9" x2="20" y2="9"></line>
            <line x1="4" y1="15" x2="20" y2="15"></line>
            <line x1="10" y1="3" x2="8" y2="21"></line>
            <line x1="16" y1="3" x2="14" y2="21"></line>
        </svg>
    {:else if name === "szekek"}
        <!-- Side-view chair (matches public toolbar Székek) -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="6" y1="5" x2="6" y2="21"></line>
            <line x1="6" y1="15" x2="19" y2="15"></line>
            <line x1="19" y1="15" x2="19" y2="21"></line>
        </svg>
    {:else if name === "varosok"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="8" height="18" x="3" y="3" rx="2"></rect>
            <path d="M7 7h0"></path>
            <path d="M7 11h0"></path>
            <path d="M7 15h0"></path>
            <rect width="8" height="12" x="13" y="9" rx="2"></rect>
            <path d="M17 13h0"></path>
            <path d="M17 17h0"></path>
        </svg>
    {:else if name === "falvak"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 20v-8l7-5 7 5v8"></path>
            <path d="M7 20v-4h6v4"></path>
            <path d="M17 20h4v-7l-4-3"></path>
            <path d="M3 20h18"></path>
        </svg>
    {:else if name === "x"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
    {:else if name === "users"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
            <circle cx="9" cy="7" r="4"></circle>
            <path d="M22 21v-2a4 4 0 0 0-3-3.87"></path>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
        </svg>
    {:else if name === "settings"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            ><circle cx="12" cy="12" r="3"></circle><path
                d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
            ></path></svg
        >
    {:else if name === "weather_translations"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            ><path d="M5 8l6 6"></path><path d="M4 14l6-6 2-3"></path><path d="M2 5h12"></path><path d="M7 2h3l2 2 2 4-3 1"></path><path d="M22 22l-5-10-5 10"></path><path d="M14 18h6"></path></svg
        >
    {:else if name === "pages"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            ><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg
        >
    {:else if name === "info"}
        <!-- Circle with a lowercase i -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="16" x2="12" y2="11"></line>
            <line x1="12" y1="8" x2="12.01" y2="8"></line>
        </svg>
    {:else if name === "collapse"}
        <!-- Horizontal corners pointing inward -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="4 8 8 12 4 16"></polyline>
            <polyline points="20 8 16 12 20 16"></polyline>
        </svg>
    {:else if name === "expand"}
        <!-- Horizontal corners pointing outward -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="8 8 4 12 8 16"></polyline>
            <polyline points="16 8 20 12 16 16"></polyline>
        </svg>
    {:else if name === "upload"}
        <!-- Tray with an arrow leaving it -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="17 8 12 3 7 8"></polyline>
            <line x1="12" y1="3" x2="12" y2="15"></line>
        </svg>
    {:else if name === "szotar"}
        <!-- Open book. Same stroke set as Települések / Hírek (24 viewBox, stroke 2). -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
            <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
        </svg>
    {:else if name === "jatszoter"}
        <!-- Játszótér: the cross of tiles from its own toolbar (UI_BASELINE "ic-jatszoter"), at the set's stroke 2. -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 3h6v12h-6v6H9v-6H3V9h12V3z"></path>
            <path d="M15 9h6M9 9v6M15 9v6M9 15h6"></path>
        </svg>
    {:else if name === "trophy"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6"></path>
            <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18"></path>
            <path d="M4 22h16"></path>
            <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22"></path>
            <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22"></path>
            <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z"></path>
        </svg>
    {:else if name === "login"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path>
            <polyline points="10 17 15 12 10 7"></polyline>
            <line x1="15" y1="12" x2="3" y2="12"></line>
        </svg>
    {:else if name === "logout"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
            <polyline points="16 17 21 12 16 7"></polyline>
            <line x1="21" y1="12" x2="9" y2="12"></line>
        </svg>
    {:else if name === "list"}
        <!-- Szótár "Lista": three lines getting shorter -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="4" y1="6" x2="16" y2="6"></line>
            <line x1="4" y1="12" x2="12" y2="12"></line>
            <line x1="4" y1="18" x2="8" y2="18"></line>
        </svg>
    {:else if name === "add"}
        <!-- Plus in a circle (Szótár "Új szó") -->
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <path d="M12 8v8"></path>
            <path d="M8 12h8"></path>
        </svg>
    {:else if name === "page_faq"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
            <path d="M12 17h.01"></path>
        </svg>
    {:else if name === "category-etkezes"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M280-80v-366q-51-14-85.5-56T160-600v-280h80v280h40v-280h80v280h40v-280h80v280q0 56-34.5 98T360-446v366h-80Zm400 0v-320H560v-280q0-83 58.5-141.5T760-880v800h-80Z"></path>
        </svg>
    {:else if name === "category-szallas"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M40-200v-600h80v400h320v-320h320q66 0 113 47t47 113v360h-80v-120H120v120H40Zm155-275q-35-35-35-85t35-85q35-35 85-35t85 35q35 35 35 85t-35 85q-35 35-85 35t-85-35Zm325 75h320v-160q0-33-23.5-56.5T760-640H520v240ZM308.5-531.5Q320-543 320-560t-11.5-28.5Q297-600 280-600t-28.5 11.5Q240-577 240-560t11.5 28.5Q263-520 280-520t28.5-11.5ZM280-560Zm240-80v240-240Z"></path>
        </svg>
    {:else if name === "category-egeszseg"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="10" cy="13" r="7"></circle>
            <path d="M10 10v6"></path>
            <path style="stroke: var(--szekely-red)" d="M7 13h6"></path>
            <path style="stroke: var(--szekely-red)" d="M18 3v3"></path>
            <path style="stroke: var(--szekely-red)" d="M16.5 4.5h3"></path>
        </svg>
    {:else if name === "category-vasarlas"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M841-518v318q0 33-23.5 56.5T761-120H201q-33 0-56.5-23.5T121-200v-318q-23-21-35.5-54t-.5-72l42-136q8-26 28.5-43t47.5-17h556q27 0 47 16.5t29 43.5l42 136q12 39-.5 71T841-518Zm-272-42q27 0 41-18.5t11-41.5l-22-140h-78v148q0 21 14 36.5t34 15.5Zm-180 0q23 0 37.5-15.5T441-612v-148h-78l-22 140q-4 24 10.5 42t37.5 18Zm-178 0q18 0 31.5-13t16.5-33l22-154h-78l-40 134q-6 20 6.5 43t41.5 23Zm540 0q29 0 42-23t6-43l-42-134h-76l22 154q3 20 16.5 33t31.5 13ZM201-200h560v-282q-5 2-6.5 2H751q-27 0-47.5-9T663-518q-18 18-41 28t-49 10q-27 0-50.5-10T481-518q-17 18-39.5 28T393-480q-29 0-52.5-10T299-518q-21 21-41.5 29.5T211-480h-4.5q-2.5 0-5.5-2v282Zm560 0H201h560Z"></path>
        </svg>
    {:else if name === "category-auto"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor">
            <path fill-rule="evenodd" d="M2.52 3.515A2.5 2.5 0 0 1 4.82 2h6.362c1 0 1.904.596 2.298 1.515l.792 1.848c.075.175.21.319.38.404.5.25.855.715.965 1.262l.335 1.679c.033.161.049.325.049.49v.413c0 .814-.39 1.543-1 1.997V13.5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1-.5-.5v-1.338c-1.292.048-2.745.088-4 .088s-2.708-.04-4-.088V13.5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1-.5-.5v-1.892c-.61-.454-1-1.183-1-1.997v-.413a2.5 2.5 0 0 1 .049-.49l.335-1.68c.11-.546.465-1.012.964-1.261a.807.807 0 0 0 .381-.404l.792-1.848ZM3 10a1 1 0 1 0 0-2 1 1 0 0 0 0 2Zm10 0a1 1 0 1 0 0-2 1 1 0 0 0 0 2ZM6 8a1 1 0 0 0 0 2h4a1 1 0 1 0 0-2H6ZM2.906 5.189l.956-1.913A.5.5 0 0 1 4.309 3h7.382a.5.5 0 0 1 .447.276l.956 1.913a.51.51 0 0 1-.497.731c-.91-.073-3.35-.17-4.597-.17-1.247 0-3.688.097-4.597.17a.51.51 0 0 1-.497-.731Z"></path>
        </svg>
    {:else if name === "category-mester"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M739-83.5q-7-2.5-13-8.5L522-296q-6-6-8.5-13t-2.5-15q0-8 2.5-15t8.5-13l85-85q6-6 13-8.5t15-2.5q8 0 15 2.5t13 8.5l204 204q6 6 8.5 13t2.5 15q0 8-2.5 15t-8.5 13l-85 85q-6 6-13 8.5T754-81q-8 0-15-2.5Zm15-92.5 29-29-147-147-29 29 147 147ZM189.5-83q-7.5-3-13.5-9l-84-84q-6-6-9-13.5T80-205q0-8 3-15t9-13l212-212h85l34-34-165-165h-57L80-765l113-113 121 121v57l165 165 116-116-43-43 56-56H495l-28-28 142-142 28 28v113l56-56 142 142q17 17 26 38.5t9 45.5q0 24-9 46t-26 39l-85-85-56 56-42-42-207 207v84L233-92q-6 6-13 9t-15 3q-8 0-15.5-3Zm15.5-93 170-170v-29h-29L176-205l29 29Zm0 0-29-29 15 14 14 15Zm549 0 29-29-29 29Z"></path>
        </svg>
    {:else if name === "category-oktatas"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 4 3 9l9 5 9-5-9-5z"></path>
            <path d="M7 11.2V16c0 1.6 2.2 3 5 3s5-1.4 5-3v-4.8"></path>
            <path style="stroke: var(--szekely-red)" d="M21 9v6"></path>
        </svg>
    {:else if name === "category-hivatal"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M200-280v-280h80v280h-80Zm240 0v-280h80v280h-80ZM80-120v-80h800v80H80Zm600-160v-280h80v280h-80ZM80-640v-80l400-200 400 200v80H80Zm178-80h444-444Zm0 0h444L480-830 258-720Z"></path>
        </svg>
    {:else if name === "category-sport"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="8"></circle>
            <circle cx="12" cy="12" r="4"></circle>
            <circle cx="12" cy="12" r="1.3" style="fill: var(--szekely-red); stroke: none"></circle>
            <path d="M12 2.5V5"></path>
            <path d="M12 19v2.5"></path>
            <path d="M2.5 12H5"></path>
            <path d="M19 12h2.5"></path>
        </svg>
    {:else if name === "category-penzugy"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="6" width="18" height="12" rx="2"></rect>
            <path style="stroke: var(--szekely-red)" d="M3 10h18"></path>
            <path d="M7 15h4"></path>
        </svg>
    {:else if name === "category-informatika"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M40-120v-80h880v80H40Zm120-120q-33 0-56.5-23.5T80-320v-440q0-33 23.5-56.5T160-840h640q33 0 56.5 23.5T880-760v440q0 33-23.5 56.5T800-240H160Zm0-80h640v-440H160v440Zm0 0v-440 440Z"></path>
        </svg>
    {:else if name === "category-szakma"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="currentColor">
            <path d="M160-120q-33 0-56.5-23.5T80-200v-440q0-33 23.5-56.5T160-720h160v-80q0-33 23.5-56.5T400-880h160q33 0 56.5 23.5T640-800v80h160q33 0 56.5 23.5T880-640v440q0 33-23.5 56.5T800-120H160Zm240-600h160v-80H400v80Zm400 360H600v80H360v-80H160v160h640v-160Zm-360 0h80v-80h-80v80Zm-280-80h200v-80h240v80h200v-200H160v200Zm320 40Z"></path>
        </svg>
    {:else if name === "category-default"}
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="6" cy="12" r="1.2"></circle>
            <circle cx="12" cy="12" r="1.2"></circle>
            <circle cx="18" cy="12" r="1.2"></circle>
        </svg>
    {/if}
</span>

<style>
    .app-icon-wrap {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    .app-icon-wrap :global(svg) {
        display: block;
        width: 100%;
        height: 100%;
    }
</style>