<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pb';

	let { children } = $props();

	let isAuth = $state(false);

	onMount(() => {
		if (!pb.authStore.isValid) {
			goto('/entrar');
			return;
		}
		isAuth = true;

		return pb.authStore.onChange(() => {
			isAuth = pb.authStore.isValid;
			if (!isAuth) goto('/entrar');
		});
	});
</script>

{#if isAuth}
	{@render children()}
{/if}
