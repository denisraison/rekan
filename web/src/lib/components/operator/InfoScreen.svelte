<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import type { NudgeTemplate, UpcomingDate } from "$lib/operator/constants";
  import { initials, inviteBadgeClass, inviteBadgeLabel, profilePictureUrl } from "$lib/operator/format";
  import type { ClientHealth } from "$lib/operator/health";
  import type { Business, Post, ProfileSuggestion } from "$lib/types";

  type Props = {
    client: Business;
    clientHealth: ClientHealth | undefined;
    clientPosts: Post[];
    suggestions: ProfileSuggestion[];
    suggestionsOpen: boolean;
    nudgeTier: NudgeTemplate | null;
    nudgeText: string;
    sendingNudge: boolean;
    sendNudgeError: string;
    blockReason: string | null;
    upcomingDates: UpcomingDate[];
    cancelling: boolean;
    expandedPosts: Set<string>;
    historyLimit: number;
    onback: () => void;
    onedit: () => void;
    onacceptsuggestion: (sug: ProfileSuggestion) => void;
    ondismisssuggestion: (sug: ProfileSuggestion) => void;
    ontogglesuggestions: () => void;
    onnudgetextchange: (value: string) => void;
    onsendnudge: () => void;
    onprefillseasonal: (template: string) => void;
    oncancelsubscription: () => void;
    oncopypost: (text: string) => void;
    ontogglepost: (id: string) => void;
    onshowallposts: () => void;
  };

  let {
    client, clientHealth: health, clientPosts, suggestions, suggestionsOpen,
    nudgeTier, nudgeText, sendingNudge, sendNudgeError, blockReason, upcomingDates,
    cancelling, expandedPosts, historyLimit,
    onback, onedit, onacceptsuggestion, ondismisssuggestion, ontogglesuggestions,
    onnudgetextchange, onsendnudge, onprefillseasonal, oncancelsubscription,
    oncopypost, ontogglepost, onshowallposts,
  }: Props = $props();
  let pic = $derived(profilePictureUrl(client));
</script>

