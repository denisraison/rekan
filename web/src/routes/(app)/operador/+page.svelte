<script lang="ts">
  import QRCode from "qrcode";
  import { onDestroy, onMount } from "svelte";
  import ApprovalPanel from "$lib/components/operator/ApprovalPanel.svelte";
  import ChatHeader from "$lib/components/operator/ChatHeader.svelte";
  import ClientList from "$lib/components/operator/ClientList.svelte";
  import IdeaPicker from "$lib/components/operator/IdeaPicker.svelte";
  import InfoScreen from "$lib/components/operator/InfoScreen.svelte";
  import InputBar from "$lib/components/operator/InputBar.svelte";
  import MessageThread from "$lib/components/operator/MessageThread.svelte";
  import NewClientForm from "$lib/components/operator/NewClientForm.svelte";
  import PostReviewOverlay from "$lib/components/operator/PostReviewOverlay.svelte";
  import * as api from "$lib/operator/api";
  import { findNearestSeasonal, findNudgeTier, getUpcomingDates, resolveTemplate } from "$lib/operator/constants";
  import { groupMessagesByDate } from "$lib/operator/format";
  import { computeClientHealth } from "$lib/operator/health";
  import { pb } from "$lib/pb";
  import { onResume, readSSE } from "$lib/sse";
  import type { Business, GeneratedPost, Message, Post, ProfileSuggestion, ScheduledMessage, WAStatus } from "$lib/types";
  import { Button } from "$lib/components/ui/button";
  import { copyText } from "$lib/utils";

  let clients = $state<Business[]>([]);
  let selectedId = $state<string | null>(null);
  let mobileView = $state<'list' | 'detail' | 'info'>('list');
  let loading = $state(true);
  let waConnected = $state(false);
  let waQR = $state("");
  let waChecking = $state(true);
  let messages = $state<Message[]>([]);
  let posts = $state<Post[]>([]);
  let threadEl = $state<HTMLDivElement | null>(null);
  let unsubs: ((() => void) | null)[] = [];
  let showForm = $state(false);
  let editingId = $state<string | null>(null);
  let message = $state("");
  let generating = $state(false);
  let generateError = $state("");
  let result = $state<GeneratedPost | null>(null);
  let sending = $state(false), sendError = $state(""), editingCaption = $state("");
  let inputMode = $state<'chat' | 'generate'>('chat');
  let sendingQuick = $state(false);
  let quickReplyError = $state("");
  let selectedMessages = $state(new Set<string>());
  let showReviewOverlay = $state(false);
  let historyLimit = $state(10);
  let expandedPosts = $state(new Set<string>());
  let ideaDrafts = $state<GeneratedPost[] | null>(null);
  let generatingIdeas = $state(false);
  let ideaError = $state("");
  let isProactive = $state(false);
  let selectedIdeas = $state(new Set<number>());
  let sendingIdeas = $state(false);
  let sendingMedia = $state(false);
  let attachedFile = $state<File | null>(null);
  let attachedPreview = $state("");
  let toastMessage = $state("");
  let toastTimer: ReturnType<typeof setTimeout> | null = null;
  let scheduledMessages = $state<ScheduledMessage[]>([]);
  let showApprovalPanel = $state(false);
  let approvingId = $state<string | null>(null);
  let dismissingId = $state<string | null>(null);
  let suggestions = $state<ProfileSuggestion[]>([]);
  let suggestionCounts = $state<Record<string, number>>({});
  let suggestionsOpen = $state(false);
  let clientFilter = $state<"todos" | "inativos" | "com_mensagens" | "sazonal" | "cobranca">("todos");
  let nudgeText = $state("");
  let sendingNudge = $state(false);
  let sendNudgeError = $state("");
  let cancelling = $state(false);
  let lastSeen = $state<Record<string, string>>({});
  let cleanupVisibility: (() => void) | null = null;
  let waAbortController: AbortController | null = null;
  let waDisconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let qrDataUrl = $state("");

  let selected = $derived(clients.find((c) => c.id === selectedId) ?? null);
  let scheduledMessageCount = $derived(scheduledMessages.length);
  let unreadCounts = $derived.by(() => {
    const counts: Record<string, number> = {};
    for (const c of clients) {
      const seen = lastSeen[c.id];
      counts[c.id] = messages.filter(m => m.business === c.id && m.direction === "incoming" && (!seen || m.created > seen)).length;
    }
    return counts;
  });
  let threadMessages = $derived(selectedId ? messages.filter(m => m.business === selectedId).sort((a, b) => a.created.localeCompare(b.created)) : []);
  let recentContextIds = $derived.by(() => {
    const lastOut = [...threadMessages].reverse().find(m => m.direction === 'outgoing');
    const cutoff = lastOut ? lastOut.created : new Date(Date.now() - 86400000).toISOString();
    return new Set(threadMessages.filter(m => m.direction === 'incoming' && m.content && m.created > cutoff).map(m => m.id));
  });
  let groupedMessages = $derived(groupMessagesByDate(threadMessages));
  let clientHealth = $derived(computeClientHealth(clients, messages, posts));
  let clientPosts = $derived(selectedId ? posts.filter(p => p.business === selectedId) : []);
  let sortedClients = $derived([...clients].sort((a, b) => (clientHealth[b.id]?.daysSinceMsg ?? 0) - (clientHealth[a.id]?.daysSinceMsg ?? 0)));
  let inactiveCount = $derived(clients.filter(c => (clientHealth[c.id]?.daysSinceMsg ?? 0) >= 5).length);
  let unreadClientsCount = $derived(clients.filter(c => (unreadCounts[c.id] ?? 0) > 0).length);
  let pendingPaymentCount = $derived(clients.filter(c => c.charge_pending).length);
  let globalNearestSeasonal = $derived(findNearestSeasonal(clients.map(c => c.type)));
  let filteredClients = $derived.by(() => {
    switch (clientFilter) {
      case "inativos": return sortedClients.filter(c => (clientHealth[c.id]?.daysSinceMsg ?? 0) >= 5);
      case "com_mensagens": return sortedClients.filter(c => (unreadCounts[c.id] ?? 0) > 0);
      case "sazonal": return !globalNearestSeasonal ? sortedClients : sortedClients.filter(c => globalNearestSeasonal!.niches.length === 0 || globalNearestSeasonal!.niches.includes(c.type));
      case "cobranca": return sortedClients.filter(c => c.charge_pending);
      default: return sortedClients;
    }
  });
  let nudgeTier = $derived(selected ? findNudgeTier(clientHealth[selected.id]?.daysSinceMsg ?? 999) : null);
  let upcomingDates = $derived(selected ? getUpcomingDates(selected.type) : []);
  let blockReason = $derived(!waConnected ? "WhatsApp desconectado" : !selected?.phone ? "Cliente sem telefone cadastrado" : null);
  let showGenerateIdeasButton = $derived(!!selected && !ideaDrafts && !result && !generating && !generatingIdeas && ((clientHealth[selectedId!]?.daysSinceMsg ?? 999) >= 5 || threadMessages.length === 0));

  function handlePopState(event: PopStateEvent) {
    const v = event.state?.mobileView;
    if (v === 'detail' || v === 'info') mobileView = v;
    else { mobileView = 'list'; selectedId = null; }
  }

  onMount(async () => {
    window.addEventListener("popstate", handlePopState);
    lastSeen = JSON.parse(localStorage.getItem("rekan_operator_last_seen") ?? "{}");
    clients = await api.fetchClients();
    loading = false;
    connectWhatsAppStream();
    await Promise.all([loadMessages(), loadPosts()]);
    await Promise.all([loadScheduledMessages(), loadAllSuggestionCounts()]);
    unsubs.push(await api.subscribeMessages((action, record) => {
      if (action === "create") messages = [...messages, record];
      else if (action === "update") messages = messages.map(m => m.id === record.id ? record : m);
    }));
    unsubs.push(await api.subscribeBusinesses((action, record) => {
      if (action === "create" && !clients.some(c => c.id === record.id)) clients = [...clients, record].sort((a, b) => a.name.localeCompare(b.name));
      else if (action === "update") clients = clients.map(c => c.id === record.id ? record : c);
      else if (action === "delete") clients = clients.filter(c => c.id !== record.id);
    }));
    unsubs.push(await api.subscribePosts((action, record) => {
      if (action === "create" && !posts.some(p => p.id === record.id)) posts = [record, ...posts];
      else if (action === "update") posts = posts.map(p => p.id === record.id ? record : p);
      else if (action === "delete") posts = posts.filter(p => p.id !== record.id);
    }));
    unsubs.push(await api.subscribeScheduledMessages(loadScheduledMessages));
    unsubs.push(await api.subscribeSuggestions((action, record) => {
      if (action === "create" && !record.dismissed) {
        suggestionCounts = { ...suggestionCounts, [record.business]: (suggestionCounts[record.business] ?? 0) + 1 };
        if (record.business === selectedId) suggestions = [...suggestions, record];
      } else if (action === "update" && record.dismissed) {
        decSugCount(record.business);
        suggestions = suggestions.filter(s => s.id !== record.id);
      }
    }));
    cleanupVisibility = onResume(() => {
      waAbortController?.abort(); waChecking = true;
      connectWhatsAppStream(); loadMessages(); loadPosts(); loadScheduledMessages(); loadAllSuggestionCounts();
    });
  });
  $effect(() => { if (waQR) QRCode.toDataURL(waQR, { width: 256, margin: 2 }).then((u: string) => { qrDataUrl = u; }); else qrDataUrl = ""; });
  $effect(() => { void [threadMessages.length, selectedId]; if (threadEl) threadEl.scrollTop = threadEl.scrollHeight; });
  $effect(() => { suggestionsOpen = false; if (selectedId) loadSuggestions(selectedId); else suggestions = []; });
  $effect(() => { if (result) { showReviewOverlay = true; editingCaption = result.caption; } else { showReviewOverlay = false; } });
  onDestroy(() => {
    window.removeEventListener("popstate", handlePopState);
    cleanupVisibility?.(); unsubs.forEach(u => u?.());
    waAbortController?.abort();
    if (waDisconnectTimer) { clearTimeout(waDisconnectTimer); waDisconnectTimer = null; }
    removeAttachment();
  });
  async function connectWhatsAppStream() {
    waAbortController = new AbortController();
    try {
      const res = await fetch(`${pb.baseUrl}/api/whatsapp/stream`, { headers: { Authorization: pb.authStore.token }, signal: waAbortController.signal });
      if (!res.body) return;
      await readSSE(res.body, (data) => {
        const s = data as WAStatus;
        if (waDisconnectTimer) { clearTimeout(waDisconnectTimer); waDisconnectTimer = null; }
        waConnected = s.connected; waQR = s.qr ?? ""; waChecking = false;
      });
    } catch (err) {
      if (err instanceof Error && err.name === "AbortError") return;
      if (waDisconnectTimer) clearTimeout(waDisconnectTimer);
      waDisconnectTimer = setTimeout(() => { waConnected = false; waDisconnectTimer = null; }, 5000);
      waQR = ""; waChecking = false;
    }
  }

  async function loadMessages() { messages = await api.fetchMessages(); }
  async function loadPosts() { try { posts = await api.fetchPosts(); } catch { /* non-critical */ } }
  async function loadScheduledMessages() { try { scheduledMessages = await api.fetchScheduledMessages(); } catch { /* non-critical */ } }
  async function loadAllSuggestionCounts() { try { suggestionCounts = await api.fetchSuggestionCounts(); } catch { /* non-critical */ } }
  async function loadSuggestions(bid: string) { try { const items = await api.fetchSuggestions(bid); if (selectedId === bid) suggestions = items; } catch { if (selectedId === bid) suggestions = []; } }
  function decSugCount(bid: string) { suggestionCounts = { ...suggestionCounts, [bid]: Math.max(0, (suggestionCounts[bid] ?? 1) - 1) }; }
  function selectClient(id: string) {
    selectedId = id; mobileView = 'detail'; history.pushState({ mobileView: 'detail' }, '');
    result = null; generateError = ""; sendNudgeError = ""; historyLimit = 10; expandedPosts = new Set();
    showForm = false; editingId = null; showApprovalPanel = false;
    ideaDrafts = null; ideaError = ""; isProactive = false; selectedIdeas = new Set(); sendingIdeas = false;
    inputMode = 'chat'; message = ""; quickReplyError = ""; selectedMessages = new Set(); removeAttachment();
    lastSeen = { ...lastSeen, [id]: new Date().toISOString() };
    localStorage.setItem("rekan_operator_last_seen", JSON.stringify(lastSeen));
    const client = clients.find(c => c.id === id);
    const tier = clientHealth[id] ? findNudgeTier(clientHealth[id].daysSinceMsg) : null;
    nudgeText = client && tier ? resolveTemplate(tier.template, client.client_name, client.name) : "";
  }

  function toggleMsg(id: string) { const n = new Set(selectedMessages); if (n.has(id)) n.delete(id); else n.add(id); selectedMessages = n; }
  function toggleIdea(i: number) { if (selectedIdeas.has(i)) { const n = new Set(selectedIdeas); n.delete(i); selectedIdeas = n; } else selectedIdeas = new Set(selectedIdeas).add(i); }

  async function sendQuickReply() {
    if (!selectedId || (!message.trim() && !attachedFile)) return;
    if (attachedFile) { await sendMedia(attachedFile); return; }
    sendingQuick = true; quickReplyError = "";
    try { await api.sendMessage(selectedId, message.trim()); message = ""; } catch { quickReplyError = "Erro ao enviar. Tente novamente."; } finally { sendingQuick = false; }
  }

  function removeAttachment() { if (attachedPreview) URL.revokeObjectURL(attachedPreview); attachedFile = null; attachedPreview = ""; }
  async function sendMedia(file: File) {
    if (!selectedId) return; sendingMedia = true;
    try { await api.sendMediaMessage(selectedId, file, message.trim()); message = ""; removeAttachment(); } catch { showToast("Erro ao enviar midia. Tente novamente."); } finally { sendingMedia = false; }
  }
  function showToast(msg: string) { toastMessage = msg; if (toastTimer) clearTimeout(toastTimer); toastTimer = setTimeout(() => { toastMessage = ""; }, 3000); }
  function handleAttachFile(accept: string, capture?: string) {
    const input = document.createElement("input"); input.type = "file"; input.accept = accept;
    if (capture) input.setAttribute("capture", capture);
    input.onchange = () => { const f = input.files?.[0]; if (f) { attachedFile = f; attachedPreview = URL.createObjectURL(f); } }; input.click();
  }

  function openNewForm() { editingId = null; showForm = true; if (mobileView === 'list') history.pushState({ mobileView: 'detail' }, ''); mobileView = 'detail'; }
  function openEditForm(biz: Business) { editingId = biz.id; showForm = true; if (mobileView === 'list') history.pushState({ mobileView: 'detail' }, ''); mobileView = 'detail'; }
  function closeForm() { showForm = false; editingId = null; if (!selectedId) history.back(); }



  async function acceptSuggestion(sug: ProfileSuggestion) {
    const biz = clients.find(c => c.id === sug.business); if (!biz) return;
    const update = await api.acceptSuggestion(biz, sug); if (!Object.keys(update).length) return;
    clients = clients.map(c => c.id === biz.id ? ({ ...c, ...update } as Business) : c);
    suggestions = suggestions.filter(s => s.id !== sug.id); decSugCount(sug.business);
  }
  async function dismissSuggestion(sug: ProfileSuggestion) {
    await api.dismissSuggestion(sug.id); suggestions = suggestions.filter(s => s.id !== sug.id); decSugCount(sug.business);
  }

  async function generate() {
    const hasSel = selectedMessages.size > 0, hasTxt = message.trim().length > 0, hasAtt = !!attachedFile;
    if (!selectedId || (!hasSel && !hasTxt && !hasAtt)) return;
    generating = true; generateError = ""; result = null;
    try {
      const payload: Record<string, string> = {};
      const imgDesc = hasAtt && attachedFile ? await api.describeMedia(attachedFile) : "";
      if (selectedMessages.size === 1 && !hasTxt && !hasAtt) {
        const [id] = selectedMessages; payload.message_id = id; payload.message = threadMessages.find(m => m.id === id)?.content || '';
      } else {
        const parts: string[] = [];
        if (imgDesc) parts.push("[Foto do operador] " + imgDesc);
        const msgParts: string[] = [];
        for (const m of threadMessages) { if (!selectedMessages.has(m.id)) continue; msgParts.push(m.type === 'image' ? (m.content || '[Imagem]') : m.type === 'video' ? (m.content || '[Video]') : m.content || ''); }
        if (message.trim()) msgParts.push(message.trim());
        if (msgParts.length) parts.push(msgParts.join('\n'));
        payload.message = parts.join("\n\n");
      }
      result = await api.generatePost(selectedId, payload); removeAttachment();
    } catch (err: unknown) { generateError = (err as { data?: { message?: string } })?.data?.message ?? "Erro ao gerar conteúdo. Tente novamente."; }
    finally { generating = false; }
  }

  async function generateIdeas() {
    if (!selectedId) return; generatingIdeas = true; ideaError = "";
    try { ideaDrafts = await api.generateIdeas(selectedId); } catch { ideaError = "Erro ao gerar ideias. Tente novamente."; } finally { generatingIdeas = false; }
  }

  async function sendViaWhatsApp(caption: string) {
    if (!selectedId || !result) return; sending = true; sendError = "";
    try {
      if (isProactive) await api.saveProactivePost(selectedId, caption, result.hashtags, result.production_note || "");
      await api.sendMessage(selectedId, caption, result.hashtags, result.production_note || "");
      result = null; message = ""; isProactive = false; ideaDrafts = null; selectedIdeas = new Set();
    } catch { sendError = "Erro ao enviar. Tente novamente."; } finally { sending = false; }
  }

  async function sendSelectedIdeas() {
    if (!selectedId || !ideaDrafts || selectedIdeas.size === 0) return; sendingIdeas = true; sendError = "";
    try {
      for (const idx of [...selectedIdeas].sort((a, b) => a - b)) {
        const d = ideaDrafts[idx];
        await api.saveProactivePost(selectedId, d.caption, d.hashtags, d.production_note || "");
        await api.sendMessage(selectedId, d.caption, d.hashtags, d.production_note || "");
      }
      ideaDrafts = null; selectedIdeas = new Set();
    } catch { sendError = "Erro ao enviar ideias. Tente novamente."; } finally { sendingIdeas = false; }
  }

  async function sendNudge() {
    if (!selectedId || !nudgeText.trim()) return; sendingNudge = true; sendNudgeError = "";
    try { await api.sendMessage(selectedId, nudgeText.trim()); nudgeText = ""; } catch { sendNudgeError = "Erro ao enviar lembrete. Tente novamente."; } finally { sendingNudge = false; }
  }

  async function approveScheduled(id: string) { approvingId = id; try { await api.approveScheduledMessage(id); scheduledMessages = scheduledMessages.filter(m => m.id !== id); } catch {} finally { approvingId = null; } }
  async function dismissScheduled(id: string) { dismissingId = id; try { await api.dismissScheduledMessage(id); scheduledMessages = scheduledMessages.filter(m => m.id !== id); } catch {} finally { dismissingId = null; } }
  async function cancelSub() {
    if (!selected || !confirm(`Cancelar assinatura de ${selected.name}? Essa ação não pode ser desfeita.`)) return;
    cancelling = true;
    try { await api.cancelSubscription(selected.id); const r = await api.refreshBusiness(selected.id); clients = clients.map(c => c.id === selected!.id ? r : c); } catch { alert("Erro ao cancelar assinatura. Tente novamente."); } finally { cancelling = false; }
  }
