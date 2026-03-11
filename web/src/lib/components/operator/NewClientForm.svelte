<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Textarea } from "$lib/components/ui/textarea";
  import * as api from "$lib/operator/api";
  import { businessTypes, states } from "$lib/operator/constants";
  import { fmtTime } from "$lib/operator/format";
  import { pb } from "$lib/pb";
  import type { Business, Service } from "$lib/types";

  type Props = {
    editingId: string | null;
    editingBusiness: Business | null;
    clients: Business[];
    onclose: () => void;
    onclientschange: (clients: Business[]) => void;
    onselect: (id: string) => void;
  };
  let { editingId, editingBusiness, clients, onclose, onclientschange, onselect }: Props = $props();

  let formName = $state(""), formType = $state(""), formCity = $state(""), formState = $state("");
  let formPhone = $state(""), formClientName = $state(""), formClientEmail = $state("");
  let formServices: Service[] = $state([{ name: "", price_brl: 0 }]);
  let formTargetAudience = $state(""), formBrandVibe = $state(""), formQuirks = $state("");
  let formError = $state(""), formSaving = $state(false);
  type VoiceMode = 'idle' | 'recording' | 'analyzing' | 'done' | 'manual';
  let voiceMode = $state<VoiceMode>('idle');
  let voiceError = $state(''), recordingSeconds = $state(0), aiFilledFields = $state(new Set<string>());
  let mediaRecorderRef: MediaRecorder | null = null, recordingChunks: Blob[] = [], recordingTimer: ReturnType<typeof setInterval> | null = null;
  let inviteUrl = $state(""), inviteCopied = $state(false);

  $effect(() => {
    if (editingId && editingBusiness) {
      const b = editingBusiness;
      formName = b.name; formType = b.type === "Desconhecido" ? "" : b.type;
      formCity = b.city === "-" ? "" : b.city; formState = b.state === "-" ? "" : b.state;
      formPhone = b.phone || ""; formClientName = b.client_name || ""; formClientEmail = b.client_email || "";
      formServices = b.services?.length > 0 ? [...b.services] : [{ name: "", price_brl: 0 }];
      formTargetAudience = b.target_audience || ""; formBrandVibe = b.brand_vibe || ""; formQuirks = b.quirks || "";
      formError = ""; inviteUrl = ""; voiceMode = 'manual'; voiceError = ''; aiFilledFields = new Set();
    } else { resetForm(); voiceMode = 'idle'; }
  });

  function resetForm() {
    formName = ""; formType = ""; formCity = ""; formState = ""; formPhone = "";
    formClientName = ""; formClientEmail = ""; formServices = [{ name: "", price_brl: 0 }];
    formTargetAudience = ""; formBrandVibe = ""; formQuirks = ""; formError = ""; inviteUrl = ""; resetVoice();
  }
  function addService() { formServices = [...formServices, { name: "", price_brl: 0 }]; }
  function removeService(i: number) { formServices = formServices.filter((_: Service, idx: number) => idx !== i); }
  function cancelRecording() {
    if (recordingTimer) { clearInterval(recordingTimer); recordingTimer = null; }
    voiceMode = 'idle'; recordingSeconds = 0; recordingChunks = [];
    if (mediaRecorderRef && mediaRecorderRef.state !== 'inactive') mediaRecorderRef.stop(); mediaRecorderRef = null;
  }
  function submitRecording() {
    if (recordingTimer) { clearInterval(recordingTimer); recordingTimer = null; }
    if (mediaRecorderRef && mediaRecorderRef.state !== 'inactive') { voiceMode = 'analyzing'; mediaRecorderRef.stop(); } mediaRecorderRef = null;
  }
  function resetVoice() { cancelRecording(); voiceError = ''; aiFilledFields = new Set(); voiceMode = 'idle'; }

  async function startVoiceRecording() {
    voiceError = '';
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const recorder = new MediaRecorder(stream); recordingChunks = [];
      recorder.ondataavailable = (e) => { if (e.data.size > 0) recordingChunks.push(e.data); };
      recorder.onstop = async () => { for (const t of stream.getTracks()) t.stop(); await extractVoice(new Blob(recordingChunks, { type: recorder.mimeType || 'audio/webm' })); };
      recorder.start(200); mediaRecorderRef = recorder; recordingSeconds = 0;
      recordingTimer = setInterval(() => recordingSeconds++, 1000); voiceMode = 'recording';
    } catch (err) {
      const name = err instanceof DOMException ? err.name : '';
      voiceError = name === 'NotAllowedError' || name === 'PermissionDeniedError' ? 'Permissão do microfone negada. Permita o acesso nas configurações do navegador ou preencha os campos manualmente.'
        : !navigator.mediaDevices ? 'O microfone requer uma conexão segura (HTTPS). Preencha os campos manualmente.'
        : 'Não foi possível acessar o microfone. Preencha os campos manualmente.';
      voiceMode = 'manual';
    }
  }

  async function extractVoice(blob: Blob) {
    if (voiceMode !== 'analyzing') return;
    try {
      const data = await api.extractVoiceProfile(blob, formType || '');
      const filled = new Set<string>();
      if (data.services?.length) { formServices = formServices.some((s: Service) => s.name.trim()) ? [...formServices, ...data.services.map(s => ({ name: s.name, price_brl: s.price_brl ?? 0 }))] : data.services.map(s => ({ name: s.name, price_brl: s.price_brl ?? 0 })); filled.add('services'); }
      if (data.target_audience && !formTargetAudience.trim()) { formTargetAudience = data.target_audience; filled.add('target_audience'); }
      if (data.brand_vibe && !formBrandVibe.trim()) { formBrandVibe = data.brand_vibe; filled.add('brand_vibe'); }
      if (data.quirks?.length && !formQuirks.trim()) { formQuirks = data.quirks.join('\n'); filled.add('quirks'); }
      aiFilledFields = filled; voiceMode = 'done';
    } catch { voiceError = 'Não foi possível analisar o áudio. Preencha os campos manualmente.'; voiceMode = 'manual'; }
  }

  function validateForm(invite: boolean): string | null {
    if (!formName.trim() || !formType || !formCity.trim() || !formState) return "Preencha nome, tipo, cidade e estado.";
    if (invite) { if (!formClientName.trim()) return "Preencha o nome do cliente para enviar convite."; if (!formClientEmail.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formClientEmail.trim())) return "Preencha um email válido para enviar convite."; }
    else { if (formClientName.trim() && !formClientEmail.trim()) return "Preencha o email do cliente."; if (formClientEmail.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formClientEmail.trim())) return "Email inválido."; }
    if (!formServices.some((s: Service) => s.name.trim())) return "Adicione pelo menos um serviço.";
    return null;
  }

  async function saveBiz(): Promise<string | null> {
    const d = { user: pb.authStore.record!.id, name: formName.trim(), type: formType, city: formCity.trim(), state: formState, phone: formPhone.trim(), client_name: formClientName.trim(), client_email: formClientEmail.trim(), services: formServices.filter((s: Service) => s.name.trim()), target_audience: formTargetAudience.trim(), brand_vibe: formBrandVibe.trim(), quirks: formQuirks.trim() };
    if (editingId) { const { user: _, ...u } = d; const up = await api.updateBusiness(editingId, u); onclientschange(clients.map(c => c.id === editingId ? up : c)); return editingId; }
    const created = await api.createBusiness(d); onclientschange([...clients, created].sort((a, b) => a.name.localeCompare(b.name))); onselect(created.id); return created.id;
  }

  async function saveClient() { const e = validateForm(false); if (e) { formError = e; return; } formError = ""; formSaving = true; try { await saveBiz(); onclose(); } catch { formError = "Erro ao salvar. Tente novamente."; } finally { formSaving = false; } }
  async function saveAndInvite() { const e = validateForm(true); if (e) { formError = e; return; } formError = ""; formSaving = true; try { const id = await saveBiz(); inviteUrl = await api.sendInvite(id!); const r = await api.refreshBusiness(id!); onclientschange(clients.map(c => c.id === id ? r : c)); } catch { formError = "Erro ao enviar convite. Tente novamente."; } finally { formSaving = false; } }
  async function copyInviteUrl() { if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(inviteUrl); inviteCopied = true; setTimeout(() => { inviteCopied = false; }, 2000); }

  const inputCls = "px-3 py-3 rounded-xl text-base outline-none border border-[--border-strong] bg-[--surface] text-foreground";
  const labelCls = "text-base font-medium text-foreground";