<div class="flex-1 flex flex-col overflow-hidden">
  <!-- Header -->
  <div class="bg-[var(--surface)] border-b border-border flex items-center min-h-15 px-4 gap-1 shrink-0">
    <Button
      onclick={onback}
      variant="ghost"
      size="sm"
      class="text-coral shrink-0 gap-1 pr-3 min-h-15"
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
      Voltar
    </Button>
    <div class="w-11 h-11 rounded-full overflow-hidden shrink-0 flex items-center justify-center text-base font-semibold bg-coral-pale text-coral">
      {#if pic}
        <img src={pic} alt={client.name} class="w-full h-full object-cover" />
      {:else}
        {initials(client)}
      {/if}
    </div>
    <div class="flex-1 min-w-0">
      <div class="text-[17px] font-semibold text-foreground truncate">{client.name}</div>
      <div class="text-sm text-text-secondary">{client.type} — {client.city}</div>
    </div>
    <Button
      onclick={onedit}
      variant="outline"
      size="sm"
      class="shrink-0 px-4.5 text-text-secondary"
    >Editar</Button>
  </div>

  <div class="flex-1 overflow-y-auto">
    <!-- Status strip -->
    <div class="bg-[var(--surface)] px-5 py-3.5 border-b border-border flex items-center gap-2.5 flex-wrap">
      {#if client.tier}
        <span class="text-sm px-3 py-1 rounded-full bg-[var(--bg)] text-text-secondary border border-[var(--border-strong)] font-medium">{client.tier}</span>
      {/if}
      {#if client.invite_status && client.invite_status !== 'draft'}
        <span
          class="text-sm px-3 py-1 rounded-full font-medium {inviteBadgeClass(client.invite_status)}"
        >{inviteBadgeLabel(client.invite_status)}</span>
      {/if}
      {#if client.invite_status === 'active' && client.next_charge_date}
        <span class="text-sm text-muted-foreground ml-1">
          Próx. cobrança: {new Date(client.next_charge_date).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' })}
        </span>
      {/if}
      {#if client.charge_pending}
        <span class="text-sm px-3 py-1 rounded-full font-medium bg-[#FEE2E2] text-[#991B1B]">Pagamento pendente</span>
      {/if}
      {#if client.type === 'Desconhecido'}
        <Button onclick={onedit} variant="whatsapp" class="text-[15px] font-semibold">Criar conta</Button>
      {/if}
    </div>

    <!-- Services -->
    {#if client.services?.length > 0}
      <span class="text-[13px] font-bold tracking-wider uppercase text-muted-foreground px-5 pt-3.5 pb-2 bg-[var(--bg)] border-t border-border block">Serviços</span>
      <div class="bg-[var(--surface)] py-1 pb-3">
        {#each client.services as svc}
          <div class="flex items-baseline gap-3 px-5 py-1.5">
            <span class="text-[13px] font-semibold text-muted-foreground whitespace-nowrap shrink-0 w-[100px] overflow-hidden text-ellipsis">{svc.name}</span>
            <span class="text-[15px] text-text-secondary flex-1">{svc.price_brl != null ? `R$${svc.price_brl.toFixed(2).replace('.', ',')}` : '—'}</span>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Profile -->
    {#if client.target_audience || client.brand_vibe || client.quirks}
      <span class="text-[13px] font-bold tracking-wider uppercase text-muted-foreground px-5 pt-3.5 pb-2 bg-[var(--bg)] border-t border-border block">Perfil</span>
      <div class="bg-[var(--surface)] py-1 pb-3">
        {#if client.target_audience}
          <div class="flex items-baseline gap-3 px-5 py-1.5">
            <span class="text-[13px] font-semibold text-muted-foreground whitespace-nowrap shrink-0 w-16">Público</span>
            <span class="text-[15px] text-text-secondary flex-1">{client.target_audience}</span>
          </div>
        {/if}
        {#if client.brand_vibe}
          <div class="flex items-baseline gap-3 px-5 py-1.5">
            <span class="text-[13px] font-semibold text-muted-foreground whitespace-nowrap shrink-0 w-16">Estilo</span>
            <span class="text-[15px] text-text-secondary flex-1 italic">{client.brand_vibe}</span>
          </div>
        {/if}
        {#if client.quirks}
          <div class="flex items-baseline gap-3 px-5 py-1.5">
            <span class="text-[13px] font-semibold text-muted-foreground whitespace-nowrap shrink-0 w-16">Obs.</span>
            <span class="text-[15px] text-text-secondary flex-1 whitespace-pre-wrap">{client.quirks}</span>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Suggestions -->
    {#if suggestions.length > 0}
      <Button
        onclick={ontogglesuggestions}
        variant="ghost"
        class="w-full flex items-center justify-between px-5 pt-3.5 pb-2 bg-[var(--bg)] border-t border-border rounded-none min-h-0 h-auto"
      >
        <div class="flex items-center gap-2">
          <span class="text-[13px] font-bold tracking-wider uppercase text-sage-dark">Sugestões de perfil</span>
          <span class="text-xs font-bold px-1.5 py-px rounded-full bg-sage text-white">{suggestions.length}</span>
        </div>
        <span class="text-[13px] text-muted-foreground">{suggestionsOpen ? '▴' : '▾'}</span>
      </Button>
      {#if suggestionsOpen}
        <div class="bg-[var(--surface)]">
          {#each suggestions as sug (sug.id)}
            <div class="px-5 py-3 border-b border-border">
              <span class="text-[13px] font-bold uppercase tracking-wide text-muted-foreground">
                {sug.field === 'services' ? 'Serviço detectado' : sug.field === 'quirks' ? 'Diferencial detectado' : sug.field === 'target_audience' ? 'Público detectado' : 'Estilo detectado'}
              </span>
              <p class="text-[15px] text-text-secondary my-1 leading-relaxed">{sug.suggestion}</p>
              <div class="flex gap-2">
                <Button
                  onclick={() => onacceptsuggestion(sug)}
                  variant="secondary"
                  size="sm"
                  class="font-semibold border border-sage-light"
                >Adicionar</Button>
                <Button
                  onclick={() => ondismisssuggestion(sug)}
                  variant="outline"
                  size="sm"
                  class="text-muted-foreground"
                >Ignorar</Button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Recent posts -->
    {#if clientPosts.length > 0}
      <span class="text-[13px] font-bold tracking-wider uppercase text-muted-foreground px-5 pt-3.5 pb-2 bg-[var(--bg)] border-t border-border block">Posts recentes</span>
      <div class="bg-[var(--surface)]">
        {#each clientPosts.slice(0, historyLimit) as post (post.id)}
          {@const postExpanded = expandedPosts.has(post.id)}
          <div class="px-5 py-3 border-b border-border flex gap-3 items-start">
            <span class="text-sm text-muted-foreground whitespace-nowrap pt-0.5 w-12 shrink-0">
              {new Date(post.created).toLocaleDateString("pt-BR", { day: "numeric", month: "short" })}
            </span>
            <div class="flex-1 min-w-0">
              <Button
                onclick={() => ontogglepost(post.id)}
                variant="ghost"
                class="text-[15px] text-text-secondary leading-relaxed text-left w-full min-h-0 h-auto p-0 whitespace-normal font-normal {postExpanded ? '' : 'line-clamp-2'}"
              >
                {post.caption}
              </Button>
              {#if !postExpanded}
                <span class="text-[13px] text-muted-foreground mt-0.5 block">ver mais</span>
              {/if}
              {#if postExpanded && post.production_note}
                <p class="text-sm italic mt-1 text-muted-foreground">{post.production_note}</p>
              {/if}
            </div>
            <Button
              onclick={() => oncopypost(post.caption + (post.hashtags?.length ? '\n\n' + post.hashtags.join(' ') : ''))}
              variant="soft"
              size="sm"
              class="shrink-0 border border-coral-light whitespace-nowrap"
            >Copiar</Button>
          </div>
        {/each}
        {#if clientPosts.length > historyLimit}
          <Button
            onclick={onshowallposts}
            variant="ghost"
            size="sm"
            class="w-full text-left px-5 text-coral rounded-none justify-start"
          >
            Ver todos ({clientPosts.length})
          </Button>
        {/if}
      </div>
    {/if}

    <!-- Nudge / reminder -->
    {#if nudgeTier || nudgeText}
      <span
        class="text-[13px] font-bold tracking-wider uppercase px-5 pt-3.5 pb-2 block border-t {nudgeTier ? 'bg-[#FFF7ED] border-[#FDE68A] text-[#92400E]' : 'bg-[var(--bg)] border-border text-muted-foreground'}"
      >
        {nudgeTier ? `⚠ Lembrete — ${health?.daysSinceMsg} dias sem mensagem` : 'Mensagem'}
      </span>
      <div class="px-5 py-3 pb-4 border-b border-border {nudgeTier ? 'bg-[#FFFBEB]' : 'bg-[var(--surface)]'}">
        <Textarea
          value={nudgeText}
          oninput={(e) => onnudgetextchange(e.currentTarget.value)}
          rows={2}
          class="w-full rounded-xl text-base border border-[var(--border-strong)] px-4 py-3 bg-[var(--surface)] text-foreground"
        />
        <div class="flex items-center gap-2 mt-2">
          <Button
            onclick={onsendnudge}
            disabled={sendingNudge || !nudgeText.trim() || !!blockReason}
            title={blockReason ?? undefined}
            variant="whatsapp"
            class="text-[15px] font-semibold"
          >
            {sendingNudge ? "Enviando..." : "Enviar lembrete"}
          </Button>
          {#if blockReason && nudgeText.trim()}
            <span class="text-sm text-muted-foreground">{blockReason}</span>
          {/if}
          {#if sendNudgeError}
            <span class="text-sm text-destructive">{sendNudgeError}</span>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Upcoming dates -->
    {#if upcomingDates.length > 0}
      <span class="text-[13px] font-bold tracking-wider uppercase text-muted-foreground px-5 pt-3.5 pb-2 bg-[var(--bg)] border-t border-border block">Datas próximas</span>
      <div class="bg-[var(--surface)] px-5 py-2 pb-4 flex flex-wrap gap-2">
        {#each upcomingDates as sd}
          <Button
            onclick={() => onprefillseasonal(sd.template)}
            variant="outline"
            size="sm"
            class="text-text-secondary"
          >
            {sd.label} · {sd.daysUntil}d
          </Button>
        {/each}
      </div>
    {/if}

    <!-- Danger zone -->
    {#if client.invite_status === 'active'}
      <div class="mx-5 my-4 mb-6 p-4 rounded-2xl border border-[#FCA5A5] bg-[#FFF5F5]">
        <p class="text-sm leading-relaxed mb-2.5 text-[#991B1B]">
          Cancelar a assinatura desativa o acesso de {client.name} ao rekan.
        </p>
        <Button
          onclick={oncancelsubscription}
          disabled={cancelling}
          variant="outline"
          size="sm"
          class="border-[1.5px] border-[#EF4444] text-[#EF4444]"
        >
          {cancelling ? "Cancelando..." : "Cancelar assinatura"}
        </Button>
      </div>
    {/if}
  </div>
</div>
