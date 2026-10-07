<script>
	import { page } from '$app/stores';
	import ErrorShell from '$lib/components/ErrorShell.svelte';

	const lamszaOrigin = import.meta.env.VITE_LAMSZA_ORIGIN || 'http://localhost:5174';
</script>

<!-- The network's shared minimal error shell (lamsza UI_BASELINE "err-page"):
     the Lámsza and Admin buttons, no icon rail or header, nothing fetched.
     Admin's own shell lives inside its one page, so the error page never gets
     it. ErrorShell is synced from lamsza: edit it there, never here. -->
<ErrorShell
	status={$page.status}
	appName="Lámsza admin"
	lamszaHref={lamszaOrigin}
	app={{ label: 'Admin', icon: 'dashboard' }}
/>
