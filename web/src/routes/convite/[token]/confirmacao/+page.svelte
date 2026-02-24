<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import { pb } from '$lib/pb';

	let token = $derived($page.params.token);

	let loading = $state(true);
	let status = $state<string | null>(null);
	let clientName = $state('');
	let errorState = $state<'expired' | 'not_found' | null>(null);
	let timedOut = $state(false);

	let pollCount = 0;
	let pollTimer: ReturnType<typeof setInterval>;

	onMount(async () => {
		await fetchStatus();
		loading = false;

		if (status === 'accepted') {
			pollTimer = setInterval(async () => {
				pollCount++;
				if (pollCount >= 120) {
					timedOut = true;
					clearInterval(pollTimer);
					return;
				}
				await fetchStatus();
				if (status !== 'accepted') {
					clearInterval(pollTimer);
				}
			}, 5000);
		}
	});

	onDestroy(() => {
		clearInterval(pollTimer);
	});

	async function fetchStatus() {
		try {
			const res = await pb.send(`/api/invites/${token}`, { method: 'GET' });
			status = res.status;
			clientName = res.client_name || '';
		} catch (err: unknown) {
			const e = err as { status?: number };
			if (e?.status === 410) {
				errorState = 'expired';
			} else {
				errorState = 'not_found';
			}
		}
	}
</script>

<svelte:head>
	<title>Confirmação — Rekan</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center px-4 py-12" style="background: var(--bg)">
	<div class="w-full max-w-md">
		<div class="flex justify-center mb-8">
			<LogoCombo />
		</div>

		<div
			class="rounded-2xl p-8 text-center"
			style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
		>
			{#if loading}
				<p class="text-sm" style="color: var(--text-muted)">Carregando...</p>
			{:else if errorState === 'expired'}
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Link expirado
				</h1>
				<p class="text-sm" style="color: var(--text-secondary)">
					Este convite não é mais válido.
				</p>
			{:else if errorState === 'not_found'}
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Link inválido
				</h1>
				<p class="text-sm" style="color: var(--text-secondary)">
					Este link não existe.
				</p>
			{:else if status === 'active'}
				<div
					class="w-12 h-12 rounded-full flex items-center justify-center mx-auto mb-4"
					style="background: #DEF7EC"
				>
					<svg viewBox="0 0 20 20" fill="#03543F" width="24" height="24">
						<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
					</svg>
				</div>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Tudo certo{clientName ? `, ${clientName}` : ''}!
				</h1>
				<p class="text-sm mb-4" style="color: var(--text-secondary)">
					Pagamento confirmado. O Rekan já está pronto para gerar conteúdo para o seu negócio. Em breve você receberá seus primeiros posts pelo WhatsApp.
				</p>
				<!-- TODO: replace phone number -->
				<a
					href="https://wa.me/5500000000000?text=Oi,%20acabei%20de%20assinar!"
					target="_blank"
					rel="noopener"
					class="inline-block px-5 py-2.5 rounded-full text-sm font-medium"
					style="background: #25D366; color: #fff"
				>
					Falar no WhatsApp
				</a>
			{:else if status === 'accepted' && !timedOut}
				<div class="mb-4">
					<svg class="animate-spin h-8 w-8 mx-auto" viewBox="0 0 24 24" fill="none" style="color: var(--coral)">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z" />
					</svg>
				</div>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Aguardando confirmação do pagamento...
				</h1>
				<p class="text-sm" style="color: var(--text-secondary)">
					Assim que o PIX for confirmado, esta página será atualizada automaticamente.
				</p>
			{:else if status === 'accepted' && timedOut}
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Pagamento ainda não confirmado
				</h1>
				<p class="text-sm mb-4" style="color: var(--text-secondary)">
					Se você já realizou o pagamento, pode levar alguns minutos para a confirmação. Caso tenha dúvidas, fale com a gente.
				</p>
				<!-- TODO: replace phone number -->
				<a
					href="https://wa.me/5500000000000?text=Oi,%20fiz%20o%20pagamento%20mas%20nao%20confirmou"
					target="_blank"
					rel="noopener"
					class="inline-block px-5 py-2.5 rounded-full text-sm font-medium"
					style="background: #25D366; color: #fff"
				>
					Falar no WhatsApp
				</a>
			{:else}
				<p class="text-sm" style="color: var(--text-muted)">
					Status desconhecido. Entre em contato pelo WhatsApp.
				</p>
			{/if}
		</div>
	</div>
</div>
