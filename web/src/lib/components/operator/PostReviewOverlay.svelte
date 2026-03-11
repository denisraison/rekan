<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
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
    <Button onclick={onback} variant="ghost" size="sm" class="text-coral shrink-0 gap-1 pr-3">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
      Voltar
    </Button>
    <span class="text-base font-semibold text-foreground">Post gerado</span>
  </div>

  <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
    <div>
      <div class="flex items-center justify-between mb-1">
        <span class="text-sm font-medium text-muted-foreground">Legenda</span>
        <Button onclick={() => copyWithFeedback("caption", editingCaption)} variant="ghost" size="sm" class="text-coral py-1 min-h-0">
          {copied.caption ? "Copiado!" : "Copiar"}
        </Button>
      </div>
      <Textarea
        bind:value={editingCaption}
        class="w-full rounded-xl p-3 text-base leading-relaxed bg-[--surface] border border-border text-foreground min-h-[120px]"
        style="field-sizing: content"
      />
    </div>

    <div>
      <div class="flex items-center justify-between mb-1">
        <span class="text-sm font-medium text-muted-foreground">Hashtags</span>
        <Button onclick={() => copyWithFeedback("hashtags", result.hashtags.join(" "))} variant="ghost" size="sm" class="text-coral py-1 min-h-0">
          {copied.hashtags ? "Copiado!" : "Copiar"}
        </Button>
      </div>
      <p class="text-sm text-text-secondary">{result.hashtags.join(" ")}</p>
    </div>

    {#if result.production_note}
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-sm font-medium text-muted-foreground">Nota de produção</span>
          <Button onclick={() => copyWithFeedback("note", result.production_note!)} variant="ghost" size="sm" class="text-coral py-1 min-h-0">
            {copied.note ? "Copiado!" : "Copiar"}
          </Button>
        </div>
        <p class="text-sm italic mt-1 text-text-secondary border-l-2 border-[--border-strong] pl-3">{result.production_note}</p>
      </div>
    {/if}
  </div>

  <div class="shrink-0 px-4 py-3 flex flex-col gap-2 bg-[--surface] border-t border-border">
    {#if blockReason}
      <span class="text-sm text-muted-foreground">{blockReason} — não é possível enviar agora.</span>
    {:else}
      <Button onclick={() => onsend(editingCaption)} disabled={sending} variant="whatsapp" class="w-full">
        {sending ? "Enviando..." : "Enviar pelo WhatsApp"}
      </Button>
      {#if sendError}
        <span class="text-sm text-destructive">{sendError}</span>
      {/if}
    {/if}
    <Button onclick={ondiscard} variant="ghost" size="sm" class="w-full text-destructive">Descartar</Button>
  </div>
</div>
