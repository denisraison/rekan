<script lang="ts">
  import type { GeneratedPost } from "$lib/types";

  type Props = {
    inputMode: 'chat' | 'generate';
    message: string;
    blockReason: string | null;
    generating: boolean;
    generatingIdeas: boolean;
    sendingQuick: boolean;
    sendingMedia: boolean;
    sendingIdeas: boolean;
    quickReplyError: string;
    generateError: string;
    ideaError: string;
    sendError: string;
    selectedMessages: Set<string>;
    recentContextIds: Set<string>;
    showGenerateIdeasButton: boolean;
    ideaDrafts: GeneratedPost[] | null;
    selectedIdeas: Set<number>;
    result: GeneratedPost | null;
    showReviewOverlay: boolean;
    attachedPreview: string;
    onmodechange: (mode: 'chat' | 'generate') => void;
    onmessagechange: (value: string) => void;
    onsendquick: () => void;
    ongenerate: () => void;
    ongenerateideas: () => void;
    onselectrecent: () => void;
    onattachfile: (accept: string, capture?: string) => void;
    onremoveattachment: () => void;
    onshowreview: () => void;
    ontoggleidea: (index: number) => void;
    onreviewidea: (index: number) => void;
    onsendideas: () => void;
    onclearideas: () => void;
    ondismissideas: () => void;
  };

  let {
    inputMode, message, blockReason, generating, generatingIdeas, sendingQuick, sendingMedia,
    sendingIdeas, quickReplyError, generateError, ideaError, sendError,
    selectedMessages, recentContextIds, showGenerateIdeasButton, ideaDrafts, selectedIdeas,
    result, showReviewOverlay, attachedPreview,
    onmodechange, onmessagechange, onsendquick, ongenerate, ongenerateideas, onselectrecent,
    onattachfile, onremoveattachment, onshowreview,
    ontoggleidea, onreviewidea, onsendideas, onclearideas, ondismissideas,
  }: Props = $props();

  let showAttachMenu = $state(false);

  let hasInput = $derived(message.trim().length > 0 || selectedMessages.size > 0 || attachedPreview.length > 0);
</script>

<div
  data-testid="input-bar"
  class="shrink-0 border-t px-3 md:px-4 py-3 flex flex-col gap-2 {inputMode === 'generate' ? 'border-coral bg-coral-pale border-t-2' : 'border-border bg-[--surface]'}"
