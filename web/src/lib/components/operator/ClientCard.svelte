<script lang="ts">
  import { inviteBadgeClass, inviteBadgeLabel } from "$lib/operator/format";
  import type { ClientHealth } from "$lib/operator/health";
  import type { Business } from "$lib/types";

  type Props = {
    client: Business;
    selected: boolean;
    health: ClientHealth | undefined;
    unreadCount: number;
    suggestionCount: number;
    onselect: () => void;
  };

  let { client, selected, health, unreadCount, suggestionCount, onselect }: Props = $props();
</script>

<button
  onclick={onselect}
  data-testid="client-card"
  class="w-full text-left px-5 py-4 border-b border-border transition-colors {selected ? 'bg-coral-pale' : 'hover:bg-coral-pale/40'}"
>
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2.5 min-w-0">
      {#if health}
        <span
          class="w-3 h-3 rounded-full shrink-0"
          style="background: {health.color}"
        ></span>
      {/if}
      <span class="font-semibold text-base truncate text-foreground">{client.name}</span>
      {#if client.invite_status && client.invite_status !== 'draft'}
        <span
          class="text-sm px-2 py-0.5 rounded-full shrink-0 {inviteBadgeClass(client.invite_status)}"
        >
          {inviteBadgeLabel(client.invite_status)}
        </span>
        {#if client.invite_status === 'accepted' && client.invite_sent_at && (Date.now() - new Date(client.invite_sent_at).getTime()) > 48 * 3600000}
          <span class="text-sm shrink-0" title="Aceito há mais de 48h sem pagamento">&#9888;</span>
        {/if}
      {/if}
    </div>
    {#if unreadCount > 0}
      <span class="text-sm font-bold px-2.5 py-1 rounded-full shrink-0 ml-2 bg-coral text-white">
        {unreadCount}
      </span>
    {/if}
    {#if suggestionCount > 0}
      <span
        class="shrink-0 ml-1 w-2 h-2 rounded-full bg-sage inline-block"
        title="Sugestões de perfil pendentes"
      ></span>
    {/if}
  </div>
  <div class="flex items-center justify-between mt-1 pl-5 gap-2">
    <span class="text-sm truncate text-muted-foreground min-w-0">{client.type} · {client.city}</span>
    {#if health && health.daysSinceMsg === 0}
      <span class="text-sm shrink-0 text-muted-foreground">Hoje</span>
    {:else if health && health.daysSinceMsg < 999}
      <span class="text-sm font-medium shrink-0" style="color: {health.daysSinceMsg >= 5 ? health.color : 'var(--text-muted)'}">
        {health.daysSinceMsg}d sem postar
      </span>
    {/if}
  </div>
  {#if client.charge_pending}
    <div class="mt-1 pl-5">
      <span class="text-[13px] px-2.5 py-0.5 rounded-full bg-[#FEF3C7] text-[#92400E]">⚠ pagamento pendente</span>
    </div>
  {/if}
</button>
