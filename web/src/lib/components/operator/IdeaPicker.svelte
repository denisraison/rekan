<script lang="ts">
  import type { GeneratedPost } from "$lib/types";

  type Props = {
    ideaDrafts: GeneratedPost[] | null;
    generatingIdeas: boolean;
    selectedIdeas: Set<number>;
    sendingIdeas: boolean;
    ontoggle: (index: number) => void;
    onreview: (index: number) => void;
    onsend: () => void;
    onclear: () => void;
    onback: () => void;
  };

  let { ideaDrafts, generatingIdeas, selectedIdeas, sendingIdeas, ontoggle, onreview, onsend, onclear, onback }: Props = $props();
</script>

<div class="md:hidden absolute inset-0 flex flex-col z-10 bg-[--bg]">
  <div class="flex items-center gap-3 px-4 shrink-0 min-h-15 bg-[--surface] border-b border-border">
    {#if !generatingIdeas}
      <button
        onclick={onback}
        class="flex items-center gap-1 min-h-15 pr-3 text-sm font-medium text-coral shrink-0"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        Voltar
      </button>
    {/if}
    <span class="text-base font-semibold text-foreground">
      {generatingIdeas ? 'Gerando ideias...' : 'Selecione ideias'}
    </span>
  </div>
  {#if generatingIdeas}
    <div class="flex-1 flex flex-col items-center justify-center gap-4 p-8">
      <svg class="animate-spin text-coral" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
      <p class="text-base text-center text-muted-foreground">Escrevendo 3 ideias<br>caprichadas pra você...</p>
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
      {#each ideaDrafts! as draft, i}
        <button
          onclick={() => ontoggle(i)}
          class="rounded-2xl p-5 text-left transition-colors bg-[--surface] border-2 {selectedIdeas.has(i) ? 'border-coral' : 'border-border'}"
        >
          <div class="flex items-start gap-3">
            <div class="shrink-0 w-6 h-6 rounded-full flex items-center justify-center mt-0.5 border-2 {selectedIdeas.has(i) ? 'border-coral bg-coral' : 'border-border bg-transparent'}">
              {#if selectedIdeas.has(i)}
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-base leading-relaxed text-foreground whitespace-pre-wrap">{draft.caption}</p>
              {#if draft.hashtags?.length}
                <p class="text-sm mt-3 text-muted-foreground">{draft.hashtags.join(' ')}</p>
              {/if}
            </div>
          </div>
        </button>
      {/each}
    </div>
    {#if selectedIdeas.size > 0}
      <div class="shrink-0 p-4 flex gap-2 bg-[--surface] border-t border-border">
        {#if selectedIdeas.size === 1}
          <button
            onclick={() => onreview([...selectedIdeas][0])}
            class="flex-1 rounded-full text-base font-semibold min-h-13 bg-coral text-white"
          >Revisar e enviar</button>
        {:else}
          <button
            onclick={onsend}
            disabled={sendingIdeas}
            class="flex-1 rounded-full text-base font-semibold min-h-13 text-white bg-[#25D366] disabled:opacity-60"
          >{sendingIdeas ? 'Enviando...' : `Enviar ${selectedIdeas.size} selecionadas`}</button>
        {/if}
        <button
          onclick={onclear}
          class="shrink-0 px-4 rounded-full text-base font-medium min-h-13 text-destructive bg-[--bg] border border-border"
        >Cancelar</button>
      </div>
    {/if}
  {/if}
</div>
