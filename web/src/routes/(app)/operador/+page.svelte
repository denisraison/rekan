<script lang="ts">
	import { pb } from '$lib/pb';
	import { onMount, onDestroy } from 'svelte';
	import QRCode from 'qrcode';
	import type { Business, Service, GeneratedPost, Message, Post } from '$lib/types';

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

	// WhatsApp status
	let waConnected = $state(false);
	let waQR = $state('');
	let waChecking = $state(true);

	// Messages
	let messages = $state<Message[]>([]);
	let messagesLoading = $state(false);
	let unsubscribeMessages: (() => void) | null = null;

	// Client form
	let showForm = $state(false);
	let editingId = $state<string | null>(null);
	let formName = $state('');
	let formType = $state('');
	let formCity = $state('');
	let formState = $state('');
	let formPhone = $state('');
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
	let sending = $state(false);
	let sendError = $state('');

	// Posts (for health indicators)
	let posts = $state<Post[]>([]);

	let selected = $derived(clients.find((c) => c.id === selectedId) ?? null);

	// Unread message counts per business
	let lastSeen = $state<Record<string, string>>({});
	let unreadCounts = $derived.by(() => {
		const counts: Record<string, number> = {};
		for (const client of clients) {
			const seen = lastSeen[client.id];
			if (!seen) {
				counts[client.id] = messages.filter(
					(m) => m.business === client.id && m.direction === 'incoming'
				).length;
			} else {
				counts[client.id] = messages.filter(
					(m) => m.business === client.id && m.direction === 'incoming' && m.created > seen
				).length;
			}
		}
		return counts;
	});

	// Messages for selected client
	let threadMessages = $derived(
		selectedId
			? messages
					.filter((m) => m.business === selectedId)
					.sort((a, b) => a.created.localeCompare(b.created))
			: []
	);

	// Latest incoming message for "Gerar post" pre-fill
	let latestIncoming = $derived.by(() => {
		const incoming = threadMessages.filter(
			(m) => m.direction === 'incoming' && m.content
		);
		return incoming.length > 0 ? incoming[incoming.length - 1] : null;
	});
	let latestIncomingText = $derived(latestIncoming?.content ?? '');

	// Health indicators per client
	type ClientHealth = { daysSinceMsg: number; postsThisMonth: number; color: string };
	let clientHealth = $derived.by(() => {
		const now = Date.now();
		const monthStart = new Date();
		monthStart.setDate(1);
		monthStart.setHours(0, 0, 0, 0);
		const monthStr = monthStart.toISOString();

		const health: Record<string, ClientHealth> = {};
		for (const client of clients) {
			const clientMsgs = messages.filter(
				(m) => m.business === client.id && m.direction === 'incoming'
			);
			const lastMsg = clientMsgs.length > 0
				? clientMsgs.reduce((a, b) => a.created > b.created ? a : b)
				: null;
			const daysSinceMsg = lastMsg
				? Math.floor((now - new Date(lastMsg.wa_timestamp || lastMsg.created).getTime()) / 86400000)
				: 999;

			const postsThisMonth = posts.filter(
				(p) => p.business === client.id && p.created >= monthStr
			).length;

			let color = '#10B981'; // green
			if (daysSinceMsg >= 10) color = '#EF4444'; // red
			else if (daysSinceMsg >= 5) color = '#F59E0B'; // yellow

			health[client.id] = { daysSinceMsg, postsThisMonth, color };
		}
		return health;
	});

	// Sort clients by urgency (red first, then yellow, then green)
	let sortedClients = $derived(
		[...clients].sort((a, b) => {
			const ha = clientHealth[a.id];
			const hb = clientHealth[b.id];
			if (!ha || !hb) return 0;
			return hb.daysSinceMsg - ha.daysSinceMsg;
		})
	);

	onMount(async () => {
		// Load clients and check WhatsApp in parallel
		const [clientsRes] = await Promise.all([
			pb.collection('businesses').getList<Business>(1, 200, { sort: 'name' }),
			checkWhatsAppStatus()
		]);
		clients = clientsRes.items;
		loading = false;

		// Load all messages and posts
		await Promise.all([loadMessages(), loadPosts()]);

		// Subscribe to realtime message updates
		unsubscribeMessages = await pb.collection('messages').subscribe<Message>('*', (e) => {
			if (e.action === 'create') {
				messages = [...messages, e.record];
			} else if (e.action === 'update') {
				messages = messages.map((m) => (m.id === e.record.id ? e.record : m));
			}
		});

		// Poll WhatsApp status every 5s
		waInterval = setInterval(checkWhatsAppStatus, 5000);
	});

	let waInterval: ReturnType<typeof setInterval>;
	let qrDataUrl = $state('');

	$effect(() => {
		if (waQR) {
			QRCode.toDataURL(waQR, { width: 256, margin: 2 }).then((url: string) => {
				qrDataUrl = url;
			});
		} else {
			qrDataUrl = '';
		}
	});

	onDestroy(() => {
		unsubscribeMessages?.();
		clearInterval(waInterval);
	});

	async function checkWhatsAppStatus() {
		try {
			const res = await pb.send('/api/whatsapp/status', { method: 'GET' });
			waConnected = res.connected;
			waQR = res.qr || '';
		} catch {
			waConnected = false;
			waQR = '';
		}
		waChecking = false;
	}

	async function loadMessages() {
		messagesLoading = true;
		try {
			const res = await pb.collection('messages').getList<Message>(1, 500, {
				sort: 'created'
			});
			messages = res.items;
		} finally {
			messagesLoading = false;
		}
	}

	async function loadPosts() {
		try {
			const res = await pb.collection('posts').getList<Post>(1, 500, {
				sort: '-created'
			});
			posts = res.items;
		} catch {
			// Posts loading is non-critical for the page to work
		}
	}

	function selectClient(id: string) {
		selectedId = id;
		result = null;
		generateError = '';
		// Mark as seen
		lastSeen = { ...lastSeen, [id]: new Date().toISOString() };
	}

	function prefillGenerate() {
		message = latestIncomingText;
	}

	function mediaUrl(msg: Message): string {
		return pb.files.getURL({ id: msg.id, collectionId: msg.collectionId } as any, msg.media);
	}

	// --- Client form logic (unchanged) ---

	function resetForm() {
		formName = '';
		formType = '';
		formCity = '';
		formState = '';
		formPhone = '';
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
		formPhone = biz.phone || '';
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
			phone: formPhone.trim(),
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
			const payload: Record<string, string> = { message: message.trim() };
			if (latestIncoming && message.trim() === latestIncoming.content) {
				payload.message_id = latestIncoming.id;
			}
			const res = await pb.send(`/api/businesses/${selectedId}/posts:generateFromMessage`, {
				method: 'POST',
				body: JSON.stringify(payload)
			});
			result = res as GeneratedPost;
		} catch (err: unknown) {
			const e = err as { data?: { message?: string } };
			generateError = e?.data?.message ?? 'Erro ao gerar conteúdo. Tente novamente.';
		} finally {
			generating = false;
		}
	}

	async function sendViaWhatsApp() {
		if (!selectedId || !result) return;
		sending = true;
		sendError = '';
		try {
			await pb.send('/api/messages:send', {
				method: 'POST',
				body: JSON.stringify({
					business_id: selectedId,
					caption: result.caption,
					hashtags: result.hashtags.join(' '),
					production_note: result.production_note || ''
				})
			});
			result = null;
			message = '';
		} catch {
			sendError = 'Erro ao enviar. Tente novamente.';
		} finally {
			sending = false;
		}
	}

	async function copyText(text: string, label: string) {
		await navigator.clipboard.writeText(text);
		copied = label;
		setTimeout(() => { copied = null; }, 2000);
	}