</script>

{#snippet serviceEditor(highlight: boolean)}
  <div>
    <span class={labelCls}>Serviços</span>
    {#if voiceMode !== 'done'}<p class="text-sm mt-0.5 mb-1.5 text-muted-foreground">Coloca os serviços mais pedidos e o preço de cada um.</p>{/if}
    <div class="flex flex-col gap-2 {voiceMode === 'done' ? 'mt-1.5' : ''}">
      {#each formServices as service, i}
        <div class="flex gap-2 items-center">
          <Input bind:value={service.name} placeholder="Nome do serviço" class={`flex-1 ${highlight && aiFilledFields.has('services') ? 'border-sage bg-sage-pale' : 'border-[--border-strong] bg-[--surface]'}`} />
          <div class="relative w-28">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-base text-muted-foreground">R$</span>
            <Input type="number" bind:value={service.price_brl} min="0" class={`w-full pl-9 ${highlight && aiFilledFields.has('services') ? 'border-sage bg-sage-pale' : 'border-[--border-strong] bg-[--surface]'}`} />
          </div>
          {#if formServices.length > 1 || voiceMode === 'done'}
            <Button onclick={() => removeService(i)} variant="ghost" size="icon" class="shrink-0 w-10 h-13 text-muted-foreground text-[22px]">×</Button>
          {/if}
        </div>
      {/each}
      <Button onclick={addService} variant="ghost" size="sm" class="text-primary self-start mt-1 py-1 min-h-0">+ Adicionar serviço</Button>
    </div>
  </div>
{/snippet}

{#snippet contentFields(highlight: boolean)}
  <label class="flex flex-col gap-1.5">
    <span class={labelCls}>Quem são os clientes?</span>
    <span class="text-sm text-muted-foreground">ex: mulheres de 25 a 50 anos que moram no bairro</span>
    <Textarea bind:value={formTargetAudience} placeholder="Descreve quem costuma ir lá..." rows={2} class={highlight && aiFilledFields.has('target_audience') ? 'border-sage bg-sage-pale' : 'border-[--border-strong] bg-[--surface]'} />
  </label>
  <label class="flex flex-col gap-1.5">
    <span class={labelCls}>Como é o ambiente?</span>
    <span class="text-sm text-muted-foreground">ex: acolhedor, descontraído, serve cafezinho</span>
    <Textarea bind:value={formBrandVibe} placeholder="Conta um pouco sobre o clima do lugar..." rows={2} class={highlight && aiFilledFields.has('brand_vibe') ? 'border-sage bg-sage-pale' : 'border-[--border-strong] bg-[--surface]'} />
  </label>
  <label class="flex flex-col gap-1.5">
    <span class={labelCls}>O que faz diferente?</span>
    <span class="text-sm text-muted-foreground">ex: agenda lotada às quintas</span>
    <Textarea bind:value={formQuirks} placeholder="Detalhes que fazem a cliente escolher esse lugar..." rows={2} class={highlight && aiFilledFields.has('quirks') ? 'border-sage bg-sage-pale' : 'border-[--border-strong] bg-[--surface]'} />
  </label>
{/snippet}

{#snippet actionButtons(showSave: boolean)}
  {#if voiceMode === 'idle'}
    <Button onclick={onclose} variant="outline" class="w-full text-text-secondary">Cancelar</Button>
  {:else if showSave}
    <Button onclick={saveClient} disabled={formSaving} class="w-full">{formSaving ? "Salvando..." : "Salvar"}</Button>
    <Button onclick={saveAndInvite} disabled={formSaving} variant="whatsapp" class="w-full">{formSaving ? "Salvando..." : "Salvar e Enviar Convite"}</Button>
    <Button onclick={onclose} variant="ghost" size="sm" class="w-full text-muted-foreground">Cancelar</Button>
  {/if}
{/snippet}

<div class="flex-1 overflow-y-auto p-5 md:p-6 flex flex-col {voiceMode === 'idle' ? 'justify-center' : ''}">
  <div class="max-w-xl rounded-2xl p-5 md:p-6 {voiceMode === 'idle' ? 'mx-auto w-full' : ''} bg-[--surface] border border-border shadow-sm">
    <h2 class="text-lg font-semibold mb-4 text-foreground">{editingId ? "Editar cliente" : "Novo cliente"}</h2>
    {#if formError}<p class="text-base mb-4 p-3 rounded-lg text-[#DC2626] bg-[#FEF2F2]">{formError}</p>{/if}

    {#if voiceMode === 'done'}
      <div class="flex items-center gap-3 mb-4 p-3 rounded-xl bg-[--bg] border border-border">
        <div class="text-sm flex-1 min-w-0">
          <span class="font-semibold text-foreground">{formName || 'Novo cliente'}</span>
          {#if formType || formCity}<span class="text-muted-foreground"> · {[formType, formCity].filter(Boolean).join(', ')}</span>{/if}
        </div>
        <Button onclick={() => voiceMode = 'idle'} variant="outline" size="sm" class="text-xs px-2 py-1 rounded-lg min-h-0 text-muted-foreground">Editar</Button>
      </div>
      <div class="flex items-center gap-3 mb-5 p-3 rounded-xl bg-sage-pale border-[1.5px] border-sage-light">
        <div class="flex items-center justify-center shrink-0 w-8 h-8 rounded-full bg-sage">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        </div>
        <div>
          <p class="text-sm font-bold text-foreground mb-0.5">Perfil extraído da gravação</p>
          <p class="text-sm text-text-secondary">Revise os campos antes de salvar.</p>
        </div>
      </div>
      <div class="flex flex-col gap-4">
        {@render serviceEditor(true)}
        {@render contentFields(true)}
      </div>
      <div class="text-center mt-4">
        <Button onclick={resetVoice} variant="ghost" size="sm" class="text-muted-foreground gap-1.5">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V5a3 3 0 0 1 3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1M12 18v4M8 22h8"/></svg>
          Gravar de novo
        </Button>
      </div>
      <div class="flex flex-col md:flex-row gap-3 mt-4">
        <Button onclick={onclose} variant="outline" class="text-text-secondary">Cancelar</Button>
        <Button onclick={saveClient} disabled={formSaving}>{formSaving ? "Salvando..." : "Salvar e continuar"}</Button>
        <Button onclick={saveAndInvite} disabled={formSaving} variant="whatsapp">{formSaving ? "Salvando..." : "Salvar e Enviar Convite"}</Button>
      </div>
    {:else}
      {#if voiceError}<p class="text-sm mb-4 p-3 rounded-lg text-[#DC2626] bg-[#FEF2F2]">{voiceError}</p>{/if}
      <div class="flex flex-col gap-4">
        {#if voiceMode === 'idle'}
          <div class="flex items-center gap-3 md:gap-4 p-3 md:p-5 rounded-2xl bg-coral-pale border-[1.5px] border-coral-light">
            <Button onclick={startVoiceRecording} aria-label="Gravar descrição" variant="default" class="mic-btn rounded-full flex items-center justify-center shadow-[0_4px_16px_rgba(249,115,104,0.35)] p-0">
              <svg width="30" height="30" viewBox="0 0 24 24" fill="white"><path d="M12 2a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V5a3 3 0 0 1 3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1M12 18v4M8 22h8" stroke="white" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg>
            </Button>
            <div class="min-w-0">
              <p class="text-base font-bold text-foreground mb-1">Gravar descrição</p>
              <p class="text-sm text-text-secondary leading-snug">Toca no microfone e fala sobre a cliente</p>
            </div>
          </div>
          <div class="text-center">
            <Button onclick={() => voiceMode = 'manual'} variant="link" size="sm" class="text-muted-foreground underline underline-offset-[3px]">Preencher manualmente</Button>
          </div>
        {:else if voiceMode === 'recording'}
          <div class="rec-bar flex items-stretch rounded-2xl overflow-hidden border-[1.5px] border-[--border-strong]">
            <button onclick={cancelRecording} aria-label="Cancelar gravação" class="rec-side-btn flex items-center justify-center bg-[#FEF2F2] border-r border-r-[--border-strong]">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#EF4444" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
            <div class="flex-1 flex items-center justify-center gap-3">
              <div class="w-2.5 h-2.5 rounded-full shrink-0 animate-[blink_1s_ease-in-out_infinite] bg-[#EF4444]"></div>
              <span class="rec-timer font-bold tracking-wide text-foreground tabular-nums">{fmtTime(recordingSeconds)}</span>
              <span class="text-sm rec-label text-muted-foreground">Gravando</span>
            </div>
            <button onclick={submitRecording} aria-label="Enviar gravação" class="rec-side-btn flex items-center justify-center bg-primary text-white">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
            </button>
          </div>
        {:else if voiceMode === 'analyzing'}
          <div class="rounded-2xl flex items-center justify-center gap-3 rec-bar border-[1.5px] border-[--border-strong]">
            <div class="w-5.5 h-5.5 rounded-full border-[2.5px] border-coral border-t-transparent animate-spin shrink-0"></div>
            <span class="text-base font-medium text-text-secondary">Lendo o que você falou...</span>
          </div>
        {:else}
          {#if voiceMode === 'manual'}
            <Button onclick={resetVoice} variant="ghost" class="w-full flex items-center gap-3 mb-1 min-h-16 bg-coral-pale rounded-xl px-4 py-3.5 text-left border-b-[1.5px] border-coral-light whitespace-normal">
              <div class="flex items-center justify-center shrink-0 w-9 h-9 rounded-full bg-coral">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="white"><path d="M12 2a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V5a3 3 0 0 1 3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1M12 18v4M8 22h8" stroke="white" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg>
              </div>
              <p class="text-sm flex-1 min-w-0 text-text-secondary leading-snug">Prefere gravar uma descrição? É mais rápido.</p>
              <span class="text-sm font-bold text-coral whitespace-nowrap">Gravar →</span>
            </Button>
          {/if}
          <label class="flex flex-col gap-1.5"><span class={labelCls}>Nome do cliente</span><Input bind:value={formClientName} placeholder="Ex: Ana Silva" class={inputCls} /></label>
          <label class="flex flex-col gap-1.5"><span class={labelCls}>Email do cliente</span><Input bind:value={formClientEmail} type="email" placeholder="ana@email.com" class={inputCls} /></label>
          <label class="flex flex-col gap-1.5"><span class={labelCls}>Nome do negócio</span><Input bind:value={formName} placeholder="Nome do negócio" class={inputCls} /></label>
          <label class="flex flex-col gap-1.5"><span class={labelCls}>Tipo de negócio</span>
            <select bind:value={formType} class={inputCls}><option value="">Selecione...</option>{#each businessTypes as t}<option value={t}>{t}</option>{/each}</select>
          </label>
          <div class="flex gap-3">
            <label class="flex flex-col gap-1.5 flex-1"><span class={labelCls}>Cidade</span><Input bind:value={formCity} placeholder="Ex: São Paulo" class={inputCls} /></label>
            <label class="flex flex-col gap-1.5 w-28"><span class={labelCls}>Estado</span><select bind:value={formState} class={inputCls}><option value="">UF</option>{#each states as s}<option value={s}>{s}</option>{/each}</select></label>
          </div>
          <label class="flex flex-col gap-1.5"><span class={labelCls}>Telefone WhatsApp</span><Input bind:value={formPhone} placeholder="5511999998888" class={inputCls} /></label>
          <div class="h-px bg-border my-1"></div>
          {@render serviceEditor(false)}
          {@render contentFields(false)}
        {/if}
      </div>
      {#if inviteUrl}
        <div class="mt-4 rounded-xl p-4 bg-sage-pale border border-border">
          <p class="text-base font-medium mb-2 text-foreground">Convite enviado!</p>
          <div class="flex items-center gap-2">
            <Input readonly value={inviteUrl} class="flex-1 rounded-lg text-sm" />
            <Button onclick={copyInviteUrl} size="sm" class="rounded-lg">{inviteCopied ? "Copiado!" : "Copiar"}</Button>
          </div>
        </div>
      {/if}
      {#if voiceMode !== 'recording' && voiceMode !== 'analyzing'}
        <div class="flex flex-col md:flex-row gap-3 mt-6">{@render actionButtons(voiceMode !== 'idle')}</div>
      {/if}
    {/if}
  </div>
</div>

<style>
  @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.2; } }
  .mic-btn { width: 56px; height: 56px; }
  .rec-bar { height: 64px; }
  .rec-side-btn { width: 64px; }
  .rec-timer { font-size: 20px; }
  @media (max-width: 359px) { .rec-label { display: none; } }
  @media (min-width: 768px) { .mic-btn { width: 72px; height: 72px; } .rec-bar { height: 72px; } .rec-side-btn { width: 72px; } }
</style>
