<script>
    let {
        value = $bindable(""),
        id = "",
        name = "",
        required = false,
        disabled = false,
        min = "",
        class: className = "",
    } = $props();

    function toDisplay(raw) {
        const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(raw ?? ""));
        if (!match) return "";
        return `${match[1]}. ${match[2]}. ${match[3]}.`;
    }

    let text = $state("");
    let focused = $state(false);
    let shown = $derived(focused ? text : toDisplay(value));

    function onFocus() {
        text = toDisplay(value);
        focused = true;
    }

    function commit() {
        focused = false;
        const match = /^(\d{4})\.\s*(\d{1,2})\.\s*(\d{1,2})\.?$/.exec(text.trim());
        if (!match) {
            if (text.trim() === "") value = "";
            return;
        }
        const year = Number(match[1]);
        const month = Number(match[2]);
        const day = Number(match[3]);
        const date = new Date(year, month - 1, day);
        if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return;
        const next = `${match[1]}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
        if (min && next < min) return;
        value = next;
        text = toDisplay(value);
    }
</script>

<input
    {id}
    {name}
    type="text"
    inputmode="numeric"
    autocomplete="off"
    placeholder="éééé. hh. nn."
    class={className}
    {required}
    {disabled}
    value={shown}
    oninput={(event) => (text = event.currentTarget.value)}
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
