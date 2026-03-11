<script lang="ts">
  import type { MessageGroup } from "$lib/operator/format";
  import MessageBubble from "./MessageBubble.svelte";

  type Props = {
    groupedMessages: MessageGroup[];
    selectedMessages: Set<string>;
    selectableMode: boolean;
    clientName: string;
    threadEl: HTMLDivElement | null;
    ontoggle: (id: string) => void;
  };

  let { groupedMessages, selectedMessages, selectableMode, clientName, threadEl = $bindable(null), ontoggle }: Props = $props();
</script>

<div bind:this={threadEl} data-testid="message-thread" class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-2 bg-chat-bg">
  {#if groupedMessages.length === 0}
    <div class="flex-1 flex items-center justify-center">
      {#if selectableMode}
        <p class="text-base text-center px-8 text-coral">
          Toque nas mensagens que quer usar no post
        </p>
      {:else}
        <p class="text-base text-center px-8 text-muted-foreground">
          Quando {clientName} mandar mensagem, aparece aqui.
        </p>
      {/if}
    </div>
  {:else}
    {#each groupedMessages as group}
      <div class="flex items-center gap-3 my-1">
        <hr class="flex-1 border-border" />
        <span class="text-sm shrink-0 text-muted-foreground">{group.label}</span>
        <hr class="flex-1 border-border" />
      </div>
      {#each group.msgs as msg (msg.id)}
        <MessageBubble
          {msg}
          selected={selectedMessages.has(msg.id)}
          selectable={selectableMode}
          ontoggle={() => ontoggle(msg.id)}
        />
      {/each}
    {/each}
  {/if}
</div>
