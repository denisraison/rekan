<script lang="ts">
	import { pb } from '$lib/pb';
	import type { GeneratedPost } from '$lib/types';

	const BUSINESS_TYPES = [
		'Confeitaria',
		'Salão de Beleza',
		'Barbearia',
		'Restaurante',
		'Loja de Roupas',
		'Personal Trainer',
		'Oficina',
		'Outro',
	];

	let businessName = $state('');
	let businessType = $state('');
	let city = $state('');
	let services = $state('');
	let message = $state('');
	let generating = $state(false);
	let error = $state('');
	let result = $state<GeneratedPost | null>(null);
	let copied = $state<string | null>(null);

	async function generate() {
		if (!businessName.trim() || !businessType || !city.trim() || !message.trim()) {
			error = 'Preencha nome do negócio, tipo, cidade e mensagem.';
			return;
		}
		generating = true;
		error = '';
		result = null;
		try {
			const res = await pb.send('/api/demo:generate', {
				method: 'POST',
				body: JSON.stringify({
					business_name: businessName.trim(),
					business_type: businessType,
					city: city.trim(),
					services: services.trim(),
					message: message.trim(),
				}),
			});
			result = res as GeneratedPost;
		} catch (err: unknown) {
			const e = err as { data?: { message?: string } };
			error = e?.data?.message ?? 'Erro ao gerar conteúdo. Tente novamente.';
		} finally {
			generating = false;
		}
	}

	async function copyText(text: string, label: string) {
		await navigator.clipboard.writeText(text);
		copied = label;
		setTimeout(() => {
			copied = null;
		}, 2000);
	}
</script>

<div
	class="min-h-screen flex flex-col items-center px-4 py-8"
	style="background: var(--bg)"
>
	<div class="w-full max-w-lg">
		<h1
			class="text-lg font-semibold mb-6"
			style="color: var(--text); font-family: var(--font-primary)"
		>
			Rekan · Experimentar
		</h1>

		<div class="flex flex-col gap-4">
			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium" style="color: var(--text)">Nome do negócio</span>
				<input
					bind:value={businessName}
					placeholder="Ex: Doces da Ana"
					class="px-3 py-2.5 rounded-xl text-sm outline-none border"
					style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
				/>
			</label>

			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium" style="color: var(--text)">Tipo</span>
				<select
					bind:value={businessType}
					class="px-3 py-2.5 rounded-xl text-sm outline-none border"
					style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
				>
					<option value="">Selecione...</option>
					{#each BUSINESS_TYPES as t}
						<option value={t}>{t}</option>
					{/each}
				</select>
			</label>

			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium" style="color: var(--text)">Cidade</span>
				<input
					bind:value={city}
					placeholder="Ex: Guarulhos"
					class="px-3 py-2.5 rounded-xl text-sm outline-none border"
					style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
				/>
			</label>

			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium" style="color: var(--text)"
					>Serviços <span class="font-normal" style="color: var(--text-muted)"
						>— opcional</span
					></span
				>
				<input
					bind:value={services}
					placeholder="Bolo caseiro R$85, Brigadeiro R$35"
					class="px-3 py-2.5 rounded-xl text-sm outline-none border"
					style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
				/>
			</label>

			<label class="flex flex-col gap-1.5">
				<span class="text-sm font-medium" style="color: var(--text)"
					>O que o cliente quer postar?</span
				>
				<textarea
					bind:value={message}
					placeholder="Ex: Fiz um bolo rosa e dourado lindo hoje, a noiva amou"
					rows={3}
					class="px-3 py-2.5 rounded-xl text-sm outline-none border resize-none"
					style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
				></textarea>
			</label>

			<button
				onclick={generate}
				disabled={generating}
				class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity self-center"
				style="background: var(--coral); color: #fff; opacity: {generating
					? '0.6'
					: '1'}; cursor: {generating ? 'not-allowed' : 'pointer'}"
			>
				{generating ? 'Gerando...' : 'Gerar post'}
			</button>
		</div>

		{#if error}
			<p class="mt-4 text-sm p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">
				{error}
			</p>
		{/if}

		{#if result}
			<div
				class="mt-6 rounded-xl p-4"
				style="background: var(--surface); border: 1px solid var(--border)"
			>
				<div class="mb-3">
					<div class="flex items-center justify-between mb-1">
						<span
							class="text-xs font-medium uppercase tracking-widest"
							style="color: var(--text-muted)">Legenda</span
						>
						<button
							onclick={() => copyText(result!.caption, 'caption')}
							class="text-xs"
							style="color: var(--coral)"
						>
							{copied === 'caption' ? 'Copiado!' : 'Copiar'}
						</button>
					</div>
					<p
						class="text-sm leading-relaxed whitespace-pre-wrap"
						style="color: var(--text)"
					>
						{result.caption}
					</p>
				</div>

				<div class="mb-3">
					<div class="flex items-center justify-between mb-1">
						<span
							class="text-xs font-medium uppercase tracking-widest"
							style="color: var(--text-muted)">Hashtags</span
						>
						<button
							onclick={() => copyText(result!.hashtags.join(' '), 'hashtags')}
							class="text-xs"
							style="color: var(--coral)"
						>
							{copied === 'hashtags' ? 'Copiado!' : 'Copiar'}
						</button>
					</div>
					<p class="text-xs" style="color: var(--text-secondary)">
						{result.hashtags.join(' ')}
					</p>
				</div>

				{#if result.production_note}
					<div>
						<span
							class="text-xs font-medium uppercase tracking-widest"
							style="color: var(--text-muted)">Direção de foto/vídeo</span
						>
						<p
							class="text-xs italic mt-1"
							style="color: var(--text-secondary); border-left: 2px solid var(--border-strong); padding-left: 0.75rem"
						>
							{result.production_note}
						</p>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
