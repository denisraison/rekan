<script lang="ts">
	import '../app.css';
	// PEP-018: deploy timing test
	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href="/brand/favicon.png" />
	<link rel="apple-touch-icon" href="/icons/icon-192.png" />
</svelte:head>

{@render children()}
