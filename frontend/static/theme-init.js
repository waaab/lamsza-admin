// Set the saved theme before the first paint so the page does not flash in the
// wrong colours. This lives in its own file, not inline in app.html, because the
// Content-Security-Policy (see svelte.config.js) allows scripts from this origin
// only and would block an inline block.
(function () {
    var t = localStorage.getItem('theme');
    if (t === 'dark' || t === 'light') {
        document.documentElement.setAttribute('data-theme', t);
    }
    // 'system' or missing: leave data-theme unset → CSS media query handles it
})();
