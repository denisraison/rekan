<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import QRCode from 'qrcode';
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import { pb } from '$lib/pb';
	import { maskCpfCnpj, validateCpfCnpj } from '$lib/cpf-cnpj';

	type InviteData = {
		business_name: string;
		client_name: string;
		invite_status: string;
		tier: string;
		commitment: string;
		price: number;
		commitment_months: number;
		qr_payload?: string;
	};

	const tierNames: Record<string, string> = {
		basico: 'Básico',
		parceiro: 'Parceiro',
		profissional: 'Profissional',
	};

	const commitmentNames: Record<string, string> = {
		mensal: 'Mensal',
		trimestral: 'Trimestral',
	};

	let token = $derived($page.params.token);

	let loading = $state(true);
	let invite = $state<InviteData | null>(null);
	let errorState = $state<'expired' | 'not_found' | null>(null);

	let cpfCnpj = $state('');
	let cpfCnpjError = $state('');
	let termsAccepted = $state(false);
	let termsExpanded = $state(false);
	let submitting = $state(false);
	let submitError = $state('');

	let qrPayload = $state('');
	let qrDataUrl = $state('');
	let copied = $state(false);
	let timedOut = $state(false);
	let pollCount = 0;
	let pollTimer: ReturnType<typeof setInterval>;

	let maskedCpfCnpj = $derived(maskCpfCnpj(cpfCnpj));

	function handleCpfCnpjInput(e: Event) {
		const input = e.target as HTMLInputElement;
		cpfCnpj = input.value.replace(/\D/g, '').slice(0, 14);
		cpfCnpjError = '';
	}

	function formatPrice(value: number): string {
		return value.toFixed(2).replace('.', ',');
	}

	function priceDescription(): string {
		if (!invite) return '';
		const months = invite.commitment_months;
		if (months === 1) return `R$ ${formatPrice(invite.price)}/mês`;
		const monthly = invite.price / months;
		return `R$ ${formatPrice(invite.price)} (${months}x de R$ ${formatPrice(monthly)})`;
	}

	async function generateQR(payload: string) {
		if (!payload) return;
		qrPayload = payload;
		qrDataUrl = await QRCode.toDataURL(payload, { width: 256, margin: 2 });
	}

	async function copyPayload() {
		await navigator.clipboard.writeText(qrPayload);
		copied = true;
		setTimeout(() => { copied = false; }, 2000);
	}

	function startPolling() {
		pollTimer = setInterval(async () => {
			pollCount++;
			if (pollCount >= 120) {
				timedOut = true;
				clearInterval(pollTimer);
				return;
			}
			try {
				const res = await pb.send(`/api/invites/${token}`, { method: 'GET' });
				if (res.invite_status !== 'accepted') {
					invite = res as InviteData;
					clearInterval(pollTimer);
				}
			} catch {
				// ignore polling errors
			}
		}, 5000);
	}

	onMount(async () => {
		try {
			const res = await pb.send(`/api/invites/${token}`, { method: 'GET' });
			invite = res as InviteData;

			if (invite.invite_status === 'accepted' && invite.qr_payload) {
				await generateQR(invite.qr_payload);
				startPolling();
			}
		} catch (err: unknown) {
			const e = err as { status?: number };
			if (e?.status === 410) {
				errorState = 'expired';
			} else {
				errorState = 'not_found';
			}
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		clearInterval(pollTimer);
	});

	async function handleSubmit() {
		cpfCnpjError = '';
		submitError = '';

		if (!validateCpfCnpj(cpfCnpj)) {
			cpfCnpjError = 'CPF ou CNPJ inválido.';
			return;
		}
		if (!termsAccepted) {
			submitError = 'Você precisa aceitar os Termos de Uso.';
			return;
		}

		submitting = true;
		try {
			const res = await pb.send(`/api/invites/${token}/accept`, {
				method: 'POST',
				body: JSON.stringify({ cpf_cnpj: cpfCnpj }),
			});
			if (res.qr_payload) {
				await generateQR(res.qr_payload);
				if (invite) {
					invite.invite_status = 'accepted';
					invite.qr_payload = res.qr_payload;
				}
				startPolling();
			}
		} catch {
			submitError = 'Erro ao processar. Tente novamente.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Convite — Rekan</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center px-4 py-12" style="background: var(--bg)">
	<div class="w-full max-w-md">
		<div class="flex justify-center mb-8">
			<LogoCombo />
		</div>

		{#if loading}
			<div
				class="rounded-2xl p-8 text-center"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<p class="text-sm" style="color: var(--text-muted)">Carregando...</p>
			</div>
		{:else if errorState === 'expired'}
			<div
				class="rounded-2xl p-8 text-center"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Link expirado
				</h1>
				<p class="text-sm mb-4" style="color: var(--text-secondary)">
					Este convite não é mais válido. Entre em contato pelo WhatsApp para solicitar um novo.
				</p>
				<!-- TODO: replace phone number -->
				<a
					href="https://wa.me/5500000000000?text=Oi,%20meu%20convite%20expirou"
					target="_blank"
					rel="noopener"
					class="inline-block px-5 py-2.5 rounded-full text-sm font-medium"
					style="background: #25D366; color: #fff"
				>
					Falar no WhatsApp
				</a>
			</div>
		{:else if errorState === 'not_found'}
			<div
				class="rounded-2xl p-8 text-center"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Link inválido
				</h1>
				<p class="text-sm" style="color: var(--text-secondary)">
					Este link de convite não existe. Verifique o link ou entre em contato.
				</p>
			</div>
		{:else if invite?.invite_status === 'active'}
			<div
				class="rounded-2xl p-8 text-center"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<div
					class="w-12 h-12 rounded-full flex items-center justify-center mx-auto mb-4"
					style="background: #DEF7EC"
				>
					<svg viewBox="0 0 20 20" fill="#03543F" width="24" height="24">
						<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
					</svg>
				</div>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Tudo certo{invite.client_name ? `, ${invite.client_name}` : ''}!
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
			</div>
		{:else if invite?.invite_status === 'accepted' && qrPayload}
			<div
				class="rounded-2xl p-8"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				{#if timedOut}
					<div class="text-center">
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
					</div>
				{:else}
					<div class="text-center">
						<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
							Escaneie o QR Code para pagar
						</h1>
						<p class="text-sm mb-4" style="color: var(--text-secondary)">
							Abra o app do seu banco, escolha pagar com Pix e escaneie o código abaixo.
						</p>

						{#if qrDataUrl}
							<img src={qrDataUrl} alt="QR Code Pix" class="mx-auto mb-4 rounded-lg" width="256" height="256" />
						{/if}

						<p class="text-xs mb-2" style="color: var(--text-muted)">Ou copie o código Pix:</p>
						<div class="flex gap-2 mb-4">
							<input
								readonly
								value={qrPayload}
								class="flex-1 px-3 py-2 rounded-lg text-xs truncate"
								style="background: var(--bg); border: 1px solid var(--border); color: var(--text)"
							/>
							<button
								onclick={copyPayload}
								class="px-3 py-2 rounded-lg text-xs font-medium whitespace-nowrap"
								style="background: var(--coral); color: #fff"
							>
								{copied ? 'Copiado!' : 'Copiar'}
							</button>
						</div>

						<div class="flex items-center justify-center gap-2">
							<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none" style="color: var(--coral)">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z" />
							</svg>
							<p class="text-xs" style="color: var(--text-muted)">
								Aguardando confirmação do pagamento...
							</p>
						</div>
					</div>
				{/if}
			</div>
		{:else if invite?.invite_status === 'invited'}
			<div
				class="rounded-2xl p-8"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Bem-vindo ao Rekan, {invite.client_name}!
				</h1>
				<p class="text-sm mb-1" style="color: var(--text-secondary)">
					O Rekan vai cuidar do conteúdo do Instagram de <strong>{invite.business_name}</strong>.
				</p>
				<p class="text-sm mb-1" style="color: var(--text-secondary)">
					Plano <strong>{tierNames[invite.tier] ?? invite.tier}</strong> ({commitmentNames[invite.commitment] ?? invite.commitment}): <strong>{priceDescription()}</strong>.
				</p>
				<p class="text-sm mb-6" style="color: var(--text-secondary)">
					Se em 30 dias você não sentir a diferença, devolvemos tudo pelo Pix.
				</p>

				{#if submitError}
					<p class="text-sm mb-4 p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">
						{submitError}
					</p>
				{/if}

				<div class="flex flex-col gap-4">
					<label class="flex flex-col gap-1.5">
						<span class="text-sm font-medium" style="color: var(--text)">CPF ou CNPJ</span>
						<input
							value={maskedCpfCnpj}
							oninput={handleCpfCnpjInput}
							placeholder="000.000.000-00"
							class="px-3 py-2.5 rounded-xl text-sm outline-none border"
							style="border-color: {cpfCnpjError ? '#DC2626' : 'var(--border-strong)'}; background: var(--surface); color: var(--text); font-family: var(--font-primary)"
						/>
						{#if cpfCnpjError}
							<span class="text-xs" style="color: #DC2626">{cpfCnpjError}</span>
						{/if}
					</label>

					<div>
						<button
							onclick={() => { termsExpanded = !termsExpanded; }}
							class="text-sm font-medium flex items-center gap-1"
							style="color: var(--coral)"
						>
							<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16" class="transition-transform" style="transform: rotate({termsExpanded ? '90deg' : '0deg'})">
								<path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd" />
							</svg>
							Termos de Uso
						</button>

						{#if termsExpanded}
							<div
								class="mt-2 p-4 rounded-xl text-xs leading-relaxed overflow-y-auto"
								style="background: var(--bg); border: 1px solid var(--border); max-height: 300px; color: var(--text-secondary)"
							>
								<p class="font-semibold mb-2" style="color: var(--text)">Termos de Uso do Serviço Rekan</p>

								<p class="mb-2"><strong>1. Descrição do Serviço.</strong> O Rekan é um serviço de geração de conteúdo para Instagram destinado a micro-empreendedores brasileiros. O serviço inclui a criação de legendas, hashtags e textos para stories personalizados para o seu negócio.</p>

								<p class="mb-2"><strong>2. Preços e Pagamento.</strong> O valor do plano {tierNames[invite.tier] ?? invite.tier} é {priceDescription()}. O pagamento é realizado via Pix Automático. A assinatura é renovada automaticamente a cada período. Você tem uma garantia de 30 dias: se não estiver satisfeito, devolvemos o valor integral via Pix.</p>

								<p class="mb-2"><strong>3. Cancelamento.</strong> Conforme o Art. 49 do Código de Defesa do Consumidor, você pode cancelar o serviço em até 7 dias após a contratação, com reembolso integral. Após esse prazo, o cancelamento pode ser solicitado a qualquer momento e será efetivado ao final do período já pago. Não há multa por cancelamento.</p>

								<p class="mb-2"><strong>4. Proteção de Dados (LGPD).</strong> O Rekan é o controlador dos seus dados pessoais. Coletamos nome, email e dados do negócio para a prestação do serviço contratado. O CPF/CNPJ informado neste formulário é enviado diretamente ao processador de pagamentos (Asaas) e não é armazenado pelo Rekan. Seus dados não são compartilhados com terceiros para fins de marketing. Você pode solicitar a exclusão dos seus dados a qualquer momento entrando em contato conosco.</p>

								<p class="mb-2"><strong>5. Uso do Conteúdo.</strong> Todo conteúdo gerado pelo Rekan é de sua propriedade e pode ser usado livremente no Instagram e em outras plataformas do seu negócio. O Rekan pode utilizar exemplos anonimizados para demonstração do serviço.</p>

								<p><strong>6. Contato.</strong> Para dúvidas, cancelamentos ou solicitações relacionadas aos seus dados, entre em contato pelo WhatsApp.</p>
							</div>
						{/if}
					</div>

					<label class="flex items-start gap-2 cursor-pointer">
						<input
							type="checkbox"
							bind:checked={termsAccepted}
							class="mt-0.5 rounded"
							style="accent-color: var(--coral)"
						/>
						<span class="text-sm" style="color: var(--text-secondary)">
							Li e aceito os Termos de Uso
						</span>
					</label>

					<button
						onclick={handleSubmit}
						disabled={submitting}
						class="w-full py-3 rounded-xl font-medium text-sm transition-opacity disabled:opacity-60"
						style="background: var(--coral); color: #fff; font-family: var(--font-primary)"
					>
						{submitting ? 'Processando...' : 'Aceitar e pagar via PIX'}
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>
