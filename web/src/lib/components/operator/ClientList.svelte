<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import type { NearestSeasonal } from "$lib/operator/constants";
  import type { ClientHealth } from "$lib/operator/health";
  import type { Business } from "$lib/types";
  import ClientCard from "./ClientCard.svelte";

  type ClientFilter = "todos" | "inativos" | "com_mensagens" | "sazonal" | "cobranca";

  type Props = {
    clients: Business[];
    filteredClients: Business[];
    selectedId: string | null;
    clientHealth: Record<string, ClientHealth>;
    unreadCounts: Record<string, number>;
    suggestionCounts: Record<string, number>;
    clientFilter: ClientFilter;
    unreadClientsCount: number;
    inactiveCount: number;
    pendingPaymentCount: number;
    scheduledMessageCount: number;
    globalNearestSeasonal: NearestSeasonal | null;
    onselectclient: (id: string) => void;
    onselectfilter: (filter: ClientFilter) => void;
    onshowaproval: () => void;
  };

  let {
    clients, filteredClients, selectedId, clientHealth, unreadCounts, suggestionCounts,
    clientFilter, unreadClientsCount, inactiveCount, pendingPaymentCount, scheduledMessageCount,
    globalNearestSeasonal, onselectclient, onselectfilter, onshowaproval,
  }: Props = $props();
</script>

<!-- Morning summary bar -->
{#if unreadClientsCount > 0 || inactiveCount > 0 || globalNearestSeasonal || pendingPaymentCount > 0 || scheduledMessageCount > 0}
  <div class="border-b-2 border-border bg-[--bg] pb-1">
    {#if unreadClientsCount > 0}
      <Button
        onclick={() => onselectfilter("com_mensagens")}
        variant="ghost"
        class="flex items-center gap-2.5 w-full min-h-12 px-5 text-left text-sm text-text-secondary border-b border-border rounded-none justify-start"
      >
        <span class="w-2 h-2 rounded-full bg-coral shrink-0 inline-block"></span>
        {unreadClientsCount} {unreadClientsCount === 1 ? "cliente" : "clientes"} com mensagens novas
        <span class="ml-auto text-muted-foreground text-xl leading-none">›</span>
      </Button>
    {/if}
    {#if inactiveCount > 0}
      <Button
        onclick={() => onselectfilter("inativos")}
        variant="ghost"
        class="flex items-center gap-2.5 w-full min-h-12 px-5 text-left text-sm text-text-secondary border-b border-border rounded-none justify-start"
      >
        <span class="w-2 h-2 rounded-full shrink-0 inline-block bg-[#EF4444]"></span>
        {inactiveCount} {inactiveCount === 1 ? "cliente" : "clientes"} inativos
        <span class="ml-auto text-muted-foreground text-xl leading-none">›</span>
      </Button>
    {/if}
    {#if pendingPaymentCount > 0}
      <Button
        onclick={() => onselectfilter("cobranca")}
        variant="ghost"
        class="flex items-center gap-2.5 w-full min-h-12 px-5 text-left text-sm text-text-secondary border-b border-border rounded-none justify-start"
      >
        <span class="w-2 h-2 rounded-full shrink-0 inline-block bg-[#F59E0B]"></span>
        {pendingPaymentCount} {pendingPaymentCount === 1 ? "cliente" : "clientes"} com pagamento pendente
        <span class="ml-auto text-muted-foreground text-xl leading-none">›</span>
      </Button>
    {/if}
    {#if scheduledMessageCount > 0}
      <Button
        onclick={onshowaproval}
        variant="ghost"
        class="flex items-center gap-2.5 w-full min-h-12 px-5 text-left text-sm font-medium text-coral rounded-none justify-start"
      >
        <span class="text-base shrink-0">📅</span>
        {scheduledMessageCount} {scheduledMessageCount === 1 ? "mensagem sazonal" : "mensagens sazonais"} para aprovar
        <span class="ml-auto text-coral text-xl leading-none">›</span>
      </Button>
    {/if}
    {#if globalNearestSeasonal}
      <Button
        onclick={() => onselectfilter("sazonal")}
        variant="ghost"
        class="flex items-center gap-2.5 w-full min-h-12 px-5 text-left text-sm text-text-secondary border-t border-border rounded-none justify-start"
      >
        <span class="w-2 h-2 rounded-full bg-sage shrink-0 inline-block"></span>
        {globalNearestSeasonal.label} em {globalNearestSeasonal.daysUntil}d ({globalNearestSeasonal.eligibleCount} clientes)
        <span class="ml-auto text-muted-foreground text-xl leading-none">›</span>
      </Button>
    {/if}
  </div>
{/if}

<!-- Color legend -->
<div class="flex gap-4 items-center px-5 py-2 bg-[--bg] border-b border-border">
  <span class="text-[13px] text-muted-foreground">Estado:</span>
  <span class="flex items-center gap-1.5 text-[13px] text-muted-foreground">
    <span class="w-2 h-2 rounded-full inline-block bg-[#10B981]"></span>Ativo
  </span>
  <span class="flex items-center gap-1.5 text-[13px] text-muted-foreground">
    <span class="w-2 h-2 rounded-full inline-block bg-[#F59E0B]"></span>5–9d
  </span>
  <span class="flex items-center gap-1.5 text-[13px] text-muted-foreground">
    <span class="w-2 h-2 rounded-full inline-block bg-[#EF4444]"></span>+10d
  </span>
</div>

<!-- Client list -->
<div class="flex-1 overflow-y-auto">
  {#if clientFilter !== 'todos'}
    <Button
      onclick={() => onselectfilter('todos')}
      variant="soft"
      size="sm"
      class="flex items-center gap-1.5 w-full min-h-11 px-5 rounded-none border-b border-coral-light justify-start"
    >← Todos os clientes</Button>
  {/if}
  {#if clients.length === 0}
    <p class="text-base p-5 text-muted-foreground">
      Você ainda não tem clientes. Toca no + pra começar!
    </p>
  {:else}
    {#each filteredClients as client (client.id)}
      <ClientCard
        {client}
        selected={selectedId === client.id}
        health={clientHealth[client.id]}
        unreadCount={unreadCounts[client.id] || 0}
        suggestionCount={suggestionCounts[client.id] ?? 0}
        onselect={() => onselectclient(client.id)}
      />
    {/each}
  {/if}
</div>