>
  <!-- Idea drafts (desktop only) -->
  {#if ideaDrafts !== null}
    <div class="hidden md:flex flex-col gap-3 mb-2">
      <div class="flex items-center justify-between">
        <span class="text-sm font-medium text-muted-foreground">Selecione ideias</span>
        <button onclick={ondismissideas} class="text-sm py-1 text-muted-foreground">Cancelar</button>
      </div>
      {#each ideaDrafts as draft, i}
        <button
          onclick={() => ontoggleidea(i)}
          class="rounded-xl p-4 text-left transition-colors bg-[--bg] border-2 {selectedIdeas.has(i) ? 'border-coral' : 'border-border'}"
        >
          <div class="flex items-start gap-3">
            <div class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center mt-0.5 border-2 {selectedIdeas.has(i) ? 'border-coral bg-coral' : 'border-border bg-transparent'}">
              {#if selectedIdeas.has(i)}
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
              {/if}
            </div>
            <p
              class="text-base leading-relaxed min-w-0 flex-1 text-foreground line-clamp-3"
            >
              {draft.caption}
            </p>
          </div>
        </button>
      {/each}
      {#if selectedIdeas.size > 0}
        <div class="flex gap-2">
          {#if selectedIdeas.size === 1}
            <button
              onclick={() => onreviewidea([...selectedIdeas][0])}
              class="px-5 py-2.5 rounded-full text-sm font-medium bg-coral text-white"
            >Revisar e enviar</button>
          {:else}
            <button
              onclick={onsendideas}
              disabled={sendingIdeas}
              class="px-5 py-2.5 rounded-full text-sm font-medium text-white bg-[#25D366] disabled:opacity-60"
            >{sendingIdeas ? 'Enviando...' : `Enviar ${selectedIdeas.size} selecionadas`}</button>
          {/if}
          <button
            onclick={onclearideas}
            class="px-4 py-2.5 rounded-full text-sm font-medium text-destructive"
          >Limpar</button>
        </div>
      {/if}
    </div>
  {/if}

  {#if !blockReason}
    <!-- Action chips bar -->
    <div class="flex gap-2 items-center flex-wrap">
      {#if result && !showReviewOverlay}
        <button
          onclick={onshowreview}
          class="text-sm px-3 py-1.5 rounded-full font-medium flex items-center gap-1.5 bg-coral text-white"
        >
          Ver post gerado
        </button>
      {/if}
      {#if inputMode === 'generate' && ideaDrafts === null}
        {#if selectedMessages.size > 0}
          <span class="text-sm px-3 py-1.5 rounded-full font-medium bg-coral-pale text-coral">
            {selectedMessages.size} {selectedMessages.size === 1 ? 'mensagem selecionada' : 'mensagens selecionadas'}
          </span>
        {/if}
        {#if recentContextIds.size > 0 && selectedMessages.size === 0}
          <button
            onclick={onselectrecent}
            class="text-sm px-3 py-1.5 rounded-full font-medium bg-sage-pale text-text-secondary"
          >
            Selecionar recentes
          </button>
        {/if}
        {#if showGenerateIdeasButton}
          <button
            onclick={ongenerateideas}
            disabled={generatingIdeas || generating}
            class="text-sm px-3 py-1.5 rounded-full font-medium flex items-center gap-1.5 bg-sage-pale text-text-secondary disabled:opacity-50"
          >
            {#if generatingIdeas}
              <svg class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
              Criando o post...
            {:else}
              3 ideias
            {/if}
          </button>
        {/if}
      {/if}
      <button
        onclick={() => onmodechange(inputMode === 'chat' ? 'generate' : 'chat')}
        class="ml-auto text-sm px-4 py-1.5 min-h-11 rounded-full font-medium transition-colors flex items-center gap-1.5 border {inputMode === 'generate' ? 'bg-[--surface] text-[#25D366] border-[#25D366]' : 'bg-[--bg] text-coral border-coral-light'}"
      >
        {#if inputMode === 'generate'}
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          Chat
        {:else}
          Post
        {/if}
      </button>
    </div>
    <!-- Attachment preview -->
    {#if attachedPreview}
      <div class="flex items-center gap-2 px-1">
        <div class="relative">
          <img src={attachedPreview} alt="Anexo" class="w-16 h-16 rounded-lg object-cover border border-[--border-strong]" />
          <button
            onclick={onremoveattachment}
            class="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full flex items-center justify-center text-white text-xs bg-destructive"
            aria-label="Remover anexo"
          >&times;</button>
        </div>
      </div>
    {/if}
    <!-- Input row -->
    <div class="flex gap-2 items-center relative">
      <button
        onclick={() => { showAttachMenu = !showAttachMenu; }}
        disabled={sendingMedia}
        class="shrink-0 w-10 h-10 rounded-full flex items-center justify-center transition-colors text-muted-foreground disabled:opacity-50"
        aria-label="Anexar arquivo"
      >
        {#if sendingMedia}
          <svg class="animate-spin" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
        {:else}
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
        {/if}
      </button>
      {#if showAttachMenu}
        <button class="fixed inset-0 z-10" onclick={() => { showAttachMenu = false; }} aria-label="Fechar menu"></button>
        <div class="absolute bottom-12 left-0 z-20 rounded-xl shadow-lg border border-[--border-strong] p-2 flex flex-col gap-1 bg-[--bg] min-w-[180px]">
          <button
            onclick={() => { onattachfile("image/*"); showAttachMenu = false; }}
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-left transition-colors hover:bg-black/5 text-foreground"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>
            Galeria
          </button>
          <button
            onclick={() => { onattachfile("image/*", "environment"); showAttachMenu = false; }}
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-left transition-colors hover:bg-black/5 text-foreground"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/></svg>
            Camera
          </button>
        </div>
      {/if}
      <input
        value={message}
        oninput={(e) => onmessagechange(e.currentTarget.value)}
        placeholder={inputMode === 'generate' ? 'Sobre o que é o post?' : 'Escreve aqui...'}
        class="flex-1 min-w-0 px-3 md:px-4 py-3 rounded-xl text-base outline-none border border-[--border-strong] bg-[--bg] text-foreground min-h-12"
        onkeydown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            if (inputMode === 'chat') onsendquick();
            else if (hasInput) ongenerate();
          }
        }}
      />
      {#if inputMode === 'generate'}
        <button
          onclick={ongenerate}
          disabled={generating || generatingIdeas || !hasInput}
          class="shrink-0 px-3 md:px-5 py-3 rounded-full text-sm font-medium transition-opacity flex items-center gap-2 bg-coral text-white disabled:opacity-60 disabled:cursor-not-allowed"
        >
          {#if generating}
            <svg class="animate-spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
            Criando o post...
          {:else}
            Gerar
          {/if}
        </button>
      {:else}
        <button
          onclick={onsendquick}
          disabled={sendingQuick || sendingMedia || (!message.trim() && !attachedPreview)}
          class="shrink-0 px-3 md:px-5 py-3 rounded-full text-sm font-medium transition-opacity text-white bg-[#25D366] disabled:opacity-60"
        >
          {sendingQuick ? "..." : "Enviar"}
        </button>
      {/if}
    </div>
    {#if quickReplyError}
      <span class="text-sm text-destructive">{quickReplyError}</span>
    {/if}
    {#if ideaError}
      <p class="text-sm text-destructive">{ideaError}</p>
    {/if}
    {#if generateError}
      <p class="text-sm text-destructive">{generateError}</p>
    {/if}
  {:else}
    <span class="text-sm text-muted-foreground">{blockReason}</span>
  {/if}
</div>