</script>

<div class="h-dvh flex flex-col bg-[--bg]">
  <header class="border-b border-border px-5 py-4 flex items-center gap-3 shrink-0 bg-[--surface] {mobileView !== 'list' ? 'hidden md:flex' : ''}">
    <span class="font-semibold text-lg md:text-base text-foreground font-[--font-primary]">Rekan</span>
    <Button onclick={openNewForm} class="ml-auto font-semibold">+ Novo Cliente</Button>
  </header>
  {#if !waConnected && !waChecking}
    <a href="/operador/whatsapp" class="shrink-0 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium bg-[#FDE8E8] text-[#9B1C1C]">
      <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>
      WhatsApp desconectado — toque para reconectar
    </a>
  {/if}
  {#if loading}
    <p class="text-base p-6 text-muted-foreground">Já vou...</p>
  {:else}
    <main class="flex-1 flex flex-col md:flex-row overflow-hidden">
      <div class="w-full md:w-[480px] md:border-r border-border flex flex-col flex-1 min-h-0 md:flex-none bg-[--surface] {mobileView === 'list' ? '' : 'hidden md:flex'}">
        {#if showApprovalPanel}
          <ApprovalPanel {scheduledMessages} {clients} {waConnected} {approvingId} {dismissingId}
            onback={() => { showApprovalPanel = false; }} onapprove={approveScheduled} ondismiss={dismissScheduled} />
        {:else}
          <ClientList {clients} {filteredClients} {selectedId} {clientHealth} {unreadCounts} {suggestionCounts}
            {clientFilter} {unreadClientsCount} {inactiveCount} {pendingPaymentCount} {scheduledMessageCount} {globalNearestSeasonal}
            onselectclient={selectClient} onselectfilter={(f) => { clientFilter = f; }} onshowaproval={() => { showApprovalPanel = true; }} />
        {/if}
      </div>
      <div class="flex-1 flex flex-col overflow-hidden {mobileView === 'detail' || mobileView === 'info' ? '' : 'hidden md:flex'}">
        {#if showForm}
          <NewClientForm {editingId} editingBusiness={editingId ? clients.find(c => c.id === editingId) ?? null : null}
            {clients} onclose={closeForm} onclientschange={(c) => { clients = c; }} onselect={(id) => { selectedId = id; }} />
        {:else if selected}
          {#if mobileView === 'info'}
            <InfoScreen client={selected} clientHealth={clientHealth[selected.id]} {clientPosts} {suggestions} {suggestionsOpen}
              {nudgeTier} {nudgeText} {sendingNudge} {sendNudgeError} {blockReason} {upcomingDates} {cancelling} {expandedPosts} {historyLimit}
              onback={() => { history.back(); }} onedit={() => openEditForm(selected!)}
              onacceptsuggestion={acceptSuggestion} ondismisssuggestion={dismissSuggestion}
              ontogglesuggestions={() => { suggestionsOpen = !suggestionsOpen; }}
              onnudgetextchange={(v) => { nudgeText = v; }} onsendnudge={sendNudge}
              onprefillseasonal={(t) => { if (selected) nudgeText = resolveTemplate(t, selected.client_name, selected.name); }}
              oncancelsubscription={cancelSub} oncopypost={copyText}
              ontogglepost={(id) => { const n = new Set(expandedPosts); if (n.has(id)) n.delete(id); else n.add(id); expandedPosts = n; }}
              onshowallposts={() => { historyLimit = clientPosts.length; }} />
          {/if}
          <div class="{mobileView === 'info' ? 'hidden' : 'flex'} flex-col flex-1 overflow-hidden relative">
            <ChatHeader client={selected} onback={() => { history.back(); }}
              onopeninfo={() => { mobileView = 'info'; history.pushState({ mobileView: 'info' }, ''); }} />
            {#if generatingIdeas || ideaDrafts !== null}
              <IdeaPicker {ideaDrafts} {generatingIdeas} {selectedIdeas} {sendingIdeas}
                ontoggle={toggleIdea} onreview={(i) => { result = ideaDrafts![i]; isProactive = true; }}
                onsend={sendSelectedIdeas} onclear={() => { selectedIdeas = new Set(); }}
                onback={() => { ideaDrafts = null; selectedIdeas = new Set(); }} />
            {/if}
            {#if result && showReviewOverlay}
              <PostReviewOverlay {result} bind:editingCaption {blockReason} {sending} {sendError}
                onsend={sendViaWhatsApp}
                ondiscard={() => { result = null; message = ""; isProactive = false; ideaDrafts = null; selectedIdeas = new Set(); }}
                onback={() => { if (ideaDrafts) result = null; else showReviewOverlay = false; }} />
            {/if}
            <MessageThread {groupedMessages} {selectedMessages} selectableMode={inputMode === 'generate'} clientName={selected.name} bind:threadEl ontoggle={toggleMsg} />
            <InputBar {inputMode} {message} {blockReason} {generating} {generatingIdeas} {sendingQuick} {sendingMedia}
              {sendingIdeas} {quickReplyError} {generateError} {ideaError} {sendError} {selectedMessages} {recentContextIds}
              {showGenerateIdeasButton} {ideaDrafts} {selectedIdeas} {result} {showReviewOverlay} {attachedPreview}
              onmodechange={(m) => { inputMode = m; message = ''; selectedMessages = new Set(); removeAttachment(); }}
              onmessagechange={(v) => { message = v; }} onsendquick={sendQuickReply} ongenerate={generate} ongenerateideas={generateIdeas}
              onselectrecent={() => { selectedMessages = new Set(recentContextIds); }}
              onattachfile={handleAttachFile} onremoveattachment={removeAttachment}
              onshowreview={() => { showReviewOverlay = true; }} ontoggleidea={toggleIdea}
              onreviewidea={(i) => { result = ideaDrafts![i]; isProactive = true; }}
              onsendideas={sendSelectedIdeas} onclearideas={() => { selectedIdeas = new Set(); }}
              ondismissideas={() => { ideaDrafts = null; selectedIdeas = new Set(); }} />
          </div>
        {:else}
          <div class="flex-1 flex items-center justify-center">
            <p class="text-base text-muted-foreground">Escolhe uma cliente na lista pra começar.</p>
          </div>
        {/if}
      </div>
    </main>
    {#if !waConnected && waQR}
      <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="rounded-2xl p-8 text-center max-w-sm bg-[--surface] border border-border shadow-sm">
          <h2 class="text-lg font-semibold mb-2 text-foreground">Conectar WhatsApp</h2>
          <p class="text-sm mb-6 text-text-secondary">Escaneie o QR code com o WhatsApp Business do Rekan.</p>
          <div class="bg-white p-4 rounded-xl inline-block">
            {#if qrDataUrl}<img src={qrDataUrl} alt="QR Code WhatsApp" width="256" height="256" />
            {:else}<div class="w-64 h-64 flex items-center justify-center"><span class="text-sm text-muted-foreground">Conectando ao WhatsApp...</span></div>{/if}
          </div>
          <p class="text-sm mt-4 text-muted-foreground">O QR code atualiza automaticamente.</p>
        </div>
      </div>
    {/if}
  {/if}
</div>
{#if toastMessage}
  <div class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 px-4 py-3 rounded-xl text-sm font-medium shadow-lg bg-[--surface] text-foreground border border-[--border-strong]">{toastMessage}</div>
{/if}
