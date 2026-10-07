<!--
    Network error illustration (shared by all four apps through
    scripts/sync-shared-frontend.sh): a swinging lantern searching in the dark,
    with a moth that cannot resist the light.

    Inline SVG, not an <img>, so it follows the theme. Its three colours come
    from the design tokens and can be overridden with --err-ink, --err-glow and
    --err-accent:
      ink    lines and body    --text-muted
      glow   the lamplight     --warm-light
      accent the lantern cap   --szekely-red
    Motion stops when the visitor prefers reduced motion.
-->
<script>
    /** Rendered width in px; the height follows the 400 × 320 artwork. */
    let { size = 240 } = $props();
</script>

<svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 400 320"
    width={size}
    height={Math.round((size * 320) / 400)}
    class="lmz-err-art"
    aria-hidden="true"
    focusable="false"
>
    <defs>
        <radialGradient id="lmz-halo">
            <stop offset="0" style="stop-color: var(--glow); stop-opacity: 0.45" />
            <stop offset="1" style="stop-color: var(--glow); stop-opacity: 0" />
        </radialGradient>
        <linearGradient id="lmz-cone" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" style="stop-color: var(--glow); stop-opacity: 0.32" />
            <stop offset="1" style="stop-color: var(--glow); stop-opacity: 0" />
        </linearGradient>
    </defs>

    <!-- fireflies -->
    <circle class="lmz-spark lmz-glow-f" cx="62" cy="72" r="2" />
    <circle class="lmz-spark lmz-glow-f" cx="332" cy="52" r="1.6" style="animation-delay: -0.8s" />
    <circle class="lmz-spark lmz-glow-f" cx="92" cy="208" r="1.8" style="animation-delay: -1.6s" />
    <circle class="lmz-spark lmz-glow-f" cx="312" cy="186" r="2.2" style="animation-delay: -2.3s" />
    <circle class="lmz-spark lmz-glow-f" cx="138" cy="38" r="1.4" style="animation-delay: -1.1s" />
    <circle class="lmz-spark lmz-glow-f" cx="352" cy="248" r="1.6" style="animation-delay: -2.8s" />
    <circle class="lmz-spark lmz-glow-f" cx="40" cy="150" r="1.4" style="animation-delay: -0.4s" />

    <!-- pool of light on the ground -->
    <ellipse class="lmz-pool lmz-glow-f" cx="200" cy="296" rx="96" ry="9" opacity="0.14" />

    <!-- the lantern -->
    <g class="lmz-swing">
        <polygon points="186,143 214,143 300,296 100,296" fill="url(#lmz-cone)" />
        <circle cx="200" cy="112" r="64" fill="url(#lmz-halo)" />
        <line class="lmz-ink-s" x1="200" y1="0" x2="200" y2="52" stroke-width="2" />
        <circle class="lmz-ink-s" cx="200" cy="58" r="6" stroke-width="2.5" />
        <path class="lmz-cap-f" d="M174 80 L200 64 L226 80 Z" />
        <rect class="lmz-ink-f" x="172" y="78" width="56" height="7" rx="3.5" />
        <rect class="lmz-glow-f" x="180" y="85" width="40" height="51" rx="5" opacity="0.22" />
        <path class="lmz-ink-s" d="M193 87 V134 M207 87 V134" stroke-width="1.5" opacity="0.45" />
        <path class="lmz-glow-f" d="M200 99 C208 108 208 119 200 124 C192 119 192 108 200 99 Z" />
        <rect class="lmz-ink-s" x="180" y="85" width="40" height="51" rx="5" stroke-width="2.5" />
        <rect class="lmz-ink-f" x="174" y="136" width="52" height="7" rx="3.5" />
    </g>

    <!-- the moth -->
    <g class="lmz-orbit">
        <g transform="translate(278 112)">
            <g class="lmz-wings">
                <ellipse class="lmz-ink-f" cx="-7" cy="-2" rx="8" ry="5.5" transform="rotate(-20 -7 -2)" opacity="0.7" />
                <ellipse class="lmz-ink-f" cx="7" cy="-2" rx="8" ry="5.5" transform="rotate(20 7 -2)" opacity="0.7" />
            </g>
            <ellipse class="lmz-ink-f" cx="0" cy="0" rx="2.4" ry="6.5" />
            <path class="lmz-ink-s" d="M-1 -6 Q-4 -12 -7 -13 M1 -6 Q4 -12 7 -13" stroke-width="1" />
        </g>
    </g>
</svg>

<style>
    .lmz-err-art {
        --ink: var(--err-ink, var(--text-muted, currentColor));
        --glow: var(--err-glow, var(--warm-light, #f2b44f));
        --cap: var(--err-accent, var(--szekely-red, #c8463c));
        display: block;
        max-width: 100%;
        height: auto;
        overflow: visible;
    }
    .lmz-ink-s {
        stroke: var(--ink);
        fill: none;
        stroke-linecap: round;
        stroke-linejoin: round;
    }
    .lmz-ink-f {
        fill: var(--ink);
    }
    .lmz-glow-f {
        fill: var(--glow);
    }
    .lmz-cap-f {
        fill: var(--cap);
    }
    .lmz-swing {
        transform-origin: 200px 0;
        animation: lmz-swing 4.5s ease-in-out infinite alternate;
    }
    .lmz-orbit {
        transform-origin: 200px 112px;
        animation: lmz-orbit 9s linear infinite;
    }
    .lmz-wings {
        transform-box: fill-box;
        transform-origin: center;
        animation: lmz-flap 0.16s ease-in-out infinite alternate;
    }
    .lmz-spark {
        animation: lmz-twinkle 3.2s ease-in-out infinite;
    }
    .lmz-pool {
        animation: lmz-pool 4.5s ease-in-out infinite alternate;
    }
    @keyframes lmz-swing {
        from {
            transform: rotate(-7deg);
        }
        to {
            transform: rotate(7deg);
        }
    }
    @keyframes lmz-orbit {
        to {
            transform: rotate(360deg);
        }
    }
    @keyframes lmz-flap {
        to {
            transform: scaleX(0.35);
        }
    }
    @keyframes lmz-twinkle {
        50% {
            opacity: 0.15;
        }
    }
    /* rotate(-7deg) about the hook moves the cone's foot about 36 units right, rotate(7deg) 36 units left */
    @keyframes lmz-pool {
        from {
            transform: translateX(36px);
        }
        to {
            transform: translateX(-36px);
        }
    }
    @media (prefers-reduced-motion: reduce) {
        .lmz-swing,
        .lmz-orbit,
        .lmz-wings,
        .lmz-spark,
        .lmz-pool {
            animation: none;
        }
    }
</style>
