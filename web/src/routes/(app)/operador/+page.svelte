<script lang="ts">
	import { pb } from '$lib/pb';
	import { onMount } from 'svelte';
	import type { Business, Service, GeneratedPost } from '$lib/types';

	const BUSINESS_TYPES = [
		'Salão de Beleza',
		'Restaurante',
		'Personal Trainer',
		'Nail Designer',
		'Confeitaria',
		'Barbearia',
		'Loja de Roupas',
		'Pet Shop',
		'Banda Musical',
		'Estúdio de Tatuagem',
		'Hamburgueria',
		'Loja de Açaí',
		'Outro'
	];

	const STATES = [
		'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO',
		'MA', 'MT', 'MS', 'MG', 'PA', 'PB', 'PR', 'PE', 'PI',
		'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
	];

	let clients = $state<Business[]>([]);
	let selectedId = $state<string | null>(null);
	let loading = $state(true);

	// Client form
	let showForm = $state(false);
	let editingId = $state<string | null>(null);
	let formName = $state('');
	let formType = $state('');
	let formCity = $state('');
	let formState = $state('');
	let formServices: Service[] = $state([{ name: '', price_brl: 0 }]);
	let formTargetAudience = $state('');
	let formBrandVibe = $state('');
	let formQuirks = $state('');
	let formError = $state('');
	let formSaving = $state(false);

	// Generation
	let message = $state('');
	let generating = $state(false);
	let generateError = $state('');
	let result = $state<GeneratedPost | null>(null);
	let copied = $state<string | null>(null);

	let selected = $derived(clients.find((c) => c.id === selectedId) ?? null);

	onMount(async () => {
		try {
			const res = await pb.collection('businesses').getList<Business>(1, 200, { sort: 'name' });
			clients = res.items;
		} finally {
			loading = false;
		}
	});

	function resetForm() {
		formName = '';
		formType = '';
		formCity = '';
		formState = '';
		formServices = [{ name: '', price_brl: 0 }];
		formTargetAudience = '';
		formBrandVibe = '';
		formQuirks = '';
		formError = '';
		editingId = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(biz: Business) {
		editingId = biz.id;
		formName = biz.name;
		formType = biz.type;
		formCity = biz.city;
		formState = biz.state;
		formServices = biz.services.length > 0 ? [...biz.services] : [{ name: '', price_brl: 0 }];
		formTargetAudience = biz.target_audience || '';
		formBrandVibe = biz.brand_vibe || '';
		formQuirks = biz.quirks || '';
		formError = '';
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	function addService() {
		formServices = [...formServices, { name: '', price_brl: 0 }];
	}

	function removeService(i: number) {
		formServices = formServices.filter((_: Service, idx: number) => idx !== i);
	}

	async function saveClient() {
		if (!formName.trim() || !formType || !formCity.trim() || !formState) {
			formError = 'Preencha nome, tipo, cidade e estado.';
			return;
		}
		const validServices = formServices.filter((s: Service) => s.name.trim());
		if (validServices.length === 0) {
			formError = 'Adicione pelo menos um serviço.';
			return;
		}

		formError = '';
		formSaving = true;
		const data = {
			user: pb.authStore.record!.id,
			name: formName.trim(),
			type: formType,
			city: formCity.trim(),
			state: formState,
			services: validServices,
			target_audience: formTargetAudience.trim(),
			brand_vibe: formBrandVibe.trim(),
			quirks: formQuirks.trim(),
			onboarding_step: 3
		};

		try {
			if (editingId) {
				const { user: _, ...updateData } = data;
				const updated = await pb.collection('businesses').update<Business>(editingId, updateData);
				clients = clients.map((c) => (c.id === editingId ? updated : c));
			} else {
				const created = await pb.collection('businesses').create<Business>(data);
				clients = [...clients, created].sort((a, b) => a.name.localeCompare(b.name));
				selectedId = created.id;
			}
			closeForm();
		} catch {
			formError = 'Erro ao salvar. Tente novamente.';
		} finally {
			formSaving = false;
		}
	}

	async function generate() {
		if (!selectedId || !message.trim()) return;
		generating = true;
		generateError = '';
		result = null;
		try {
			const res = await pb.send(`/api/businesses/${selectedId}/posts:generateFromMessage`, {
				method: 'POST',
				body: JSON.stringify({ message: message.trim() })
			});
			result = res as GeneratedPost;
		} catch (err: unknown) {
			const e = err as { data?: { message?: string } };
			generateError = e?.data?.message ?? 'Erro ao gerar conteúdo. Tente novamente.';
		} finally {
			generating = false;
		}
	}

	async function copyText(text: string, label: string) {
		await navigator.clipboard.writeText(text);
		copied = label;
		setTimeout(() => { copied = null; }, 2000);
	}
</script>

<div class="min-h-screen" style="background: var(--bg)">
	<header
		class="border-b px-6 py-4 flex items-center justify-between"
		style="background: var(--surface); border-color: var(--border)"
	>
		<span class="font-semibold" style="color: var(--text); font-family: var(--font-primary)">
			Rekan — Operador
		</span>
	</header>

	<main class="max-w-5xl mx-auto px-6 py-8">
		{#if loading}
			<p class="text-sm" style="color: var(--text-muted)">Carregando...</p>
		{:else}
			<div class="grid gap-8" style="grid-template-columns: 300px 1fr">
				<!-- Left: Client list -->
				<div>
					<div class="flex items-center justify-between mb-4">
						<h2 class="text-lg font-semibold" style="color: var(--text)">Clientes</h2>
						<button
							onclick={openNewForm}
							class="text-sm font-medium px-3 py-1.5 rounded-full"
							style="background: var(--coral); color: #fff; font-family: var(--font-primary)"
						>
							Adicionar
						</button>
					</div>

					{#if clients.length === 0}
						<p class="text-sm" style="color: var(--text-muted)">Nenhum cliente cadastrado.</p>
					{:else}
						<div class="flex flex-col gap-1">
							{#each clients as client (client.id)}
								<button
									onclick={() => { selectedId = client.id; result = null; generateError = ''; }}
									class="text-left px-4 py-3 rounded-xl transition-colors text-sm"
									style="background: {selectedId === client.id ? 'var(--coral-pale)' : 'var(--surface)'}; border: 1px solid {selectedId === client.id ? 'var(--coral-light)' : 'var(--border)'}; color: var(--text); font-family: var(--font-primary)"
								>
									<span class="font-medium">{client.name}</span>
									<span class="block text-xs" style="color: var(--text-muted)">{client.type} — {client.city}/{client.state}</span>
								</button>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Right: Generate or form -->
				<div>
					{#if showForm}
						<div class="rounded-2xl p-6" style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)">
							<h2 class="text-lg font-semibold mb-4" style="color: var(--text)">
								{editingId ? 'Editar cliente' : 'Novo cliente'}
							</h2>

							{#if formError}
								<p class="text-sm mb-4 p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">{formError}</p>
							{/if}

							<div class="flex flex-col gap-4">
								<label class="flex flex-col gap-1.5">
									<span class="text-sm font-medium" style="color: var(--text)">Nome</span>
									<input bind:value={formName} placeholder="Nome do negócio" class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
								</label>

								<label class="flex flex-col gap-1.5">
									<span class="text-sm font-medium" style="color: var(--text)">Tipo de negócio</span>
									<select bind:value={formType} class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)">
										<option value="">Selecione...</option>
										{#each BUSINESS_TYPES as t}
											<option value={t}>{t}</option>
										{/each}
									</select>
								</label>

								<div class="flex gap-3">
									<label class="flex flex-col gap-1.5 flex-1">
										<span class="text-sm font-medium" style="color: var(--text)">Cidade</span>
										<input bind:value={formCity} placeholder="Ex: São Paulo" class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
									</label>
									<label class="flex flex-col gap-1.5 w-24">
										<span class="text-sm font-medium" style="color: var(--text)">Estado</span>
										<select bind:value={formState} class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)">
											<option value="">UF</option>
											{#each STATES as stateCode}
												<option value={stateCode}>{stateCode}</option>
											{/each}
										</select>
									</label>
								</div>

								<div>
									<span class="text-sm font-medium" style="color: var(--text)">Serviços</span>
									<div class="flex flex-col gap-2 mt-1.5">
										{#each formServices as service, i}
											<div class="flex gap-2 items-center">
												<input bind:value={service.name} placeholder="Nome do serviço" class="flex-1 px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
												<div class="relative w-28">
													<span class="absolute left-3 top-1/2 -translate-y-1/2 text-sm" style="color: var(--text-muted)">R$</span>
													<input type="number" bind:value={service.price_brl} min="0" class="w-full pl-9 pr-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
												</div>
												{#if formServices.length > 1}
													<button onclick={() => removeService(i)} class="p-1" style="color: var(--text-muted)" aria-label="Remover">
														<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
													</button>
												{/if}
											</div>
										{/each}
										<button onclick={addService} class="text-sm font-medium mt-1" style="color: var(--primary)">+ Adicionar serviço</button>
									</div>
								</div>

								<label class="flex flex-col gap-1.5">
									<span class="text-sm font-medium" style="color: var(--text)">Público-alvo <span class="font-normal" style="color: var(--text-muted)">— opcional</span></span>
									<input bind:value={formTargetAudience} placeholder="Ex: Mulheres de 25 a 40 anos" class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
								</label>

								<label class="flex flex-col gap-1.5">
									<span class="text-sm font-medium" style="color: var(--text)">Estilo da marca <span class="font-normal" style="color: var(--text-muted)">— opcional</span></span>
									<input bind:value={formBrandVibe} placeholder="Ex: Descontraído e moderno" class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
								</label>

								<label class="flex flex-col gap-1.5">
									<span class="text-sm font-medium" style="color: var(--text)">Diferenciais <span class="font-normal" style="color: var(--text-muted)">— opcional</span></span>
									<textarea bind:value={formQuirks} placeholder="Ex: Atendimento por WhatsApp, parcela em 3x" rows={2} class="px-3 py-2.5 rounded-xl text-sm outline-none border resize-none" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"></textarea>
								</label>
							</div>

							<div class="flex gap-3 mt-6">
								<button onclick={closeForm} class="px-5 py-2.5 rounded-full text-sm font-medium border" style="border-color: var(--border-strong); color: var(--text-secondary)">Cancelar</button>
								<button onclick={saveClient} disabled={formSaving} class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity" style="background: var(--coral); color: #fff; opacity: {formSaving ? '0.6' : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}">
									{formSaving ? 'Salvando...' : 'Salvar'}
								</button>
							</div>
						</div>
					{:else if selected}
						<div class="mb-4 flex items-center justify-between">
							<div>
								<h2 class="text-lg font-semibold" style="color: var(--text)">{selected.name}</h2>
								<p class="text-sm" style="color: var(--text-secondary)">{selected.type} — {selected.city}/{selected.state}</p>
							</div>
							<button onclick={() => openEditForm(selected!)} class="text-sm" style="color: var(--text-secondary)">Editar</button>
						</div>

						<div class="rounded-2xl p-6 mb-6" style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)">
							<label class="flex flex-col gap-1.5 mb-4">
								<span class="text-sm font-medium" style="color: var(--text)">Mensagem do cliente</span>
								<textarea
									bind:value={message}
									placeholder="Cole a mensagem que o cliente mandou no WhatsApp..."
									rows={4}
									class="px-3 py-2.5 rounded-xl text-sm outline-none border resize-none"
									style="border-color: var(--border-strong); background: var(--bg); color: var(--text)"
								></textarea>
							</label>

							<button
								onclick={generate}
								disabled={generating || !message.trim()}
								class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity"
								style="background: var(--coral); color: #fff; opacity: {generating || !message.trim() ? '0.6' : '1'}; cursor: {generating || !message.trim() ? 'not-allowed' : 'pointer'}"
							>
								{generating ? 'Gerando...' : 'Gerar post'}
							</button>

							{#if generateError}
								<p class="mt-3 text-sm" style="color: var(--destructive)">{generateError}</p>
							{/if}
						</div>

						{#if result}
							<div class="rounded-2xl p-6" style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)">
								<h3 class="text-sm font-semibold mb-4" style="color: var(--text)">Post gerado</h3>

								<div class="mb-4">
									<div class="flex items-center justify-between mb-1">
										<span class="text-xs font-medium uppercase tracking-widest" style="color: var(--text-muted)">Legenda</span>
										<button onclick={() => copyText(result!.caption, 'caption')} class="text-xs" style="color: var(--coral)">
											{copied === 'caption' ? 'Copiado!' : 'Copiar'}
										</button>
									</div>
									<p class="text-sm leading-relaxed whitespace-pre-wrap" style="color: var(--text)">{result.caption}</p>
								</div>

								<div class="mb-4">
									<div class="flex items-center justify-between mb-1">
										<span class="text-xs font-medium uppercase tracking-widest" style="color: var(--text-muted)">Hashtags</span>
										<button onclick={() => copyText(result!.hashtags.join(' '), 'hashtags')} class="text-xs" style="color: var(--coral)">
											{copied === 'hashtags' ? 'Copiado!' : 'Copiar'}
										</button>
									</div>
									<p class="text-xs" style="color: var(--text-secondary)">{result.hashtags.join(' ')}</p>
								</div>

								{#if result.production_note}
									<div>
										<span class="text-xs font-medium uppercase tracking-widest" style="color: var(--text-muted)">Nota de produção</span>
										<p class="text-xs italic mt-1" style="color: var(--text-secondary); border-left: 2px solid var(--border-strong); padding-left: 0.75rem">{result.production_note}</p>
									</div>
								{/if}
							</div>
						{/if}
					{:else}
						<div class="rounded-2xl p-8 text-center" style="background: var(--surface); border: 1px solid var(--border)">
							<p class="text-sm" style="color: var(--text-muted)">Selecione um cliente ou adicione um novo para começar.</p>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</main>
</div>
