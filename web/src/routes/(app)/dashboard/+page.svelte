<script lang="ts">
	import { pb } from '$lib/pb';
	import { onMount } from 'svelte';
	import type { Business } from '$lib/types';

	let business = $state<Business | null>(null);
	let loading = $state(true);

	onMount(async () => {
		try {
			const result = await pb.collection('businesses').getList<Business>(1, 1);
			business = result.items[0] ?? null;
		} finally {
			loading = false;
		}
	});

	async function logout() {
		pb.authStore.clear();
		window.location.href = '/login';
	}
</script>

<div class="min-h-screen" style="background: var(--bg)">
	<header class="border-b px-6 py-4 flex items-center justify-between" style="background: var(--surface); border-color: var(--border)">
		<span class="font-semibold" style="color: var(--text); font-family: var(--font-primary)">Rekan</span>
		<button
			onclick={logout}
			class="text-sm"
			style="color: var(--text-secondary); font-family: var(--font-primary)"
		>
			Sair
		</button>
	</header>

	<main class="max-w-2xl mx-auto px-6 py-12">
		{#if loading}
			<p class="text-sm" style="color: var(--text-muted)">Carregando...</p>
		{:else if business}
			<h1 class="text-2xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
				{business.name}
			</h1>
			<p class="text-sm mb-8" style="color: var(--text-secondary)">
				{business.type} · {business.city}, {business.state}
			</p>

			<div
				class="rounded-2xl p-8 text-center"
				style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
			>
				<p class="text-sm" style="color: var(--text-muted)">
					Geração de conteúdo em breve.
				</p>
			</div>
		{:else}
			<p class="text-sm" style="color: var(--text-muted)">Nenhum negócio encontrado.</p>
		{/if}
	</main>
</div>
