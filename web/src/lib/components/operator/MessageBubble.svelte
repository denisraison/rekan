<script lang="ts">
  import { mediaUrl } from "$lib/operator/format";
  import type { Message } from "$lib/types";

  type Props = {
    msg: Message;
    selected: boolean;
    selectable: boolean;
    ontoggle: () => void;
  };

  let { msg, selected, selectable, ontoggle }: Props = $props();
</script>

<div class="flex {msg.direction === 'outgoing' ? 'justify-end' : 'justify-start'}">
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div
    class="rounded-2xl px-4 py-3 text-base max-w-[280px] leading-relaxed border text-[--text] {selected || msg.direction === 'outgoing' ? 'bg-coral-pale' : 'bg-[--surface]'} {msg.direction === 'outgoing' ? 'border-coral-light' : 'border-border'} {selectable ? 'cursor-pointer' : ''} {selected ? 'border-l-[3px] border-l-coral' : ''}"
    role={selectable ? 'button' : undefined}
    onclick={() => { if (selectable) ontoggle(); }}
    onkeydown={(e) => { if (selectable && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); ontoggle(); } }}
  >
    {#if msg.type === "audio"}
      <span class="text-sm font-medium block mb-1 text-muted-foreground">Áudio transcrito</span>
    {/if}

    {#if msg.type === "image" && msg.media}
      <img
        src={mediaUrl(msg)}
        alt="Imagem do cliente"
        class="rounded-xl mb-2 max-w-full max-h-60"
      />
    {/if}

    {#if msg.type === "video" && msg.media}
      <!-- svelte-ignore a11y_media_has_caption -->
      <!-- biome-ignore lint/a11y/useMediaCaption: client-uploaded video, no captions available -->
      <video
        src={mediaUrl(msg)}
        controls
        class="rounded-xl mb-2 max-w-full max-h-60"
      >
        Vídeo do cliente
      </video>
    {/if}

    {#if msg.content}
      <p class="whitespace-pre-wrap">{msg.content}</p>
    {:else if msg.type === "audio"}
      <p class="italic text-muted-foreground">Transcrição indisponível</p>
    {/if}

    <span class="text-sm block mt-1 text-muted-foreground">
      {new Date(msg.wa_timestamp || msg.created).toLocaleTimeString("pt-BR", {
        hour: "2-digit",
        minute: "2-digit",
      })}
    </span>
  </div>
</div>
