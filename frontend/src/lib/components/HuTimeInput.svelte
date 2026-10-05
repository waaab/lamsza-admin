<script>
    let {
        value = $bindable(""),
        id = "",
        name = "",
        required = false,
        disabled = false,
        placeholder = "óó:pp",
        ariaLabel = "",
        class: className = "",
        onchange,
    } = $props();

    function normalize(raw) {
        const match = /^(\d{2}):(\d{2})(?::\d{2})?$/.exec(String(raw ?? "").trim());
        if (!match) return "";
        const hour = Number(match[1]);
        const minute = Number(match[2]);
        if (hour > 23 || minute > 59) return "";
        return `${match[1]}:${match[2]}`;
    }

    let text = $state("");
    let focused = $state(false);
    let shown = $derived(focused ? text : normalize(value));

    function onFocus() {
        text = normalize(value);
        focused = true;
    }

    function onInput(event) {
        const digits = event.currentTarget.value.replace(/\D/g, "").slice(0, 4);
        text = digits.length > 2 ? `${digits.slice(0, 2)}:${digits.slice(2)}` : digits;
    }

    function commit() {
        focused = false;
        const match = /^(\d{1,2}):(\d{2})$/.exec(text.trim());
        if (!match) {
            if (text.trim() === "") value = "";
            onchange?.(value);
            return;
        }
        const hour = Number(match[1]);
        const minute = Number(match[2]);
        if (hour > 23 || minute > 59) return;
        const next = `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
        value = next;
        text = next;
        onchange?.(next);
    }
</script>

<input
    {id}
    {name}
    type="text"
    inputmode="numeric"
    autocomplete="off"
    maxlength="5"
    class={className}
    {required}
    {disabled}
    {placeholder}
    aria-label={ariaLabel || undefined}
    value={shown}
    oninput={onInput}
    onfocus={onFocus}
    onblur={commit}
/>

<style>
    input {
        width: 100%;
        min-width: 0;
        box-sizing: border-box;
    }
</style>
