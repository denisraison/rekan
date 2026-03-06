<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pb';

	type Clause = { title: string; paragraphs: string[] };

	let clauses = $state<Clause[]>([]);
	let error = $state(false);

	onMount(async () => {
		try {
			clauses = await pb.send('/api/terms', { method: 'GET' });
		} catch {
			error = true;
		}
	});
</script>

<svelte:head>
	<title>Termos de Uso — Rekan</title>
</svelte:head>

<div class="min-h-screen px-4 py-12" style="background: var(--bg)">
	<article class="max-w-2xl mx-auto text-sm leading-relaxed" style="color: var(--text-secondary)">
		<a href="/" class="inline-block text-xs mb-6 underline" style="color: var(--text-muted)">&larr; Voltar</a>
		<h1 class="text-2xl font-semibold mb-1" style="color: var(--text); font-family: var(--font-primary)">
			Termos de Uso do Serviço Rekan
		</h1>
		<p class="text-xs mb-8" style="color: var(--text-muted)">Última atualização: fevereiro de 2026</p>

		{#if error}
			<p style="color: var(--text-muted)">Erro ao carregar os termos. Tente novamente.</p>
		{:else if clauses.length === 0}
			<p style="color: var(--text-muted)">Carregando...</p>
		{:else}
			{#each clauses as clause, i}
				<section class={i < clauses.length - 1 ? 'mb-4' : ''}>
					<p class="mb-2">
						<strong style="color: var(--text)">{clause.title}</strong>
						{' '}{clause.paragraphs[0]}
					</p>
					{#each clause.paragraphs.slice(1) as paragraph}
						<p class="mb-2">{paragraph}</p>
					{/each}
				</section>
			{/each}
		{/if}
	</article>
</div>
