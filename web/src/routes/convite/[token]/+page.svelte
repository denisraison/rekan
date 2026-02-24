<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import { pb } from '$lib/pb';
	import { maskCpfCnpj, validateCpfCnpj } from '$lib/cpf-cnpj';

	type InviteData = {
		business_name: string;
		client_name: string;
		status: string;
		price_first_month: number;
		price_monthly: number;
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

	let maskedCpfCnpj = $derived(maskCpfCnpj(cpfCnpj));

	function handleCpfCnpjInput(e: Event) {
		const input = e.target as HTMLInputElement;
		cpfCnpj = input.value.replace(/\D/g, '').slice(0, 14);
		cpfCnpjError = '';
	}

	onMount(async () => {
		try {
			const res = await pb.send(`/api/invites/${token}`, { method: 'GET' });
			invite = res as InviteData;

			if (invite.status === 'accepted') {
				goto(`/convite/${token}/confirmacao`);
				return;
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
			if (res.payment_url) {
				window.location.href = res.payment_url;
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
		{:else if invite?.status === 'active'}
			<div
				class="rounded-2xl p-8 text-center"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Sua conta já está ativa!
				</h1>
				<p class="text-sm mb-4" style="color: var(--text-secondary)">
					O Rekan já está gerando conteúdo para {invite.business_name}. Qualquer dúvida, fale com a gente.
				</p>
				<!-- TODO: replace phone number -->
				<a
					href="https://wa.me/5500000000000?text=Oi,%20preciso%20de%20ajuda"
					target="_blank"
					rel="noopener"
					class="inline-block px-5 py-2.5 rounded-full text-sm font-medium"
					style="background: #25D366; color: #fff"
				>
					Falar no WhatsApp
				</a>
			</div>
		{:else if invite?.status === 'invited'}
			<div
				class="rounded-2xl p-8"
				style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
			>
				<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
					Bem-vindo ao Rekan, {invite.client_name}!
				</h1>
				<p class="text-sm mb-6" style="color: var(--text-secondary)">
					O Rekan vai cuidar do conteúdo do Instagram de <strong>{invite.business_name}</strong>.
					Primeiro mês por <strong>R$ {invite.price_first_month}</strong>, depois
					<strong>R$ {invite.price_monthly.toFixed(2).replace('.', ',')}/mês</strong>.
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

								<p class="mb-2"><strong>2. Preços e Pagamento.</strong> O primeiro mês custa R$ {invite.price_first_month},00. A partir do segundo mês, o valor é de R$ {invite.price_monthly.toFixed(2).replace('.', ',')}/mês. O pagamento é realizado via PIX. A assinatura é renovada automaticamente a cada mês.</p>

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