</script>

<div class="min-h-screen flex flex-col" style="background: var(--bg)">
	<header
		class="border-b px-6 py-4 flex items-center justify-between shrink-0"
		style="background: var(--surface); border-color: var(--border)"
	>
		<span class="font-semibold" style="color: var(--text); font-family: var(--font-primary)">
			Rekan — Operador
		</span>
		<span class="text-xs px-2 py-1 rounded-full" style="background: {waConnected ? '#DEF7EC' : '#FDE8E8'}; color: {waConnected ? '#03543F' : '#9B1C1C'}">
			WhatsApp {waConnected ? 'conectado' : 'desconectado'}
		</span>
	</header>

	{#if loading}
		<p class="text-sm p-6" style="color: var(--text-muted)">Carregando...</p>
	{:else if !waConnected && waQR}
		<!-- QR Code pairing screen -->
		<main class="flex-1 flex items-center justify-center p-6">
			<div class="rounded-2xl p-8 text-center max-w-sm" style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)">
				<h2 class="text-lg font-semibold mb-2" style="color: var(--text)">Conectar WhatsApp</h2>
				<p class="text-sm mb-6" style="color: var(--text-secondary)">
					Escaneie o QR code com o WhatsApp Business do Rekan.
				</p>
				<div class="bg-white p-4 rounded-xl inline-block">
					{#if qrDataUrl}
						<img src={qrDataUrl} alt="QR Code WhatsApp" width="256" height="256" />
					{:else}
						<div style="width: 256px; height: 256px" class="flex items-center justify-center">
							<span class="text-sm" style="color: var(--text-muted)">Carregando...</span>
						</div>
					{/if}
				</div>
				<p class="text-xs mt-4" style="color: var(--text-muted)">O QR code atualiza automaticamente.</p>
			</div>
		</main>
	{:else}
		<!-- Main operator layout -->
		<main class="flex-1 flex overflow-hidden">
			<!-- Left: Client list -->
			<div class="w-72 border-r flex flex-col shrink-0" style="border-color: var(--border); background: var(--surface)">
				<div class="flex items-center justify-between p-4 border-b" style="border-color: var(--border)">
					<h2 class="text-sm font-semibold" style="color: var(--text)">Clientes</h2>
					<button
						onclick={openNewForm}
						class="text-xs font-medium px-2.5 py-1 rounded-full"
						style="background: var(--coral); color: #fff"
					>
						Novo
					</button>
				</div>

				<div class="flex-1 overflow-y-auto">
					{#if clients.length === 0}
						<p class="text-sm p-4" style="color: var(--text-muted)">Nenhum cliente cadastrado.</p>
					{:else}
						{#each sortedClients as client (client.id)}
							{@const unread = unreadCounts[client.id] || 0}
							{@const health = clientHealth[client.id]}
							<button
								onclick={() => selectClient(client.id)}
								class="w-full text-left px-4 py-3 border-b transition-colors"
								style="background: {selectedId === client.id ? 'var(--coral-pale)' : 'transparent'}; border-color: var(--border); color: var(--text)"
							>
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-2 min-w-0">
										{#if health}
											<span class="w-2 h-2 rounded-full shrink-0" style="background: {health.color}"></span>
										{/if}
										<span class="font-medium text-sm truncate">{client.name}</span>
									</div>
									{#if unread > 0}
										<span class="text-xs font-bold px-1.5 py-0.5 rounded-full shrink-0 ml-2" style="background: var(--coral); color: #fff">
											{unread}
										</span>
									{/if}
								</div>
								<div class="flex items-center justify-between mt-0.5">
									<span class="text-xs" style="color: var(--text-muted)">{client.type} — {client.city}/{client.state}</span>
									{#if health}
										<span class="text-xs" style="color: var(--text-muted)">
											{health.daysSinceMsg < 999 ? `${health.daysSinceMsg}d` : ''}{health.postsThisMonth > 0 ? ` · ${health.postsThisMonth} posts` : ''}
										</span>
									{/if}
								</div>
							</button>
						{/each}
					{/if}
				</div>
			</div>

			<!-- Right: Thread or form -->
			<div class="flex-1 flex flex-col overflow-hidden">
				{#if showForm}
					<div class="flex-1 overflow-y-auto p-6">
						<div class="max-w-xl rounded-2xl p-6" style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)">
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

								<label class="flex flex-col gap-1.5">
									<span class="text-sm font-medium" style="color: var(--text)">Telefone WhatsApp</span>
									<input bind:value={formPhone} placeholder="5511999998888" class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
								</label>

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
									<input bind:value={formBrandVibe} placeholder="Ex: Descontraido e moderno" class="px-3 py-2.5 rounded-xl text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
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
					</div>
				{:else if selected}
					<!-- Client header -->
					<div class="px-6 py-4 border-b flex items-center justify-between shrink-0" style="border-color: var(--border); background: var(--surface)">
						<div>
							<h2 class="text-sm font-semibold" style="color: var(--text)">{selected.name}</h2>
							<p class="text-xs" style="color: var(--text-secondary)">{selected.type} — {selected.city}/{selected.state}</p>
						</div>
						<button onclick={() => openEditForm(selected!)} class="text-xs px-3 py-1.5 rounded-full border" style="border-color: var(--border-strong); color: var(--text-secondary)">Editar</button>
					</div>

					<!-- Message thread -->
					<div class="flex-1 overflow-y-auto px-6 py-4 flex flex-col gap-3">
						{#if threadMessages.length === 0}
							<p class="text-sm text-center py-8" style="color: var(--text-muted)">Nenhuma mensagem ainda.</p>
						{:else}
							{#each threadMessages as msg (msg.id)}
								<div class="flex {msg.direction === 'outgoing' ? 'justify-end' : 'justify-start'}">
									<div
										class="max-w-md rounded-2xl px-4 py-2.5 text-sm"
										style="background: {msg.direction === 'outgoing' ? 'var(--coral-pale)' : 'var(--surface)'}; border: 1px solid {msg.direction === 'outgoing' ? 'var(--coral-light)' : 'var(--border)'}; color: var(--text)"
									>
										{#if msg.type === 'audio'}
											<span class="text-xs font-medium block mb-1" style="color: var(--text-muted)">Áudio transcrito</span>
										{/if}

										{#if msg.type === 'image' && msg.media}
											<img
												src={mediaUrl(msg)}
												alt="Imagem do cliente"
												class="rounded-xl mb-2 max-w-full"
												style="max-height: 300px"
											/>
										{/if}

										{#if msg.content}
											<p class="whitespace-pre-wrap">{msg.content}</p>
										{:else if msg.type === 'audio'}
											<p class="italic" style="color: var(--text-muted)">Transcrição indisponível</p>
										{/if}

										<span class="text-xs block mt-1" style="color: var(--text-muted)">
											{new Date(msg.wa_timestamp || msg.created).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}
										</span>
									</div>
								</div>
							{/each}
						{/if}
					</div>

					<!-- Generate panel (bottom) -->
					<div class="shrink-0 border-t px-6 py-4" style="border-color: var(--border); background: var(--surface)">
						<div class="flex gap-2 items-end">
							<div class="flex-1">
								<textarea
									bind:value={message}
									placeholder="Mensagem do cliente para gerar post..."
									rows={2}
									class="w-full px-3 py-2.5 rounded-xl text-sm outline-none border resize-none"
									style="border-color: var(--border-strong); background: var(--bg); color: var(--text)"
								></textarea>
							</div>
							<div class="flex flex-col gap-1.5">
								{#if latestIncomingText && message !== latestIncomingText}
									<button
										onclick={prefillGenerate}
										class="text-xs px-3 py-1.5 rounded-full border"
										style="border-color: var(--border-strong); color: var(--text-secondary)"
									>
										Usar última msg
									</button>
								{/if}
								<button
									onclick={generate}
									disabled={generating || !message.trim()}
									class="px-4 py-2.5 rounded-full text-sm font-medium transition-opacity"
									style="background: var(--coral); color: #fff; opacity: {generating || !message.trim() ? '0.6' : '1'}; cursor: {generating || !message.trim() ? 'not-allowed' : 'pointer'}"
								>
									{generating ? 'Gerando...' : 'Gerar post'}
								</button>
							</div>
						</div>

						{#if generateError}
							<p class="mt-2 text-sm" style="color: var(--destructive)">{generateError}</p>
						{/if}

						{#if result}
							<div class="mt-4 rounded-xl p-4" style="background: var(--bg); border: 1px solid var(--border)">
								<div class="mb-3">
									<div class="flex items-center justify-between mb-1">
										<span class="text-xs font-medium uppercase tracking-widest" style="color: var(--text-muted)">Legenda</span>
										<button onclick={() => copyText(result!.caption, 'caption')} class="text-xs" style="color: var(--coral)">
											{copied === 'caption' ? 'Copiado!' : 'Copiar'}
										</button>
									</div>
									<p class="text-sm leading-relaxed whitespace-pre-wrap" style="color: var(--text)">{result.caption}</p>
								</div>

								<div class="mb-3">
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

								{#if waConnected && selected?.phone}
									<div class="flex items-center gap-2 mt-4 pt-3 border-t" style="border-color: var(--border)">
										<button
											onclick={sendViaWhatsApp}
											disabled={sending}
											class="px-4 py-2 rounded-full text-sm font-medium transition-opacity"
											style="background: #25D366; color: #fff; opacity: {sending ? '0.6' : '1'}; cursor: {sending ? 'not-allowed' : 'pointer'}"
										>
											{sending ? 'Enviando...' : 'Enviar pelo WhatsApp'}
										</button>
										{#if sendError}
											<span class="text-xs" style="color: var(--destructive)">{sendError}</span>
										{/if}
									</div>
								{/if}
							</div>
						{/if}
					</div>
				{:else}
					<div class="flex-1 flex items-center justify-center">
						<p class="text-sm" style="color: var(--text-muted)">Selecione um cliente para ver as mensagens.</p>
					</div>
				{/if}
			</div>
		</main>
	{/if}
</div>
