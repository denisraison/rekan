<script lang="ts">
	import { pb } from '$lib/pb';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		if (!pb.authStore.isValid) {
			goto('/login');
		}
	});
</script>

{#if pb.authStore.isValid}
	{@render children()}
{/if}
