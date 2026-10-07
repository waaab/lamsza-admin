<script>
	import { page } from '$app/stores';
	import ErrorPage from '$lib/components/ErrorPage.svelte';
</script>

<!-- The network's shared error page (lamsza UI_BASELINE "err-page"). Admin's
     shell (icon rail and header) lives inside its one page, so an unknown
     address shows the error on its own, like Lámsza's root error page.
     ErrorPage is synced from lamsza: edit it there, never here. -->
<main class="container admin-error">
	<ErrorPage status={$page.status} appName="Lámsza admin" />
</main>

<style>
	.admin-error {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
	}
</style>
