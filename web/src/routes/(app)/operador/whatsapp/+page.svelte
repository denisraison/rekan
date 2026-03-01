<script lang="ts">
	import QRCode from 'qrcode';
	import { onDestroy, onMount } from 'svelte';
	import { pb } from '$lib/pb';
	import { readSSE } from '$lib/sse';
	import type { WAStatus } from '$lib/types';

	type Status = 'loading' | 'not_configured' | 'disconnected' | 'waiting_qr' | 'connected';

	let status = $state<Status>('loading');
	let qrDataUrl = $state('');
	let abortController: AbortController | null = null;

	async function connect() {
		abortController = new AbortController();
		try {
			const res = await fetch(`${pb.baseUrl}/api/whatsapp/stream`, {
				headers: { Authorization: pb.authStore.token },
				signal: abortController.signal,
			});
			if (res.status === 503) {
				status = 'not_configured';
				return;
			}
			if (!res.body) return;
			await readSSE(res.body, async (data) => {
				const s = data as WAStatus;
				if (s.connected) {
					status = 'connected';
					qrDataUrl = '';
				} else if (s.qr) {
					status = 'waiting_qr';
					qrDataUrl = await QRCode.toDataURL(s.qr, { width: 280, margin: 2 });
				} else {
					status = 'disconnected';
					qrDataUrl = '';
				}
			});
		} catch (err) {
			if (err instanceof Error && err.name === 'AbortError') return;
			status = 'disconnected';
		}
	}

	onMount(connect);

	onDestroy(() => {
		abortController?.abort();
	});
</script>

<svelte:head>
	<title>WhatsApp — Rekan</title>
</svelte:head>

<div class="min-h-screen flex flex-col" style="background: var(--bg)">
	<header
		class="border-b px-6 py-4 flex items-center justify-between shrink-0"
		style="background: var(--surface); border-color: var(--border)"
	>
		<div class="flex items-center gap-3">
			<a href="/operador" class="text-sm" style="color: var(--text-muted)">← Operador</a>
			<span style="color: var(--border-strong)">|</span>
			<span class="font-semibold" style="color: var(--text); font-family: var(--font-primary)">
				WhatsApp
			</span>
		</div>
		{#if status !== 'loading'}
			<span
				class="text-xs px-2 py-1 rounded-full"
				style="background: {status === 'connected' ? '#DEF7EC' : '#FDE8E8'}; color: {status === 'connected' ? '#03543F' : '#9B1C1C'}"
			>
				{status === 'connected' ? 'Conectado' : 'Desconectado'}
			</span>
		{/if}
	</header>

	<main class="flex-1 flex items-center justify-center p-6">
		{#if status === 'loading'}
			<p class="text-sm" style="color: var(--text-muted)">Verificando status...</p>

		{:else if status === 'not_configured'}
			<div
				class="rounded-2xl p-8 text-center max-w-sm"
				style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
			>
				<h2 class="text-lg font-semibold mb-2" style="color: var(--text)">WhatsApp não configurado</h2>
				<p class="text-sm" style="color: var(--text-secondary)">
					O servidor foi iniciado sem suporte a WhatsApp. Verifique os logs do servidor.
				</p>
			</div>

		{:else if status === 'connected'}
			<div
				class="rounded-2xl p-8 text-center max-w-sm"
				style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
			>
				<div
					class="w-12 h-12 rounded-full flex items-center justify-center mx-auto mb-4"
					style="background: #DEF7EC"
				>
					<svg viewBox="0 0 20 20" fill="#03543F" width="24" height="24" aria-hidden="true">
						<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
					</svg>
				</div>
				<h2 class="text-lg font-semibold mb-2" style="color: var(--text)">WhatsApp conectado</h2>
				<p class="text-sm" style="color: var(--text-secondary)">
					O bot está online e recebendo mensagens. Nenhuma ação necessária.
				</p>
			</div>

		{:else if status === 'waiting_qr'}
			<div
				class="rounded-2xl p-8 text-center max-w-sm w-full"
				style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
			>
				<h2 class="text-lg font-semibold mb-1" style="color: var(--text)">Conectar WhatsApp</h2>
				<p class="text-sm mb-6" style="color: var(--text-secondary)">
					Escaneie o código com o celular do bot.
				</p>

				<div class="bg-white p-3 rounded-xl inline-block mb-4">
					{#if qrDataUrl}
						<img src={qrDataUrl} alt="QR Code WhatsApp" width="280" height="280" />
					{:else}
						<div class="flex items-center justify-center" style="width: 280px; height: 280px">
							<span class="text-sm" style="color: var(--text-muted)">Gerando código...</span>
						</div>
					{/if}
				</div>

				<ol class="text-left text-sm space-y-2 mb-4" style="color: var(--text-secondary)">
					<li>1. Abra o WhatsApp no celular do bot</li>
					<li>2. Vá em <strong>Configurações → Dispositivos conectados</strong></li>
					<li>3. Toque em <strong>Conectar dispositivo</strong></li>
					<li>4. Aponte a câmera para o código acima</li>
				</ol>

				<p class="text-xs" style="color: var(--text-muted)">
					O código atualiza automaticamente quando um novo é gerado.
				</p>
			</div>

		{:else}
			<!-- disconnected, no QR yet -->
			<div
				class="rounded-2xl p-8 text-center max-w-sm"
				style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
			>
				<h2 class="text-lg font-semibold mb-2" style="color: var(--text)">Aguardando QR code</h2>
				<p class="text-sm" style="color: var(--text-secondary)">
					O servidor está gerando o código de pareamento. Aguarde alguns segundos.
				</p>
			</div>
		{/if}
	</main>
</div>
