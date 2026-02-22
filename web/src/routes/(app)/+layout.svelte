<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pb';

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
