<script>
	import "../styles/global.css";
	import "../styles/admin.css";
	import { onMount } from "svelte";
	import { applyTheme } from "$lib/stores/theme";

	onMount(() => {
		const saved = localStorage.getItem("theme") || "system";
		applyTheme(saved);
	});
</script>

<slot />
