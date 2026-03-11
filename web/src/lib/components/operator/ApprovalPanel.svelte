<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import type { Business, ScheduledMessage } from "$lib/types";

  type Props = {
    scheduledMessages: ScheduledMessage[];
    clients: Business[];
    waConnected: boolean;
    approvingId: string | null;
    dismissingId: string | null;
    onback: () => void;
    onapprove: (id: string) => void;
    ondismiss: (id: string) => void;
  };

  let { scheduledMessages, clients, waConnected, approvingId, dismissingId, onback, onapprove, ondismiss }: Props = $props();
</script>

<div class="flex-1 overflow-y-auto p-3 flex flex-col gap-2">
  <Button
    onclick={onback}
    variant="ghost"
    size="sm"
    class="text-coral self-start px-1"
  >← Clientes</Button>
  {#if scheduledMessages.length === 0}
    <p class="text-base text-center py-8 text-muted-foreground">
      Tudo em dia! Nenhuma mensagem pra aprovar.
    </p>
  {:else}
    {#each scheduledMessages as msg (msg.id)}
      {@const biz = clients.find(c => c.id === msg.business)}
      <div class="rounded-xl p-4 bg-[--bg] border border-border">
        <p class="text-sm font-medium mb-1.5 text-muted-foreground">
          {biz?.name ?? msg.business}
        </p>
        <p class="text-sm py-1.5 text-text-secondary">{msg.text}</p>
        <div class="flex gap-2 mt-2">
          <Button
            onclick={() => onapprove(msg.id)}
            disabled={approvingId === msg.id || !waConnected}
            variant="whatsapp"
            size="sm"
            class="flex-1 py-3 rounded-lg"
          >
            {approvingId === msg.id ? "..." : "Enviar"}
          </Button>
          <Button
            onclick={() => ondismiss(msg.id)}
            disabled={dismissingId === msg.id}
            variant="outline"
            size="sm"
            class="flex-1 py-3 rounded-lg text-text-secondary"
          >
            {dismissingId === msg.id ? "..." : "Descartar"}
          </Button>
        </div>
      </div>
    {/each}
  {/if}
</div>
