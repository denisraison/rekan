<script lang="ts">
  import { initials, profilePictureUrl } from "$lib/operator/format";
  import type { Business } from "$lib/types";

  type Props = {
    client: Business;
    onback: () => void;
    onopeninfo: () => void;
  };

  let { client, onback, onopeninfo }: Props = $props();
  let pic = $derived(profilePictureUrl(client));
</script>

<div class="px-5 py-4 border-b border-border shrink-0 bg-[--surface]">
  <div class="flex items-center gap-2">
    <button
      onclick={onback}
      class="md:hidden flex items-center gap-1 py-2 pr-2 -ml-1 rounded-lg shrink-0 text-sm font-medium text-coral"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
      Voltar
    </button>
    <button
      onclick={onopeninfo}
      class="min-w-0 flex-1 flex items-center gap-3 text-left"
    >
      <div class="shrink-0 w-9 h-9 rounded-full overflow-hidden flex items-center justify-center text-sm font-semibold bg-coral-pale text-coral">
        {#if pic}
          <img src={pic} alt={client.name} class="w-full h-full object-cover" />
        {:else}
          {initials(client)}
        {/if}
      </div>
      <div class="min-w-0">
        <h2 class="text-base font-semibold truncate text-foreground">{client.name}</h2>
        <p class="text-sm text-text-secondary">
          {client.type} — {client.city}/{client.state}
          <span class="ml-1 text-xs text-muted-foreground">›</span>
        </p>
      </div>
    </button>
  </div>
</div>
