<script lang="ts">
  import type { GeneratedPost } from "$lib/types";
  import { copyText } from "$lib/utils";

  type Props = {
    result: GeneratedPost;
    editingCaption: string;
    blockReason: string | null;
    sending: boolean;
    sendError: string;
    onsend: (caption: string) => void;
    ondiscard: () => void;
    onback: () => void;
  };

  let { result, editingCaption = $bindable(), blockReason, sending, sendError, onsend, ondiscard, onback }: Props = $props();

  let copied = $state<Record<string, boolean>>({});
  const copyTimers: Record<string, ReturnType<typeof setTimeout>> = {};

  async function copyWithFeedback(key: string, text: string) {
    await copyText(text);
    clearTimeout(copyTimers[key]);
    copied = { ...copied, [key]: true };
    copyTimers[key] = setTimeout(() => { copied = { ...copied, [key]: false }; }, 2000);
  }
</script>

<div class="absolute inset-0 flex flex-col z-10 bg-[--bg]">
  <div class="flex items-center gap-3 px-4 shrink-0 min-h-15 bg-[--surface] border-b border-border">
    <button onclick={onback} class="flex items-center gap-1 min-h-15 pr-3 text-sm font-medium text-coral shrink-0">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
      Voltar
    </button>
    <span class="text-base font-semibold text-foreground">Post gerado</span>
  </div>

  <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
    <div>
      <div class="flex items-center justify-between mb-1">
        <span class="text-sm font-medium text-muted-foreground">Legenda</span>
        <button onclick={() => copyWithFeedback("caption", editingCaption)} class="text-sm py-1 text-coral">
          {copied.caption ? "Copiado!" : "Copiar"}
        </button>
      </div>
      <textarea
        bind:value={editingCaption}
        class="w-full rounded-xl p-3 text-base leading-relaxed resize-none bg-[--surface] border border-border text-foreground min-h-[120px]"
        style="field-sizing: content"
      ></textarea>
    </div>

    <div>
      <div class="flex items-center justify-between mb-1">
        <span class="text-sm font-medium text-muted-foreground">Hashtags</span>
        <button onclick={() => copyWithFeedback("hashtags", result.hashtags.join(" "))} class="text-sm py-1 text-coral">
          {copied.hashtags ? "Copiado!" : "Copiar"}
        </button>
      </div>
      <p class="text-sm text-text-secondary">{result.hashtags.join(" ")}</p>
    </div>

    {#if result.production_note}
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-sm font-medium text-muted-foreground">Nota de produção</span>
          <button onclick={() => copyWithFeedback("note", result.production_note!)} class="text-sm py-1 text-coral">
            {copied.note ? "Copiado!" : "Copiar"}
          </button>
        </div>
        <p class="text-sm italic mt-1 text-text-secondary border-l-2 border-[--border-strong] pl-3">{result.production_note}</p>
      </div>
    {/if}
  </div>

  <div class="shrink-0 px-4 py-3 flex flex-col gap-2 bg-[--surface] border-t border-border">
    {#if blockReason}
      <span class="text-sm text-muted-foreground">{blockReason} — não é possível enviar agora.</span>
    {:else}
      <button onclick={() => onsend(editingCaption)} disabled={sending}
        class="w-full px-6 py-3 rounded-full text-base font-medium transition-opacity text-white bg-[#25D366] disabled:opacity-60 disabled:cursor-not-allowed">
        {sending ? "Enviando..." : "Enviar pelo WhatsApp"}
      </button>
      {#if sendError}
        <span class="text-sm text-destructive">{sendError}</span>
      {/if}
    {/if}
    <button onclick={ondiscard} class="w-full py-2 text-sm font-medium text-destructive">Descartar</button>
  </div>
</div>
