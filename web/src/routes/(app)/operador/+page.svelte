<script lang="ts">
  import QRCode from "qrcode";
  import { onDestroy, onMount } from "svelte";
  import { pb } from "$lib/pb";
  import { onResume, readSSE } from "$lib/sse";
  import type {
    Business,
    GeneratedPost,
    Message,
    Post,
    ProfileSuggestion,
    ScheduledMessage,
    Service,
    WAStatus,
  } from "$lib/types";

  const BUSINESS_TYPES = [
    "Salão de Beleza",
    "Restaurante",
    "Personal Trainer",
    "Nail Designer",
    "Confeitaria",
    "Barbearia",
    "Loja de Roupas",
    "Pet Shop",
    "Banda Musical",
    "Estúdio de Tatuagem",
    "Hamburgueria",
    "Loja de Açaí",
    "Outro",
  ];

  const STATES = [
    "AC","AL","AP","AM","BA","CE","DF","ES","GO","MA","MT","MS","MG",
    "PA","PB","PR","PE","PI","RJ","RN","RS","RO","RR","SC","SP","SE","TO",
  ];

  const NUDGE_TEMPLATES = [
    {
      minDays: 5,
      maxDays: 7,
      template: "Oi {name}, como foi a semana? Tem algo legal pra gente postar?",
    },
    {
      minDays: 8,
      maxDays: 14,
      template: "{name}, tudo bem? Faz um tempinho que a gente não posta. Bora preparar algo novo?",
    },
    {
      minDays: 15,
      maxDays: Infinity,
      template: "{name}, vi que faz um tempo! Quer retomar? Posso te mandar ideias de conteúdo pra essa semana.",
    },
  ];

  type SeasonalDate = {
    month: number;
    day: number;
    label: string;
    niches: string[];
    template: string;
  };
  // Moveable holidays (Carnaval, Páscoa, Dia das Mães) hardcoded for 2026
  const SEASONAL_DATES: SeasonalDate[] = [
    {
      month: 2, day: 14, label: "Carnaval",
      niches: ["Salão de Beleza", "Barbearia", "Personal Trainer", "Nail Designer"],
      template: "{name}, Carnaval tá chegando! Vamos preparar posts especiais?",
    },
    {
      month: 3, day: 8, label: "Dia da Mulher",
      niches: ["Salão de Beleza", "Nail Designer", "Confeitaria", "Loja de Roupas"],
      template: "{name}, Dia da Mulher vem ai! Que tal um post com promo especial?",
    },
    {
      month: 4, day: 5, label: "Páscoa",
      niches: ["Confeitaria", "Restaurante", "Hamburgueria", "Loja de Açaí"],
      template: "{name}, Páscoa tá chegando! Vamos montar os posts das encomendas?",
    },
    {
      month: 5, day: 10, label: "Dia das Mães",
      niches: ["Salão de Beleza", "Confeitaria", "Nail Designer", "Loja de Roupas", "Restaurante"],
      template: "{name}, Dia das Mães daqui a pouco! Bora preparar posts de presente e promo?",
    },
    {
      month: 6, day: 12, label: "Dia dos Namorados",
      niches: ["Confeitaria", "Restaurante", "Hamburgueria", "Salão de Beleza", "Loja de Roupas"],
      template: "{name}, Dia dos Namorados vem ai! Vamos criar posts romanticos pro seu negocio?",
    },
    {
      month: 6, day: 13, label: "Festas Juninas",
      niches: ["Confeitaria", "Restaurante", "Hamburgueria", "Banda Musical"],
      template: "{name}, Junho ta ai! Vamos postar algo com tema junino?",
    },
    {
      month: 9, day: 1, label: "Dia do Educador Físico",
      niches: ["Personal Trainer"],
      template: "{name}, vem ai o Dia do Educador Fisico! Bora fazer um post especial?",
    },
    {
      month: 10, day: 1, label: "Início do Verão",
      niches: ["Personal Trainer", "Loja de Açaí"],
      template: "{name}, verao chegando! Momento perfeito pra postar sobre preparacao e resultados.",
    },
    {
      month: 10, day: 12, label: "Dia das Crianças",
      niches: ["Confeitaria", "Pet Shop", "Loja de Roupas"],
      template: "{name}, Dia das Criancas ta perto! Vamos criar posts com ofertas kids?",
    },
    {
      month: 12, day: 19, label: "Dia do Cabeleireiro",
      niches: ["Salão de Beleza", "Barbearia"],
      template: "{name}, Dia do Cabeleireiro chegando! Que tal um post especial celebrando a profissao?",
    },
    {
      month: 12, day: 25, label: "Natal",
      niches: [],
      template: "{name}, Natal chegando! Vamos preparar posts com ofertas e mensagem de final de ano?",
    },
    {
      month: 12, day: 31, label: "Réveillon",
      niches: ["Salão de Beleza", "Barbearia", "Nail Designer", "Personal Trainer", "Loja de Roupas"],
      template: "{name}, Réveillon vem aí! Bora postar sobre agendamento e preparação?",
    },
  ];
  const SEASONAL_DATES_SORTED = [...SEASONAL_DATES].sort((a, b) =>
    a.month !== b.month ? a.month - b.month : a.day - b.day
  );

  let clients = $state<Business[]>([]);
  let selectedId = $state<string | null>(null);
  let mobileView = $state<'list' | 'detail' | 'info'>('list');
  let loading = $state(true);

  // WhatsApp status
  let waConnected = $state(false);
  let waQR = $state("");
  let waChecking = $state(true);

  // Messages
  let messages = $state<Message[]>([]);
  let messagesLoading = $state(false);
  let unsubscribeMessages: (() => void) | null = null;
  let unsubscribeBusinesses: (() => void) | null = null;
  let unsubscribePosts: (() => void) | null = null;
  let unsubscribeScheduledMessages: (() => void) | null = null;
  let threadEl = $state<HTMLDivElement | null>(null);

  // Client form
  let showForm = $state(false);
  let editingId = $state<string | null>(null);
  let formName = $state("");
  let formType = $state("");
  let formCity = $state("");
  let formState = $state("");
  let formPhone = $state("");
  let formClientName = $state("");
  let formClientEmail = $state("");
  let formServices: Service[] = $state([{ name: "", price_brl: 0 }]);
  let formTargetAudience = $state("");
  let formBrandVibe = $state("");
  let formQuirks = $state("");
  let formError = $state("");
  let formSaving = $state(false);

  // Voice profile intake
  type VoiceMode = 'idle' | 'recording' | 'analyzing' | 'done' | 'manual';
  let voiceMode = $state<VoiceMode>('idle');
  let voiceError = $state('');
  let recordingSeconds = $state(0);
  let aiFilledFields = $state(new Set<string>());
  let mediaRecorderRef: MediaRecorder | null = null;
  let recordingChunks: Blob[] = [];
  let recordingTimer: ReturnType<typeof setInterval> | null = null;

  // Invite
  let inviteUrl = $state("");
  let inviteCopied = $state(false);
  let cancelling = $state(false);

  // Generation
  let message = $state("");
  let generating = $state(false);
  let generateError = $state("");
  let result = $state<GeneratedPost | null>(null);
  let copied = $state<Record<string, boolean>>({});
  let sending = $state(false);
  let sendError = $state("");

  // Input mode: 'chat' for quick reply, 'generate' for post generation
  let inputMode = $state<'chat' | 'generate'>('chat');
  let sendingQuick = $state(false);
  let quickReplyError = $state("");

  // Wave 2 — message selection for generation
  let selectedMessages = $state(new Set<string>());

  // Wave 3 — post review overlay
  let editingCaption = $state("");
  let showReviewOverlay = $state(false);

  // Feature 4 — post history
  let historyLimit = $state(10);
  let expandedPosts = $state(new Set<string>());

  // Feature 7 — proactive idea generation
  let ideaDrafts = $state<GeneratedPost[] | null>(null);
  let generatingIdeas = $state(false);
  let ideaError = $state("");
  let isProactive = $state(false);

  // Wave 4 — multi-select ideas
  let selectedIdeas = $state(new Set<number>());
  let sendingIdeas = $state(false);

  // Wave 5 — attach button (camera/gallery)
  let showAttachMenu = $state(false);
  let sendingMedia = $state(false);
  let attachedFile = $state<File | null>(null);
  let attachedPreview = $state<string>("");
  let toastMessage = $state("");
  let toastTimer: ReturnType<typeof setTimeout> | null = null;

  let scheduledMessages = $state<ScheduledMessage[]>([]);
  let scheduledMessageCount = $derived(scheduledMessages.length);
  let showApprovalPanel = $state(false);
  let approvingId = $state<string | null>(null);
  let dismissingId = $state<string | null>(null);

  // Profile suggestions (Wave 3)
  let suggestions = $state<ProfileSuggestion[]>([]);
  let suggestionCounts = $state<Record<string, number>>({});
  let suggestionsOpen = $state(false);
  let unsubscribeSuggestions: (() => void) | null = null;

  // Nudge / engagement
  let clientFilter = $state<"todos" | "inativos" | "com_mensagens" | "sazonal" | "cobranca">("todos");
  let nudgeText = $state("");
  let sendingNudge = $state(false);
  let sendNudgeError = $state("");


  // Posts (for health indicators)
  let posts = $state<Post[]>([]);

  let selected = $derived(clients.find((c) => c.id === selectedId) ?? null);

  // Unread message counts per business
  let lastSeen = $state<Record<string, string>>({});
  let unreadCounts = $derived.by(() => {
    const counts: Record<string, number> = {};
    for (const client of clients) {
      const seen = lastSeen[client.id];
      if (!seen) {
        counts[client.id] = messages.filter(
          (m) => m.business === client.id && m.direction === "incoming",
        ).length;
      } else {
        counts[client.id] = messages.filter(
          (m) =>
            m.business === client.id &&
            m.direction === "incoming" &&
            m.created > seen,
        ).length;
      }
    }
    return counts;
  });

  // Messages for selected client
  let threadMessages = $derived(
    selectedId
      ? messages
          .filter((m) => m.business === selectedId)
          .sort((a, b) => a.created.localeCompare(b.created))
      : [],
  );

  // IDs of recent incoming messages (all incoming since last outgoing)
  let recentContextIds = $derived.by(() => {
    const lastOut = [...threadMessages].reverse().find(m => m.direction === 'outgoing');
    const cutoff = lastOut
      ? lastOut.created
      : new Date(Date.now() - 86400000).toISOString();
    return new Set(
      threadMessages
        .filter(m => m.direction === 'incoming' && m.content && m.created > cutoff)
        .map(m => m.id)
    );
  });

  type MessageGroup = { date: Date; label: string; msgs: Message[] };
  let groupedMessages = $derived.by(() => {
    if (threadMessages.length === 0) return [];
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const yesterday = new Date(today.getTime() - 86400000);
    const groups: MessageGroup[] = [];
    let current: MessageGroup | null = null;
    for (const msg of threadMessages) {
      const d = new Date(msg.wa_timestamp || msg.created);
      d.setHours(0, 0, 0, 0);
      if (!current || current.date.getTime() !== d.getTime()) {
        let label: string;
        if (d.getTime() === today.getTime()) label = "Hoje";
        else if (d.getTime() === yesterday.getTime()) label = "Ontem";
        else
          label = d.toLocaleDateString("pt-BR", {
            weekday: "short",
            day: "numeric",
            month: "short",
          });
        current = { date: d, label, msgs: [] };
        groups.push(current);
      }
      current.msgs.push(msg);
    }
    return groups;
  });

  // Health indicators per client
  type ClientHealth = {
    daysSinceMsg: number;
    postsThisMonth: number;
    color: string;
  };
  let clientHealth = $derived.by(() => {
    const now = Date.now();
    const monthStart = new Date();
    monthStart.setDate(1);
    monthStart.setHours(0, 0, 0, 0);
    const monthStr = monthStart.toISOString();

    const health: Record<string, ClientHealth> = {};
    for (const client of clients) {
      const clientMsgs = messages.filter(
        (m) => m.business === client.id && m.direction === "incoming",
      );
      const lastMsg =
        clientMsgs.length > 0
          ? clientMsgs.reduce((a, b) => (a.created > b.created ? a : b))
          : null;
      const daysSinceMsg = lastMsg
        ? Math.floor(
            (now -
              new Date(lastMsg.wa_timestamp || lastMsg.created).getTime()) /
              86400000,
          )
        : 999;

      const postsThisMonth = posts.filter(
        (p) => p.business === client.id && p.created >= monthStr,
      ).length;

      let color = "#10B981"; // green
      if (daysSinceMsg >= 10)
        color = "#EF4444"; // red
      else if (daysSinceMsg >= 5) color = "#F59E0B"; // yellow

      health[client.id] = { daysSinceMsg, postsThisMonth, color };
    }
    return health;
  });

  // Feature 4 — posts for selected client
  let clientPosts = $derived(selectedId ? posts.filter(p => p.business === selectedId) : []);

  // Sort clients by urgency (red first, then yellow, then green)
  let sortedClients = $derived(
    [...clients].sort((a, b) => {
      const ha = clientHealth[a.id];
      const hb = clientHealth[b.id];
      if (!ha || !hb) return 0;
      return hb.daysSinceMsg - ha.daysSinceMsg;
    }),
  );

  let inactiveCount = $derived(
    clients.filter((c) => {
      const h = clientHealth[c.id];
      return h && h.daysSinceMsg >= 5;
    }).length,
  );

  // Feature 6 — morning summary derived values
  let unreadClientsCount = $derived(clients.filter(c => (unreadCounts[c.id] ?? 0) > 0).length);
  let pendingPaymentCount = $derived(clients.filter(c => c.charge_pending).length);
  let globalNearestSeasonal = $derived.by(() => {
    const now = new Date();
    const limit = new Date(now.getTime() + 30 * 86400000);
    const year = now.getFullYear();
    for (const sd of SEASONAL_DATES_SORTED) {
      const date = new Date(year, sd.month - 1, sd.day);
      if (date < now) date.setFullYear(year + 1);
      if (date > limit) continue;
      const eligible = clients.filter(c => sd.niches.length === 0 || sd.niches.includes(c.type));
      if (eligible.length > 0) {
        return { ...sd, daysUntil: Math.ceil((date.getTime() - now.getTime()) / 86400000), eligibleCount: eligible.length };
      }
    }
    return null;
  });

  let filteredClients = $derived.by(() => {
    switch (clientFilter) {
      case "inativos":
        return sortedClients.filter((c) => {
          const h = clientHealth[c.id];
          return h && h.daysSinceMsg >= 5;
        });
      case "com_mensagens":
        return sortedClients.filter(c => (unreadCounts[c.id] ?? 0) > 0);
      case "sazonal":
        if (!globalNearestSeasonal) return sortedClients;
        return sortedClients.filter(c =>
          globalNearestSeasonal!.niches.length === 0 || globalNearestSeasonal!.niches.includes(c.type)
        );
      case "cobranca":
        return sortedClients.filter(c => c.charge_pending);
      default:
        return sortedClients;
    }
  });

  let nudgeTier = $derived.by(() => {
    if (!selected) return null;
    const h = clientHealth[selected.id];
    if (!h || h.daysSinceMsg < 5 || h.daysSinceMsg === 999) return null;
    return (
      NUDGE_TEMPLATES.find(
        (t) => h.daysSinceMsg >= t.minDays && h.daysSinceMsg <= t.maxDays,
      ) ?? NUDGE_TEMPLATES[NUDGE_TEMPLATES.length - 1]
    );
  });

  let upcomingDates = $derived.by(() => {
    if (!selected) return [];
    const now = new Date();
    const limit = new Date(now.getTime() + 30 * 86400000);
    const year = now.getFullYear();

    return SEASONAL_DATES.filter((d) => {
      if (d.niches.length > 0 && !d.niches.includes(selected!.type))
        return false;
      const date = new Date(year, d.month - 1, d.day);
      if (date < now) date.setFullYear(year + 1);
      return date >= now && date <= limit;
    })
      .map((d) => {
        const date = new Date(year, d.month - 1, d.day);
        if (date < now) date.setFullYear(year + 1);
        const daysUntil = Math.ceil(
          (date.getTime() - now.getTime()) / 86400000,
        );
        return { ...d, daysUntil };
      })
      .sort((a, b) => a.daysUntil - b.daysUntil);
  });

  let blockReason = $derived(
    !waConnected
      ? "WhatsApp desconectado"
      : !selected?.phone
        ? "Cliente sem telefone cadastrado"
        : null,
  );

  // Feature 7 — show generate ideas button condition
  let showGenerateIdeasButton = $derived(
    !!selected &&
    ideaDrafts === null &&
    result === null &&
    !generating &&
    !generatingIdeas &&
    ((clientHealth[selectedId!]?.daysSinceMsg ?? 999) >= 5 || threadMessages.length === 0)
  );


  function handlePopState(event: PopStateEvent) {
    const view = event.state?.mobileView;
    if (view === 'detail' || view === 'info') {
      mobileView = view;
    } else {
      mobileView = 'list';
      selectedId = null;
    }
  }

  onMount(async () => {
    window.addEventListener("popstate", handlePopState);

    lastSeen = JSON.parse(
      localStorage.getItem("rekan_operator_last_seen") ?? "{}",
    );

    const [clientsRes] = await Promise.all([
      pb.collection("businesses").getList<Business>(1, 200, { sort: "name" }),
    ]);
    clients = clientsRes.items;
    loading = false;

    // Load all messages and posts
    await Promise.all([loadMessages(), loadPosts()]);

    // Load scheduled messages count and suggestion badges in parallel
    await Promise.all([loadScheduledMessages(), loadAllSuggestionCounts()]);

    // Subscribe to realtime updates
    unsubscribeMessages = await pb
      .collection("messages")
      .subscribe<Message>("*", (e) => {
        if (e.action === "create") {
          messages = [...messages, e.record];
        } else if (e.action === "update") {
          messages = messages.map((m) => (m.id === e.record.id ? e.record : m));
        }
      });

    unsubscribeBusinesses = await pb
      .collection("businesses")
      .subscribe<Business>("*", (e) => {
        if (e.action === "create") {
          if (!clients.some((c) => c.id === e.record.id)) {
            clients = [...clients, e.record].sort((a, b) =>
              a.name.localeCompare(b.name),
            );
          }
        } else if (e.action === "update") {
          clients = clients.map((c) => (c.id === e.record.id ? e.record : c));
        } else if (e.action === "delete") {
          clients = clients.filter((c) => c.id !== e.record.id);
        }
      });

    unsubscribePosts = await pb
      .collection("posts")
      .subscribe<Post>("*", (e) => {
        if (e.action === "create") {
          if (!posts.some((p) => p.id === e.record.id)) {
            posts = [e.record, ...posts];
          }
        } else if (e.action === "update") {
          posts = posts.map((p) => (p.id === e.record.id ? e.record : p));
        } else if (e.action === "delete") {
          posts = posts.filter((p) => p.id !== e.record.id);
        }
      });

    unsubscribeScheduledMessages = await pb
      .collection("scheduled_messages")
      .subscribe<ScheduledMessage>("*", async () => {
        await loadScheduledMessages();
      });

    unsubscribeSuggestions = await pb
      .collection("profile_suggestions")
      .subscribe<ProfileSuggestion>("*", (e) => {
        if (e.action === "create" && !e.record.dismissed) {
          const biz = e.record.business;
          suggestionCounts = { ...suggestionCounts, [biz]: (suggestionCounts[biz] ?? 0) + 1 };
          if (biz === selectedId) {
            suggestions = [...suggestions, e.record];
          }
        } else if (e.action === "update" && e.record.dismissed) {
          decrementSuggestionCount(e.record.business);
          suggestions = suggestions.filter((s) => s.id !== e.record.id);
        }
      });

    connectWhatsAppStream();

    // SSE stream dies when the browser suspends the page; restart on resume
    cleanupVisibility = onResume(() => {
      waAbortController?.abort();
      waChecking = true;
      connectWhatsAppStream();
      loadMessages();
      loadPosts();
      loadScheduledMessages();
      loadAllSuggestionCounts();
    });
  });

  let cleanupVisibility: (() => void) | null = null;
  let waAbortController: AbortController | null = null;
  let waDisconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let qrDataUrl = $state("");

  $effect(() => {
    if (waQR) {
      QRCode.toDataURL(waQR, { width: 256, margin: 2 }).then((url: string) => {
        qrDataUrl = url;
      });
    } else {
      qrDataUrl = "";
    }
  });

  $effect(() => {
    const _ = [threadMessages.length, selectedId];
    if (threadEl) threadEl.scrollTop = threadEl.scrollHeight;
  });

  $effect(() => {
    suggestionsOpen = false;
    if (selectedId) {
      loadSuggestions(selectedId);
    } else {
      suggestions = [];
    }
  });

  $effect(() => {
    if (result) {
      editingCaption = result.caption;
      showReviewOverlay = true;
    } else {
      showReviewOverlay = false;
    }
  });

  onDestroy(() => {
    window.removeEventListener("popstate", handlePopState);
    cleanupVisibility?.();
    unsubscribeMessages?.();
    unsubscribeBusinesses?.();
    unsubscribePosts?.();
    unsubscribeScheduledMessages?.();
    unsubscribeSuggestions?.();
    waAbortController?.abort();
    if (waDisconnectTimer) { clearTimeout(waDisconnectTimer); waDisconnectTimer = null; }
    removeAttachment();
  });

  async function connectWhatsAppStream() {
    waAbortController = new AbortController();
    try {
      const res = await fetch(`${pb.baseUrl}/api/whatsapp/stream`, {
        headers: { Authorization: pb.authStore.token },
        signal: waAbortController.signal,
      });
      if (!res.body) return;
      await readSSE(res.body, (data) => {
        const s = data as WAStatus;
        if (waDisconnectTimer) { clearTimeout(waDisconnectTimer); waDisconnectTimer = null; }
        waConnected = s.connected;
        waQR = s.qr ?? "";
        waChecking = false;
      });
    } catch (err) {
      if (err instanceof Error && err.name === "AbortError") return;
      // Grace period: wait 5s before showing disconnected banner
      if (waDisconnectTimer) clearTimeout(waDisconnectTimer);
      waDisconnectTimer = setTimeout(() => {
        waConnected = false;
        waDisconnectTimer = null;
      }, 5000);
      waQR = "";
      waChecking = false;
    }
  }

  async function loadMessages() {
    messagesLoading = true;
    try {
      const res = await pb.collection("messages").getList<Message>(1, 500, {
        sort: "created",
      });
      messages = res.items;
    } finally {
      messagesLoading = false;
    }
  }

  async function loadPosts() {
    try {
      const res = await pb.collection("posts").getList<Post>(1, 500, {
        sort: "-created",
      });
      posts = res.items;
    } catch {
      // Posts loading is non-critical for the page to work
    }
  }

  async function loadScheduledMessages() {
    try {
      const res = await pb.send("/api/scheduled-messages", { method: "GET" });
      scheduledMessages = res as ScheduledMessage[];
    } catch {
      // non-critical
    }
  }

  async function loadAllSuggestionCounts() {
    try {
      const res = await pb.collection("profile_suggestions").getList<ProfileSuggestion>(1, 500, {
        filter: "dismissed = false",
        fields: "business",
      });
      const counts: Record<string, number> = {};
      for (const s of res.items) {
        counts[s.business] = (counts[s.business] ?? 0) + 1;
      }
      suggestionCounts = counts;
    } catch {
      // non-critical
    }
  }

  async function loadSuggestions(businessId: string) {
    try {
      const res = await pb.collection("profile_suggestions").getList<ProfileSuggestion>(1, 50, {
        filter: `business = "${businessId}" && dismissed = false`,
        sort: "created",
      });
      if (selectedId !== businessId) return;
      suggestions = res.items;
    } catch {
      if (selectedId === businessId) suggestions = [];
    }
  }

  function decrementSuggestionCount(businessId: string) {
    suggestionCounts = {
      ...suggestionCounts,
      [businessId]: Math.max(0, (suggestionCounts[businessId] ?? 1) - 1),
    };
  }

  async function acceptSuggestion(sug: ProfileSuggestion) {
    const business = clients.find((c) => c.id === sug.business);
    if (!business) return;

    const update: Record<string, unknown> = {};
    if (sug.field === "services") {
      const parts = sug.suggestion.split("|");
      const name = parts[0]?.trim() ?? sug.suggestion;
      const price = parseFloat(parts[1] ?? "0") || 0;
      update.services = [...(business.services ?? []), { name, price_brl: price }];
    } else if (sug.field === "quirks") {
      const existing = business.quirks ?? "";
      update.quirks = existing ? existing + "\n" + sug.suggestion : sug.suggestion;
    } else if (sug.field === "target_audience") {
      const existing = business.target_audience ?? "";
      update.target_audience = existing ? existing + ", " + sug.suggestion : sug.suggestion;
    } else if (sug.field === "brand_vibe") {
      const existing = business.brand_vibe ?? "";
      update.brand_vibe = existing ? existing + ", " + sug.suggestion : sug.suggestion;
    } else {
      return;
    }

    await Promise.all([
      pb.collection("businesses").update<Business>(business.id, update),
      pb.collection("profile_suggestions").update(sug.id, { dismissed: true }),
    ]);

    clients = clients.map((c) => (c.id === business.id ? ({ ...c, ...update } as Business) : c));
    suggestions = suggestions.filter((s) => s.id !== sug.id);
    decrementSuggestionCount(sug.business);
  }

  async function dismissSuggestion(sug: ProfileSuggestion) {
    await pb.collection("profile_suggestions").update(sug.id, { dismissed: true });
    suggestions = suggestions.filter((s) => s.id !== sug.id);
    decrementSuggestionCount(sug.business);
  }

  function selectClient(id: string) {
    selectedId = id;
    mobileView = 'detail';
    history.pushState({ mobileView: 'detail' }, '');
    result = null;
    generateError = "";
    sendNudgeError = "";
    historyLimit = 10;
    expandedPosts = new Set();
    ideaDrafts = null;
    ideaError = "";
    isProactive = false;
    selectedIdeas = new Set();
    sendingIdeas = false;
    inputMode = 'chat';
    message = "";
    quickReplyError = "";
    selectedMessages = new Set();
    showAttachMenu = false;
    removeAttachment();

    // Mark as seen
    lastSeen = { ...lastSeen, [id]: new Date().toISOString() };
    localStorage.setItem("rekan_operator_last_seen", JSON.stringify(lastSeen));

    // Auto-populate nudge text for inactive clients
    const client = clients.find((c) => c.id === id);
    const health = clientHealth[id];
    if (client && health && health.daysSinceMsg >= 5 && health.daysSinceMsg !== 999) {
      const tier =
        NUDGE_TEMPLATES.find(
          (t) =>
            health.daysSinceMsg >= t.minDays &&
            health.daysSinceMsg <= t.maxDays,
        ) ?? NUDGE_TEMPLATES[NUDGE_TEMPLATES.length - 1];
      nudgeText = tier.template.replace("{name}", client.client_name ? client.client_name.split(" ")[0] : client.name);
    } else {
      nudgeText = "";
    }

  }

  // Wave 2 — select recent incoming messages for generation
  function selectRecentMessages() {
    selectedMessages = new Set(recentContextIds);
  }

  // Wave 2 — toggle a message in selectedMessages
  function toggleMessageSelection(id: string) {
    const next = new Set(selectedMessages);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selectedMessages = next;
  }

  // Wave 2 — build generation payload from selected messages
  function buildSelectedContent(): string {
    const parts: string[] = [];
    for (const msg of threadMessages) {
      if (!selectedMessages.has(msg.id)) continue;
      if (msg.type === 'image') {
        parts.push(msg.content || '[Imagem]');
      } else if (msg.type === 'video') {
        parts.push(msg.content || '[Video]');
      } else if (msg.content) {
        parts.push(msg.content);
      }
    }
    if (message.trim()) parts.push(message.trim());
    return parts.join('\n');
  }

  // Quick reply (chat mode), sends media if attached
  async function sendQuickReply() {
    if (!selectedId || (!message.trim() && !attachedFile)) return;
    if (attachedFile) {
      await sendMedia(attachedFile);
      return;
    }
    sendingQuick = true;
    quickReplyError = "";
    try {
      await pb.send("/api/messages:send", {
        method: "POST",
        body: JSON.stringify({
          business_id: selectedId,
          caption: message.trim(),
          hashtags: "",
          production_note: "",
        }),
      });
      message = "";
    } catch {
      quickReplyError = "Erro ao enviar. Tente novamente.";
    } finally {
      sendingQuick = false;
    }
  }

  function showToast(msg: string) {
    toastMessage = msg;
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toastMessage = ""; }, 3000);
  }

  function attachFile(file: File) {
    attachedFile = file;
    attachedPreview = URL.createObjectURL(file);
    showAttachMenu = false;
  }

  function removeAttachment() {
    if (attachedPreview) URL.revokeObjectURL(attachedPreview);
    attachedFile = null;
    attachedPreview = "";
  }

  async function sendMedia(file: File) {
    if (!selectedId) return;
    sendingMedia = true;
    try {
      const form = new FormData();
      form.append("business_id", selectedId);
      form.append("file", file);
      form.append("caption", message.trim());
      await pb.send("/api/messages:sendMedia", { method: "POST", body: form });
      message = "";
      removeAttachment();
    } catch {
      showToast("Erro ao enviar midia. Tente novamente.");
    } finally {
      sendingMedia = false;
    }
  }

  async function describeAttachment(): Promise<string> {
    if (!attachedFile) return "";
    const form = new FormData();
    form.append("file", attachedFile);
    const res = await pb.send("/api/media:describe", { method: "POST", body: form }) as { description: string };
    return res.description;
  }

  function handleAttachFile(accept: string, capture?: string) {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = accept;
    if (capture) input.setAttribute("capture", capture);
    input.onchange = () => {
      const file = input.files?.[0];
      if (file) attachFile(file);
    };
    input.click();
  }

  function mediaUrl(msg: Message): string {
    return pb.files.getURL(
      { id: msg.id, collectionId: msg.collectionId },
      msg.media,
    );
  }

  function profilePictureUrl(business: Business): string | null {
    if (!business.profile_picture) return null;
    return pb.files.getURL(
      { id: business.id, collectionId: business.collectionId },
      business.profile_picture,
    );
  }

  function initials(business: Business): string {
    const name = business.client_name || business.name;
    return name.split(' ').slice(0, 2).map((w: string) => w[0]).join('').toUpperCase();
  }

  // --- Client form logic ---

  function resetForm() {
    formName = "";
    formType = "";
    formCity = "";
    formState = "";
    formPhone = "";
    formClientName = "";
    formClientEmail = "";
    formServices = [{ name: "", price_brl: 0 }];
    formTargetAudience = "";
    formBrandVibe = "";
    formQuirks = "";
    formError = "";
    inviteUrl = "";
    editingId = null;
    resetVoice();
  }

  function openNewForm() {
    resetForm();
    voiceMode = 'idle';
    showForm = true;
    if (mobileView === 'list') history.pushState({ mobileView: 'detail' }, '');
    mobileView = 'detail';
  }

  function openEditForm(biz: Business) {
    editingId = biz.id;
    formName = biz.name;
    formType = biz.type === "Desconhecido" ? "" : biz.type;
    formCity = biz.city === "-" ? "" : biz.city;
    formState = biz.state === "-" ? "" : biz.state;
    formPhone = biz.phone || "";
    formClientName = biz.client_name || "";
    formClientEmail = biz.client_email || "";
    formServices =
      biz.services?.length > 0
        ? [...biz.services]
        : [{ name: "", price_brl: 0 }];
    formTargetAudience = biz.target_audience || "";
    formBrandVibe = biz.brand_vibe || "";
    formQuirks = biz.quirks || "";
    formError = "";
    inviteUrl = "";
    voiceMode = 'manual';
    voiceError = '';
    aiFilledFields = new Set();
    showForm = true;
    if (mobileView === 'list') history.pushState({ mobileView: 'detail' }, '');
    mobileView = 'detail';
  }

  function closeForm() {
    showForm = false;
    resetForm();
    if (!selectedId) history.back();
  }

  function addService() {
    formServices = [...formServices, { name: "", price_brl: 0 }];
  }

  function removeService(i: number) {
    formServices = formServices.filter((_: Service, idx: number) => idx !== i);
  }

  function fmtTime(s: number): string {
    return `${Math.floor(s / 60)}:${(s % 60).toString().padStart(2, '0')}`;
  }

  function cancelRecording() {
    if (recordingTimer) { clearInterval(recordingTimer); recordingTimer = null; }
    // Set idle before stop so the onstop guard skips extraction
    voiceMode = 'idle';
    recordingSeconds = 0;
    recordingChunks = [];
    if (mediaRecorderRef && mediaRecorderRef.state !== 'inactive') mediaRecorderRef.stop();
    mediaRecorderRef = null;
  }

  function submitRecording() {
    if (recordingTimer) { clearInterval(recordingTimer); recordingTimer = null; }
    if (mediaRecorderRef && mediaRecorderRef.state !== 'inactive') {
      voiceMode = 'analyzing';
      mediaRecorderRef.stop();
    }
    mediaRecorderRef = null;
  }

  function resetVoice() {
    cancelRecording();
    voiceError = '';
    aiFilledFields = new Set();
    voiceMode = 'idle';
  }

  async function startVoiceRecording() {
    voiceError = '';
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const recorder = new MediaRecorder(stream);
      recordingChunks = [];
      recorder.ondataavailable = (e) => { if (e.data.size > 0) recordingChunks.push(e.data); };
      recorder.onstop = async () => {
        for (const t of stream.getTracks()) t.stop();
        const blob = new Blob(recordingChunks, { type: recorder.mimeType || 'audio/webm' });
        await extractVoiceProfile(blob, recorder.mimeType || 'audio/webm');
      };
      recorder.start(200);
      mediaRecorderRef = recorder;
      recordingSeconds = 0;
      recordingTimer = setInterval(() => recordingSeconds++, 1000);
      voiceMode = 'recording';
    } catch (err) {
      const name = err instanceof DOMException ? err.name : '';
      if (name === 'NotAllowedError' || name === 'PermissionDeniedError') {
        voiceError = 'Permissão do microfone negada. Permita o acesso nas configurações do navegador ou preencha os campos manualmente.';
      } else if (!navigator.mediaDevices) {
        voiceError = 'O microfone requer uma conexão segura (HTTPS). Preencha os campos manualmente.';
      } else {
        voiceError = 'Não foi possível acessar o microfone. Preencha os campos manualmente.';
      }
      voiceMode = 'manual';
    }
  }

  async function extractVoiceProfile(blob: Blob, mimeType: string) {
    if (voiceMode !== 'analyzing') return; // recording was cancelled
    try {
      const form = new FormData();
      form.append('audio', blob, 'recording.webm');
      form.append('business_type', formType || '');
      const res = await fetch(`${pb.baseUrl}/api/businesses/profile:extract`, {
        method: 'POST',
        headers: { Authorization: pb.authStore.token },
        body: form,
      });
      if (!res.ok) throw new Error('extract failed');
      const data = await res.json();

      const filled = new Set<string>();
      type ExtractedService = { name: string; price_brl: number | null };
      const hasServices = formServices.some((s: Service) => s.name.trim());
      if (data.services?.length) {
        const newServices = (data.services as ExtractedService[]).map((s) => ({
          name: s.name,
          price_brl: s.price_brl ?? 0,
        }));
        formServices = hasServices ? [...formServices, ...newServices] : newServices;
        filled.add('services');
      }
      if (data.target_audience && !formTargetAudience.trim()) {
        formTargetAudience = data.target_audience;
        filled.add('target_audience');
      }
      if (data.brand_vibe && !formBrandVibe.trim()) {
        formBrandVibe = data.brand_vibe;
        filled.add('brand_vibe');
      }
      if (data.quirks?.length && !formQuirks.trim()) {
        formQuirks = (data.quirks as string[]).join('\n');
        filled.add('quirks');
      }
      aiFilledFields = filled;
      voiceMode = 'done';
    } catch {
      voiceError = 'Não foi possível analisar o áudio. Preencha os campos manualmente.';
      voiceMode = 'manual';
    }
  }

  function validateForm(requireInviteFields: boolean): string | null {
    if (!formName.trim() || !formType || !formCity.trim() || !formState) {
      return "Preencha nome, tipo, cidade e estado.";
    }
    if (requireInviteFields) {
      if (!formClientName.trim()) return "Preencha o nome do cliente para enviar convite.";
      if (!formClientEmail.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formClientEmail.trim())) {
        return "Preencha um email válido para enviar convite.";
      }
    } else {
      if (formClientName.trim() && !formClientEmail.trim()) return "Preencha o email do cliente.";
      if (formClientEmail.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formClientEmail.trim())) {
        return "Email inválido.";
      }
    }
    const validServices = formServices.filter((s: Service) => s.name.trim());
    if (validServices.length === 0) return "Adicione pelo menos um serviço.";
    return null;
  }

  function buildFormData() {
    return {
      user: pb.authStore.record!.id,
      name: formName.trim(),
      type: formType,
      city: formCity.trim(),
      state: formState,
      phone: formPhone.trim(),
      client_name: formClientName.trim(),
      client_email: formClientEmail.trim(),
      services: formServices.filter((s: Service) => s.name.trim()),
      target_audience: formTargetAudience.trim(),
      brand_vibe: formBrandVibe.trim(),
      quirks: formQuirks.trim(),
    };
  }

  async function saveBusiness(): Promise<string | null> {
    const data = buildFormData();
    if (editingId) {
      const { user: _, ...updateData } = data;
      const updated = await pb
        .collection("businesses")
        .update<Business>(editingId, updateData);
      clients = clients.map((c) => (c.id === editingId ? updated : c));
      return editingId;
    }
    const created = await pb
      .collection("businesses")
      .create<Business>(data);
    clients = [...clients, created].sort((a, b) =>
      a.name.localeCompare(b.name),
    );
    selectedId = created.id;
    return created.id;
  }

  async function saveClient() {
    const error = validateForm(false);
    if (error) { formError = error; return; }
    formError = "";
    formSaving = true;
    try {
      await saveBusiness();
      closeForm();
    } catch {
      formError = "Erro ao salvar. Tente novamente.";
    } finally {
      formSaving = false;
    }
  }

  async function saveAndInvite() {
    const error = validateForm(true);
    if (error) { formError = error; return; }
    formError = "";
    formSaving = true;
    try {
      const bizId = await saveBusiness();
      editingId = bizId;

      const res = await pb.send(`/api/businesses/${bizId}/invites:send`, {
        method: "POST",
      });
      inviteUrl = res.invite_url || "";

      const refreshed = await pb.collection("businesses").getOne<Business>(bizId!);
      clients = clients.map((c) => (c.id === bizId ? refreshed : c));
    } catch {
      formError = "Erro ao enviar convite. Tente novamente.";
    } finally {
      formSaving = false;
    }
  }

  async function copyText(text: string) {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
    } else {
      const el = document.createElement("textarea");
      el.value = text;
      el.style.position = "fixed";
      el.style.opacity = "0";
      document.body.appendChild(el);
      el.select();
      document.execCommand("copy");
      document.body.removeChild(el);
    }
  }

  async function copyInviteUrl() {
    await copyText(inviteUrl);
    inviteCopied = true;
    setTimeout(() => { inviteCopied = false; }, 2000);
  }

  async function cancelSubscription() {
    if (!selected) return;
    if (!confirm(`Cancelar assinatura de ${selected.name}? Essa ação não pode ser desfeita.`)) return;
    cancelling = true;
    try {
      await pb.send(`/api/businesses/${selected.id}/authorization:cancel`, {
        method: "POST",
      });
      const refreshed = await pb.collection("businesses").getOne<Business>(selected.id);
      clients = clients.map((c) => (c.id === selected!.id ? refreshed : c));
    } catch {
      alert("Erro ao cancelar assinatura. Tente novamente.");
    } finally {
      cancelling = false;
    }
  }

  async function generate() {
    const hasSelected = selectedMessages.size > 0;
    const hasText = message.trim().length > 0;
    const hasAttachment = !!attachedFile;
    if (!selectedId || (!hasSelected && !hasText && !hasAttachment)) return;
    generating = true;
    generateError = "";
    result = null;
    try {
      const payload: Record<string, string> = {};

      // Describe attached image/video via Gemini
      let imageDescription = "";
      if (hasAttachment) {
        imageDescription = await describeAttachment();
      }

      // If exactly one message selected and no typed text and no attachment, pass message_id
      if (selectedMessages.size === 1 && !hasText && !hasAttachment) {
        const [id] = selectedMessages;
        payload.message_id = id;
        const msg = threadMessages.find(m => m.id === id);
        payload.message = msg?.content || '';
      } else {
        const parts: string[] = [];
        if (imageDescription) parts.push("[Foto do operador] " + imageDescription);
        const selected = buildSelectedContent();
        if (selected) parts.push(selected);
        payload.message = parts.join("\n\n");
      }
      const res = await pb.send(
        `/api/businesses/${selectedId}/posts:generateFromMessage`,
        {
          method: "POST",
          body: JSON.stringify(payload),
        },
      );
      result = res as GeneratedPost;
      removeAttachment();
    } catch (err: unknown) {
      const e = err as { data?: { message?: string } };
      generateError =
        e?.data?.message ?? "Erro ao gerar conteúdo. Tente novamente.";
    } finally {
      generating = false;
    }
  }

  // Feature 7 — generate 3 proactive ideas
  async function generateIdeas() {
    if (!selectedId) return;
    generatingIdeas = true;
    ideaError = "";
    try {
      const res = await pb.send(
        `/api/businesses/${selectedId}/posts:generateIdeas`,
        { method: "POST", body: JSON.stringify({}) },
      );
      ideaDrafts = res as GeneratedPost[];
    } catch {
      ideaError = "Erro ao gerar ideias. Tente novamente.";
    } finally {
      generatingIdeas = false;
    }
  }

  async function sendViaWhatsApp() {
    if (!selectedId || !result) return;
    sending = true;
    sendError = "";
    try {
      const caption = editingCaption || result.caption;
      // Feature 7 — if proactive, save the post first
      if (isProactive) {
        await pb.send(`/api/businesses/${selectedId}/posts:saveProactive`, {
          method: "POST",
          body: JSON.stringify({
            caption,
            hashtags: result.hashtags,
            production_note: result.production_note || "",
          }),
        });
      }

      await pb.send("/api/messages:send", {
        method: "POST",
        body: JSON.stringify({
          business_id: selectedId,
          caption,
          hashtags: result.hashtags.join(" "),
          production_note: result.production_note || "",
        }),
      });
      result = null;
      message = "";
      isProactive = false;
      ideaDrafts = null;
      selectedIdeas = new Set();
    } catch {
      sendError = "Erro ao enviar. Tente novamente.";
    } finally {
      sending = false;
    }
  }

  async function sendSelectedIdeas() {
    if (!selectedId || !ideaDrafts || selectedIdeas.size === 0) return;
    sendingIdeas = true;
    sendError = "";
    try {
      const indices = [...selectedIdeas].sort((a, b) => a - b);
      for (const idx of indices) {
        const draft = ideaDrafts[idx];
        await pb.send(`/api/businesses/${selectedId}/posts:saveProactive`, {
          method: "POST",
          body: JSON.stringify({
            caption: draft.caption,
            hashtags: draft.hashtags,
            production_note: draft.production_note || "",
          }),
        });
        await pb.send("/api/messages:send", {
          method: "POST",
          body: JSON.stringify({
            business_id: selectedId,
            caption: draft.caption,
            hashtags: draft.hashtags.join(" "),
            production_note: draft.production_note || "",
          }),
        });
      }
      ideaDrafts = null;
      selectedIdeas = new Set();
    } catch {
      sendError = "Erro ao enviar ideias. Tente novamente.";
    } finally {
      sendingIdeas = false;
    }
  }

  async function sendNudge() {
    if (!selectedId || !nudgeText.trim()) return;
    sendingNudge = true;
    sendNudgeError = "";
    try {
      await pb.send("/api/messages:send", {
        method: "POST",
        body: JSON.stringify({
          business_id: selectedId,
          caption: nudgeText.trim(),
          hashtags: "",
          production_note: "",
        }),
      });
      nudgeText = "";
    } catch {
      sendNudgeError = "Erro ao enviar lembrete. Tente novamente.";
    } finally {
      sendingNudge = false;
    }
  }


  function prefillSeasonalMessage(template: string) {
    if (!selected) return;
    nudgeText = template.replace("{name}", selected.client_name ? selected.client_name.split(" ")[0] : selected.name);
  }

  const copyTimers: Record<string, ReturnType<typeof setTimeout>> = {};
  async function copyWithFeedback(key: string, text: string) {
    await copyText(text);
    clearTimeout(copyTimers[key]);
    copied = { ...copied, [key]: true };
    copyTimers[key] = setTimeout(() => { copied = { ...copied, [key]: false }; }, 2000);
  }

  // Feature 8 — approval panel actions
  async function approveScheduledMessage(id: string) {
    approvingId = id;
    try {
      await pb.send(`/api/scheduled-messages/${id}/approve`, { method: "POST" });
      scheduledMessages = scheduledMessages.filter(m => m.id !== id);
    } catch {
      // silent — rare failure
    } finally {
      approvingId = null;
    }
  }

  async function dismissScheduledMessage(id: string) {
    dismissingId = id;
    try {
      await pb.send(`/api/scheduled-messages/${id}/dismiss`, { method: "POST" });
      scheduledMessages = scheduledMessages.filter(m => m.id !== id);
    } catch {
      // silent
    } finally {
      dismissingId = null;
    }
  }
</script>

<div class="h-dvh flex flex-col" style="background: var(--bg)">
  <header
    class="border-b px-5 py-4 flex items-center gap-3 shrink-0 {mobileView !== 'list' ? 'hidden md:flex' : ''}"
    style="background: var(--surface); border-color: var(--border)"
  >
    <span
      class="font-semibold text-lg md:text-base"
      style="color: var(--text); font-family: var(--font-primary)"
    >
      Rekan
    </span>
    <button
      onclick={openNewForm}
      style="padding: 6px 16px; border-radius: 9999px; font-size: 14px; font-weight: 600; background: var(--coral); color: #fff; margin-left: auto;"
    >+ Novo</button>
  </header>
  {#if !waConnected && !waChecking}
    <a
      href="/operador/whatsapp"
      class="shrink-0 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium"
      style="background: #FDE8E8; color: #9B1C1C;"
    >
      <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>
      WhatsApp desconectado — toque para reconectar
    </a>
  {/if}

  {#if loading}
    <p class="text-base p-6" style="color: var(--text-muted)">Já vou...</p>
  {:else}
    <!-- Main operator layout -->
    <main class="flex-1 flex flex-col md:flex-row overflow-hidden">
      <!-- Left: Client list (or approval panel) -->
      <div
        class="w-full md:w-80 md:border-r flex flex-col flex-1 min-h-0 md:flex-none {mobileView === 'list' ? '' : 'hidden md:flex'}"
        style="border-color: var(--border); background: var(--surface)"
      >
        {#if showApprovalPanel}
          <!-- Feature 8 — Seasonal messages approval panel -->
          <div class="flex-1 overflow-y-auto p-3 flex flex-col gap-2">
            <button
              onclick={() => { showApprovalPanel = false; }}
              style="min-height: 48px; padding: 0 4px; border-radius: 9999px; font-size: 14px; font-weight: 500; color: var(--coral); background: none; text-align: left; align-self: flex-start;"
            >← Clientes</button>
            {#if scheduledMessages.length === 0}
              <p class="text-base text-center py-8" style="color: var(--text-muted)">
                Tudo em dia! Nenhuma mensagem pra aprovar.
              </p>
            {:else}
              {#each scheduledMessages as msg (msg.id)}
                {@const biz = clients.find(c => c.id === msg.business)}
                <div
                  class="rounded-xl p-4"
                  style="background: var(--bg); border: 1px solid var(--border)"
                >
                  <p class="text-sm font-medium mb-1.5" style="color: var(--text-muted)">
                    {biz?.name ?? msg.business}
                  </p>
                  <p class="text-sm py-1.5" style="color: var(--text-secondary)">{msg.text}</p>
                  <div class="flex gap-2 mt-2">
                    <button
                      onclick={() => approveScheduledMessage(msg.id)}
                      disabled={approvingId === msg.id || !waConnected}
                      class="flex-1 py-3 rounded-lg text-sm font-medium transition-opacity"
                      style="background: #25D366; color: #fff; opacity: {approvingId === msg.id ? '0.6' : '1'}"
                    >
                      {approvingId === msg.id ? "..." : "Enviar"}
                    </button>
                    <button
                      onclick={() => dismissScheduledMessage(msg.id)}
                      disabled={dismissingId === msg.id}
                      class="flex-1 py-3 rounded-lg text-sm font-medium border transition-opacity"
                      style="border-color: var(--border-strong); color: var(--text-secondary); opacity: {dismissingId === msg.id ? '0.6' : '1'}"
                    >
                      {dismissingId === msg.id ? "..." : "Descartar"}
                    </button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        {:else}
          <!-- Feature 6 — Morning summary bar -->
          {#if unreadClientsCount > 0 || inactiveCount > 0 || globalNearestSeasonal || pendingPaymentCount > 0 || scheduledMessageCount > 0}
            <div style="border-bottom: 1px solid var(--border); background: var(--bg);">
              {#if unreadClientsCount > 0}
                <button
                  onclick={() => { clientFilter = "com_mensagens"; }}
                  style="display: flex; align-items: center; gap: 10px; width: 100%; min-height: 48px; padding: 0 20px; text-align: left; font-size: 14px; color: var(--text-secondary); border-bottom: 1px solid var(--border); background: none;"
                >
                  <span style="width: 8px; height: 8px; border-radius: 9999px; background: var(--coral); flex-shrink: 0; display: inline-block;"></span>
                  {unreadClientsCount} {unreadClientsCount === 1 ? "cliente" : "clientes"} com mensagens novas
                  <span style="margin-left: auto; color: var(--text-muted); font-size: 20px; line-height: 1;">›</span>
                </button>
              {/if}
              {#if inactiveCount > 0}
                <button
                  onclick={() => { clientFilter = "inativos"; }}
                  style="display: flex; align-items: center; gap: 10px; width: 100%; min-height: 48px; padding: 0 20px; text-align: left; font-size: 14px; color: var(--text-secondary); border-bottom: 1px solid var(--border); background: none;"
                >
                  <span style="width: 8px; height: 8px; border-radius: 9999px; background: #EF4444; flex-shrink: 0; display: inline-block;"></span>
                  {inactiveCount} {inactiveCount === 1 ? "cliente" : "clientes"} inativos
                  <span style="margin-left: auto; color: var(--text-muted); font-size: 20px; line-height: 1;">›</span>
                </button>
              {/if}
              {#if pendingPaymentCount > 0}
                <button
                  onclick={() => { clientFilter = "cobranca"; }}
                  style="display: flex; align-items: center; gap: 10px; width: 100%; min-height: 48px; padding: 0 20px; text-align: left; font-size: 14px; color: var(--text-secondary); border-bottom: 1px solid var(--border); background: none;"
                >
                  <span style="width: 8px; height: 8px; border-radius: 9999px; background: #F59E0B; flex-shrink: 0; display: inline-block;"></span>
                  {pendingPaymentCount} {pendingPaymentCount === 1 ? "cliente" : "clientes"} com pagamento pendente
                  <span style="margin-left: auto; color: var(--text-muted); font-size: 20px; line-height: 1;">›</span>
                </button>
              {/if}
              {#if scheduledMessageCount > 0}
                <button
                  onclick={() => { showApprovalPanel = true; }}
                  style="display: flex; align-items: center; gap: 10px; width: 100%; min-height: 48px; padding: 0 20px; text-align: left; font-size: 14px; font-weight: 500; color: var(--coral); background: none;"
                >
                  <span style="font-size: 16px; flex-shrink: 0;">📅</span>
                  {scheduledMessageCount} {scheduledMessageCount === 1 ? "mensagem sazonal" : "mensagens sazonais"} para aprovar
                  <span style="margin-left: auto; color: var(--coral); font-size: 20px; line-height: 1;">›</span>
                </button>
              {/if}
              {#if globalNearestSeasonal}
                <button
                  onclick={() => { clientFilter = "sazonal"; }}
                  style="display: flex; align-items: center; gap: 10px; width: 100%; min-height: 48px; padding: 0 20px; text-align: left; font-size: 14px; color: var(--text-secondary); border-top: 1px solid var(--border); background: none;"
                >
                  <span style="width: 8px; height: 8px; border-radius: 9999px; background: var(--sage); flex-shrink: 0; display: inline-block;"></span>
                  {globalNearestSeasonal.label} em {globalNearestSeasonal.daysUntil}d ({globalNearestSeasonal.eligibleCount} clientes)
                  <span style="margin-left: auto; color: var(--text-muted); font-size: 20px; line-height: 1;">›</span>
                </button>
              {/if}
            </div>
          {/if}

          <!-- Color legend -->
          <div style="display: flex; gap: 16px; align-items: center; padding: 8px 20px; background: var(--bg); border-bottom: 1px solid var(--border);">
            <span style="font-size: 13px; color: var(--text-muted);">Estado:</span>
            <span style="display: flex; align-items: center; gap: 5px; font-size: 13px; color: var(--text-muted);">
              <span style="width: 8px; height: 8px; border-radius: 9999px; background: #10B981; display: inline-block;"></span>Ativo
            </span>
            <span style="display: flex; align-items: center; gap: 5px; font-size: 13px; color: var(--text-muted);">
              <span style="width: 8px; height: 8px; border-radius: 9999px; background: #F59E0B; display: inline-block;"></span>5–9d
            </span>
            <span style="display: flex; align-items: center; gap: 5px; font-size: 13px; color: var(--text-muted);">
              <span style="width: 8px; height: 8px; border-radius: 9999px; background: #EF4444; display: inline-block;"></span>+10d
            </span>
          </div>

          <!-- Client list -->
          <div class="flex-1 overflow-y-auto">
            {#if clientFilter !== 'todos'}
              <button
                onclick={() => { clientFilter = 'todos'; }}
                style="display: flex; align-items: center; gap: 6px; width: 100%; min-height: 44px; padding: 0 20px; font-size: 14px; color: var(--coral); background: var(--coral-pale); border-bottom: 1px solid var(--coral-light); border: none; cursor: pointer;"
              >← Todos os clientes</button>
            {/if}
            {#if clients.length === 0}
              <p class="text-base p-5" style="color: var(--text-muted)">
                Você ainda não tem clientes. Toca no + pra começar!
              </p>
            {:else}
              {#each filteredClients as client (client.id)}
                {@const unread = unreadCounts[client.id] || 0}
                {@const health = clientHealth[client.id]}
                <button
                  onclick={() => selectClient(client.id)}
                  class="w-full text-left px-5 py-4 border-b transition-colors {selectedId === client.id ? 'bg-(--coral-pale)' : 'hover:bg-(--coral-pale)/40'}"
                  style="border-color: var(--border); color: var(--text)"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2.5 min-w-0">
                      {#if health}
                        <span
                          class="w-3 h-3 rounded-full shrink-0"
                          style="background: {health.color}"
                        ></span>
                      {/if}
                      <span class="font-semibold text-base truncate">{client.name}</span>
                      {#if client.invite_status && client.invite_status !== 'draft'}
                        <span
                          class="text-sm px-2 py-0.5 rounded-full shrink-0"
                          style={
                            client.invite_status === 'invited' ? 'background: #FEF3C7; color: #92400E' :
                            client.invite_status === 'accepted' ? 'background: #DBEAFE; color: #1E40AF' :
                            client.invite_status === 'active' ? 'background: #DEF7EC; color: #03543F' :
                            client.invite_status === 'payment_failed' ? 'background: #FEE2E2; color: #991B1B' :
                            client.invite_status === 'cancelled' ? 'background: #FEE2E2; color: #991B1B; text-decoration: line-through' :
                            'background: var(--border); color: var(--text-muted)'
                          }
                        >
                          {client.invite_status === 'invited' ? 'convite' :
                           client.invite_status === 'accepted' ? 'aceito' :
                           client.invite_status === 'active' ? 'ativo' :
                           client.invite_status === 'payment_failed' ? 'falhou' :
                           client.invite_status === 'cancelled' ? 'cancelado' :
                           client.invite_status}
                        </span>
                        {#if client.invite_status === 'accepted' && client.invite_sent_at && (Date.now() - new Date(client.invite_sent_at).getTime()) > 48 * 3600000}
                          <span class="text-sm shrink-0" title="Aceito há mais de 48h sem pagamento">&#9888;</span>
                        {/if}
                      {/if}
                    </div>
                    {#if unread > 0}
                      <span
                        class="text-sm font-bold px-2.5 py-1 rounded-full shrink-0 ml-2"
                        style="background: var(--coral); color: #fff"
                      >
                        {unread}
                      </span>
                    {/if}
                    {#if (suggestionCounts[client.id] ?? 0) > 0}
                      <span
                        class="shrink-0 ml-1"
                        title="Sugestões de perfil pendentes"
                        style="width: 8px; height: 8px; border-radius: 9999px; background: var(--sage); display: inline-block;"
                      ></span>
                    {/if}
                  </div>
                  <div class="flex items-center justify-between mt-1 pl-5 gap-2">
                    <span class="text-sm truncate" style="color: var(--text-muted); min-width: 0;">{client.type} · {client.city}</span>
                    {#if health && health.daysSinceMsg === 0}
                      <span class="text-sm shrink-0" style="color: var(--text-muted)">Hoje</span>
                    {:else if health && health.daysSinceMsg < 999}
                      <span class="text-sm font-medium shrink-0" style="color: {health.daysSinceMsg >= 5 ? health.color : 'var(--text-muted)'}">
                        {health.daysSinceMsg}d sem postar
                      </span>
                    {/if}
                  </div>
                  {#if client.charge_pending}
                    <div class="mt-1 pl-5">
                      <span style="font-size: 13px; padding: 2px 10px; border-radius: 9999px; background: #FEF3C7; color: #92400E;">⚠ pagamento pendente</span>
                    </div>
                  {/if}
                </button>
              {/each}
            {/if}
          </div>
        {/if}
      </div>

      <!-- Right: Thread or form -->
      <div class="flex-1 flex flex-col overflow-hidden {mobileView === 'detail' || mobileView === 'info' ? '' : 'hidden md:flex'}">
        {#if showForm}
          <div class="flex-1 overflow-y-auto p-5 md:p-6">
            <div
              class="max-w-xl rounded-2xl p-5 md:p-6"
              style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
            >
              <h2 class="text-lg font-semibold mb-4" style="color: var(--text)">
                {editingId ? "Editar cliente" : "Novo cliente"}
              </h2>

              {#if formError}
                <p class="text-base mb-4 p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">
                  {formError}
                </p>
              {/if}

              {#if voiceMode === 'done'}
                <!-- Done: summary chip + success banner + pre-filled content fields -->
                <div class="flex items-center gap-3 mb-4 p-3 rounded-xl" style="background: var(--bg); border: 1px solid var(--border)">
                  <div class="text-sm" style="flex: 1; min-width: 0;">
                    <span class="font-semibold" style="color: var(--text)">{formName || 'Novo cliente'}</span>
                    {#if formType || formCity}
                      <span style="color: var(--text-muted)"> · {[formType, formCity].filter(Boolean).join(', ')}</span>
                    {/if}
                  </div>
                  <button onclick={() => voiceMode = 'idle'} class="text-xs px-2 py-1 rounded-lg" style="color: var(--text-muted); border: 1px solid var(--border-strong)">Editar</button>
                </div>

                <div class="flex items-center gap-3 mb-5 p-3 rounded-xl" style="background: var(--sage-pale); border: 1.5px solid var(--sage-light)">
                  <div class="flex items-center justify-center flex-shrink-0" style="width: 32px; height: 32px; border-radius: 50%; background: var(--sage);">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                  </div>
                  <div>
                    <p class="text-sm font-bold" style="color: var(--text); margin: 0 0 2px;">Perfil extraído da gravação</p>
                    <p class="text-sm" style="color: var(--text-secondary); margin: 0;">Revise os campos antes de salvar.</p>
                  </div>
                </div>

                <div class="flex flex-col gap-4">
                  <div>
                    <span class="text-base font-medium" style="color: var(--text)">Serviços</span>
                    <div class="flex flex-col gap-2 mt-1.5">
                      {#each formServices as service, i}
                        <div class="flex gap-2 items-center">
                          <input bind:value={service.name} placeholder="Nome do serviço" class="flex-1 px-3 py-3 rounded-xl text-base outline-none border" style="min-height: 52px; border-color: {aiFilledFields.has('services') ? 'var(--sage)' : 'var(--border-strong)'}; background: {aiFilledFields.has('services') ? 'var(--sage-pale)' : 'var(--surface)'}; color: var(--text)" />
                          <div class="relative w-28">
                            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-base" style="color: var(--text-muted)">R$</span>
                            <input type="number" bind:value={service.price_brl} min="0" class="w-full pl-9 pr-3 py-3 rounded-xl text-base outline-none border" style="min-height: 52px; border-color: {aiFilledFields.has('services') ? 'var(--sage)' : 'var(--border-strong)'}; background: {aiFilledFields.has('services') ? 'var(--sage-pale)' : 'var(--surface)'}; color: var(--text)" />
                          </div>
                          <button onclick={() => removeService(i)} style="width: 40px; height: 52px; border: none; background: none; color: var(--text-muted); font-size: 22px; cursor: pointer; display: flex; align-items: center; justify-content: center; flex-shrink: 0;">×</button>
                        </div>
                      {/each}
                      <button onclick={addService} class="text-base font-medium mt-1 py-1" style="color: var(--primary)">+ Adicionar serviço</button>
                    </div>
                  </div>
                  <label class="flex flex-col gap-1.5">
                    <span class="text-base font-medium" style="color: var(--text)">Quem são os clientes?</span>
                    <span class="text-sm" style="color: var(--text-muted)">ex: mulheres de 25 a 50 anos que moram no bairro</span>
                    <textarea bind:value={formTargetAudience} placeholder="Descreve quem costuma ir lá..." rows={2} class="px-3 py-3 rounded-xl text-base outline-none border resize-none" style="min-height: 52px; border-color: {aiFilledFields.has('target_audience') ? 'var(--sage)' : 'var(--border-strong)'}; background: {aiFilledFields.has('target_audience') ? 'var(--sage-pale)' : 'var(--surface)'}; color: var(--text)"></textarea>
                  </label>
                  <label class="flex flex-col gap-1.5">
                    <span class="text-base font-medium" style="color: var(--text)">Como é o ambiente?</span>
                    <span class="text-sm" style="color: var(--text-muted)">ex: acolhedor, descontraído, serve cafezinho</span>
                    <textarea bind:value={formBrandVibe} placeholder="Conta um pouco sobre o clima do lugar..." rows={2} class="px-3 py-3 rounded-xl text-base outline-none border resize-none" style="min-height: 52px; border-color: {aiFilledFields.has('brand_vibe') ? 'var(--sage)' : 'var(--border-strong)'}; background: {aiFilledFields.has('brand_vibe') ? 'var(--sage-pale)' : 'var(--surface)'}; color: var(--text)"></textarea>
                  </label>
                  <label class="flex flex-col gap-1.5">
                    <span class="text-base font-medium" style="color: var(--text)">O que faz diferente?</span>
                    <span class="text-sm" style="color: var(--text-muted)">ex: agenda lotada às quintas</span>
                    <textarea bind:value={formQuirks} placeholder="Detalhes que fazem a cliente escolher esse lugar..." rows={3} class="px-3 py-3 rounded-xl text-base outline-none border resize-none" style="border-color: {aiFilledFields.has('quirks') ? 'var(--sage)' : 'var(--border-strong)'}; background: {aiFilledFields.has('quirks') ? 'var(--sage-pale)' : 'var(--surface)'}; color: var(--text)"></textarea>
                  </label>
                </div>

                <div class="text-center mt-4">
                  <button onclick={resetVoice} class="text-sm inline-flex items-center gap-1.5 p-3" style="color: var(--text-muted); border: none; background: none; cursor: pointer;">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 2a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V5a3 3 0 0 1 3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1M12 18v4M8 22h8"/></svg>
                    Gravar de novo
                  </button>
                </div>

                <div class="flex flex-col md:flex-row gap-3 mt-4">
                  <button onclick={closeForm} class="px-5 py-3 rounded-full text-base font-medium border" style="border-color: var(--border-strong); color: var(--text-secondary)">Cancelar</button>
                  <button onclick={saveClient} disabled={formSaving} class="px-5 py-3 rounded-full text-base font-medium" style="background: var(--coral); color: #fff; opacity: {formSaving ? '0.6' : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}">
                    {formSaving ? "Salvando..." : "Salvar e continuar"}
                  </button>
                  <button onclick={saveAndInvite} disabled={formSaving} class="px-5 py-3 rounded-full text-base font-medium" style="background: #25D366; color: #fff; opacity: {formSaving ? '0.6' : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}">
                    {formSaving ? "Salvando..." : "Salvar e Enviar Convite"}
                  </button>
                </div>

              {:else}
                <!-- idle: mic-first / recording / analyzing / manual+done: fields -->

                {#if voiceError}
                  <p class="text-sm mb-4 p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">{voiceError}</p>
                {/if}

                <div class="flex flex-col gap-4">
                  {#if voiceMode === 'idle'}
                    <!-- Idle: mic card first, fields hidden -->
                    <div class="flex items-center gap-3 md:gap-4 p-3 md:p-5 rounded-2xl" style="background: var(--coral-pale); border: 1.5px solid var(--coral-light)">
                      <button onclick={startVoiceRecording} aria-label="Gravar descrição" class="mic-btn" style="border-radius: 50%; background: var(--coral); border: none; display: flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0; box-shadow: 0 4px 16px rgba(249,115,104,0.35);">
                        <svg width="30" height="30" viewBox="0 0 24 24" fill="white"><path d="M12 2a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V5a3 3 0 0 1 3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1M12 18v4M8 22h8" stroke="white" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg>
                      </button>
                      <div class="min-w-0">
                        <p class="text-base font-bold" style="margin: 0 0 4px; color: var(--text)">Gravar descrição</p>
                        <p class="text-sm" style="margin: 0; color: var(--text-secondary); line-height: 1.4;">Toca no microfone e fala sobre a cliente</p>
                      </div>
                    </div>
                    <div class="text-center">
                      <button onclick={() => voiceMode = 'manual'} class="text-sm p-3" style="color: var(--text-muted); border: none; background: none; cursor: pointer; text-decoration: underline; text-underline-offset: 3px;">Preencher manualmente</button>
                    </div>

                  {:else if voiceMode === 'recording'}
                    <!-- Recording bar: [X] [●timer] [↑] -->
                    <div class="rounded-2xl overflow-hidden" style="border: 1.5px solid var(--border-strong);">
                      <div class="flex items-center rec-bar">
                        <button onclick={cancelRecording} aria-label="Cancelar gravação" class="rec-side-btn" style="background: #FEF2F2; border: none; border-right: 1px solid var(--border-strong); display: flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0;">
                          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#EF4444" stroke-width="2.5" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                        </button>
                        <div class="flex-1 flex items-center justify-center gap-3">
                          <div style="width: 10px; height: 10px; border-radius: 50%; background: #EF4444; flex-shrink: 0; animation: blink 1s ease-in-out infinite;"></div>
                          <span class="rec-timer" style="font-weight: 700; letter-spacing: 0.04em; color: var(--text); font-variant-numeric: tabular-nums;">{fmtTime(recordingSeconds)}</span>
                          <span class="text-sm rec-label" style="color: var(--text-muted)">Gravando</span>
                        </div>
                        <button onclick={submitRecording} aria-label="Enviar gravação" class="rec-side-btn" style="background: var(--coral); border: none; border-left: 1px solid var(--coral-light); display: flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0;">
                          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
                        </button>
                      </div>
                    </div>

                  {:else if voiceMode === 'analyzing'}
                    <!-- Analyzing bar -->
                    <div class="rounded-2xl flex items-center justify-center gap-3 rec-bar" style="border: 1.5px solid var(--border-strong);">
                      <div style="width: 22px; height: 22px; border-radius: 50%; border: 2.5px solid var(--coral); border-top-color: transparent; animation: spin 0.8s linear infinite; flex-shrink: 0;"></div>
                      <span class="text-base font-medium" style="color: var(--text-secondary)">Lendo o que você falou...</span>
                    </div>

                  {:else}
                    <!-- manual/done: gravar prompt + basic fields + content fields -->
                    {#if voiceMode === 'manual'}
                      <button onclick={resetVoice} class="w-full flex items-center gap-3 mb-1" style="min-height: 64px; background: var(--coral-pale); border: none; border-bottom: 1.5px solid var(--coral-light); border-radius: 12px; padding: 14px 16px; cursor: pointer; text-align: left;">
                        <div class="flex items-center justify-center flex-shrink-0" style="width: 36px; height: 36px; border-radius: 50%; background: var(--coral);">
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="white"><path d="M12 2a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V5a3 3 0 0 1 3-3z"/><path d="M19 10v1a7 7 0 0 1-14 0v-1M12 18v4M8 22h8" stroke="white" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg>
                        </div>
                        <p class="text-sm flex-1" style="color: var(--text-secondary); margin: 0; line-height: 1.4;">Prefere gravar uma descrição? É mais rápido.</p>
                        <span class="text-sm font-bold" style="color: var(--coral); white-space: nowrap;">Gravar →</span>
                      </button>
                    {/if}
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Nome do cliente</span>
                      <input bind:value={formClientName} placeholder="Ex: Ana Silva" class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                    </label>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Email do cliente</span>
                      <input bind:value={formClientEmail} type="email" placeholder="ana@email.com" class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                    </label>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Nome do negócio</span>
                      <input bind:value={formName} placeholder="Nome do negócio" class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                    </label>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Tipo de negócio</span>
                      <select bind:value={formType} class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)">
                        <option value="">Selecione...</option>
                        {#each BUSINESS_TYPES as t}<option value={t}>{t}</option>{/each}
                      </select>
                    </label>
                    <div class="flex gap-3">
                      <label class="flex flex-col gap-1.5 flex-1">
                        <span class="text-base font-medium" style="color: var(--text)">Cidade</span>
                        <input bind:value={formCity} placeholder="Ex: São Paulo" class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                      </label>
                      <label class="flex flex-col gap-1.5 w-28">
                        <span class="text-base font-medium" style="color: var(--text)">Estado</span>
                        <select bind:value={formState} class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)">
                          <option value="">UF</option>
                          {#each STATES as stateCode}<option value={stateCode}>{stateCode}</option>{/each}
                        </select>
                      </label>
                    </div>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Telefone WhatsApp</span>
                      <input bind:value={formPhone} placeholder="5511999998888" class="px-3 py-3 rounded-xl text-base outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                    </label>
                    <div style="height: 1px; background: var(--border); margin: 4px 0;"></div>
                    <div>
                      <span class="text-base font-medium" style="color: var(--text)">Serviços</span>
                      <p class="text-sm mt-0.5 mb-1.5" style="color: var(--text-muted)">Coloca os serviços mais pedidos e o preço de cada um.</p>
                      <div class="flex flex-col gap-2">
                        {#each formServices as service, i}
                          <div class="flex gap-2 items-center">
                            <input bind:value={service.name} placeholder="Nome do serviço" class="flex-1 px-3 py-3 rounded-xl text-base outline-none border" style="min-height: 52px; border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                            <div class="relative w-28">
                              <span class="absolute left-3 top-1/2 -translate-y-1/2 text-base" style="color: var(--text-muted)">R$</span>
                              <input type="number" bind:value={service.price_brl} min="0" class="w-full pl-9 pr-3 py-3 rounded-xl text-base outline-none border" style="min-height: 52px; border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                            </div>
                            {#if formServices.length > 1}
                              <button onclick={() => removeService(i)} style="width: 40px; height: 52px; border: none; background: none; color: var(--text-muted); font-size: 22px; cursor: pointer; display: flex; align-items: center; justify-content: center; flex-shrink: 0;">×</button>
                            {/if}
                          </div>
                        {/each}
                        <button onclick={addService} class="text-base font-medium mt-1 py-1" style="color: var(--primary)">+ Adicionar serviço</button>
                      </div>
                    </div>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Quem são os clientes?</span>
                      <span class="text-sm" style="color: var(--text-muted)">ex: mulheres de 25 a 50 anos que moram no bairro</span>
                      <textarea bind:value={formTargetAudience} placeholder="Descreve quem costuma ir lá..." rows={2} class="px-3 py-3 rounded-xl text-base outline-none border resize-none" style="min-height: 52px; border-color: var(--border-strong); background: var(--surface); color: var(--text)"></textarea>
                    </label>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">Como é o ambiente?</span>
                      <span class="text-sm" style="color: var(--text-muted)">ex: acolhedor, descontraído, serve cafezinho</span>
                      <textarea bind:value={formBrandVibe} placeholder="Conta um pouco sobre o clima do lugar..." rows={2} class="px-3 py-3 rounded-xl text-base outline-none border resize-none" style="min-height: 52px; border-color: var(--border-strong); background: var(--surface); color: var(--text)"></textarea>
                    </label>
                    <label class="flex flex-col gap-1.5">
                      <span class="text-base font-medium" style="color: var(--text)">O que faz diferente?</span>
                      <span class="text-sm" style="color: var(--text-muted)">ex: agenda lotada às quintas</span>
                      <textarea bind:value={formQuirks} placeholder="Ex: Atendimento por WhatsApp, parcela em 3x" rows={2} class="px-3 py-3 rounded-xl text-base outline-none border resize-none" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"></textarea>
                    </label>
                  {/if}
                </div>

                {#if inviteUrl}
                  <div class="mt-4 rounded-xl p-4" style="background: var(--sage-pale); border: 1px solid var(--border)">
                    <p class="text-base font-medium mb-2" style="color: var(--text)">Convite enviado!</p>
                    <div class="flex items-center gap-2">
                      <input readonly value={inviteUrl} class="flex-1 px-3 py-3 rounded-lg text-sm outline-none border" style="border-color: var(--border-strong); background: var(--surface); color: var(--text)" />
                      <button onclick={copyInviteUrl} class="px-4 py-3 rounded-lg text-sm font-medium" style="background: var(--coral); color: #fff">{inviteCopied ? "Copiado!" : "Copiar"}</button>
                    </div>
                  </div>
                {/if}

                {#if voiceMode !== 'recording' && voiceMode !== 'analyzing'}
                  <div class="flex flex-col md:flex-row gap-3 mt-6">
                    <button onclick={closeForm} class="px-5 py-3 rounded-full text-base font-medium border" style="border-color: var(--border-strong); color: var(--text-secondary)">Cancelar</button>
                    {#if voiceMode === 'idle'}
                      <button disabled class="px-5 py-3 rounded-full text-base font-medium" style="background: rgba(17,17,22,0.08); color: var(--text-muted); cursor: not-allowed;">Salvar cliente</button>
                    {:else}
                      <button onclick={saveClient} disabled={formSaving} class="px-5 py-3 rounded-full text-base font-medium" style="background: var(--coral); color: #fff; opacity: {formSaving ? '0.6' : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}">{formSaving ? "Salvando..." : "Salvar"}</button>
                      <button onclick={saveAndInvite} disabled={formSaving} class="px-5 py-3 rounded-full text-base font-medium" style="background: #25D366; color: #fff; opacity: {formSaving ? '0.6' : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}">{formSaving ? "Salvando..." : "Salvar e Enviar Convite"}</button>
                    {/if}
                  </div>
                {/if}
              {/if}
            </div>
          </div>
        {:else if selected}
          <!-- Info screen (mobile only) -->
          {#if mobileView === 'info'}
            <div class="flex-1 flex flex-col overflow-hidden">

              <!-- Header: Voltar | Name/Type | Editar -->
              <div style="background: var(--surface); border-bottom: 1px solid var(--border); display: flex; align-items: center; min-height: 60px; padding: 0 16px; gap: 4px; flex-shrink: 0;">
                <button
                  onclick={() => { history.back(); }}
                  style="display: flex; align-items: center; gap: 4px; min-height: 60px; padding: 0 12px 0 0; font-size: 14px; font-weight: 500; color: var(--coral); flex-shrink: 0;"
                >
                  <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20"><path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                  Voltar
                </button>
                <!-- Avatar -->
                <div style="width: 44px; height: 44px; border-radius: 9999px; overflow: hidden; flex-shrink: 0; display: flex; align-items: center; justify-content: center; font-size: 16px; font-weight: 600; background: var(--coral-pale); color: var(--coral);">
                  {#if profilePictureUrl(selected)}
                    <img src={profilePictureUrl(selected)!} alt={selected.name} style="width: 100%; height: 100%; object-fit: cover;" />
                  {:else}
                    {initials(selected)}
                  {/if}
                </div>
                <div style="flex: 1; min-width: 0;">
                  <div style="font-size: 17px; font-weight: 600; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{selected.name}</div>
                  <div style="font-size: 14px; color: var(--text-secondary);">{selected.type} — {selected.city}/{selected.state}</div>
                </div>
                <button
                  onclick={() => openEditForm(selected!)}
                  style="min-height: 48px; padding: 0 18px; border-radius: 9999px; font-size: 14px; font-weight: 500; color: var(--text-secondary); border: 1px solid var(--border-strong); flex-shrink: 0;"
                >Editar</button>
              </div>

              <div class="flex-1 overflow-y-auto">

                <!-- Status strip -->
                <div style="background: var(--surface); padding: 14px 20px; border-bottom: 1px solid var(--border); display: flex; align-items: center; gap: 10px; flex-wrap: wrap;">
                  {#if selected.tier}
                    <span style="font-size: 14px; padding: 4px 12px; border-radius: 9999px; background: var(--bg); color: var(--text-secondary); border: 1px solid var(--border-strong); font-weight: 500;">{selected.tier}</span>
                  {/if}
                  {#if selected.invite_status && selected.invite_status !== 'draft'}
                    <span
                      style="font-size: 14px; padding: 4px 12px; border-radius: 9999px; font-weight: 500; {
                        selected.invite_status === 'invited' ? 'background: #FEF3C7; color: #92400E' :
                        selected.invite_status === 'accepted' ? 'background: #DBEAFE; color: #1E40AF' :
                        selected.invite_status === 'active' ? 'background: #DEF7EC; color: #03543F' :
                        selected.invite_status === 'payment_failed' ? 'background: #FEE2E2; color: #991B1B' :
                        selected.invite_status === 'cancelled' ? 'background: #FEE2E2; color: #991B1B; text-decoration: line-through' :
                        'background: var(--border); color: var(--text-muted)'
                      }"
                    >
                      {selected.invite_status === 'invited' ? 'convite' :
                       selected.invite_status === 'accepted' ? 'aceito' :
                       selected.invite_status === 'active' ? 'ativo' :
                       selected.invite_status === 'payment_failed' ? 'falhou' :
                       selected.invite_status === 'cancelled' ? 'cancelado' :
                       selected.invite_status}
                    </span>
                  {/if}
                  {#if selected.invite_status === 'active' && selected.next_charge_date}
                    <span style="font-size: 14px; color: var(--text-muted); margin-left: 4px;">
                      Próx. cobrança: {new Date(selected.next_charge_date).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' })}
                    </span>
                  {/if}
                  {#if selected.charge_pending}
                    <span style="font-size: 14px; padding: 4px 12px; border-radius: 9999px; font-weight: 500; background: #FEE2E2; color: #991B1B;">Pagamento pendente</span>
                  {/if}
                  {#if selected.type === 'Desconhecido'}
                    <button
                      onclick={() => openEditForm(selected!)}
                      style="min-height: 48px; padding: 0 20px; border-radius: 9999px; font-size: 15px; font-weight: 600; background: #25D366; color: #fff;"
                    >Criar conta</button>
                  {/if}
                </div>

                <!-- Section: Serviços -->
                {#if selected.services?.length > 0}
                  <span style="font-size: 13px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-muted); padding: 14px 20px 8px; background: var(--bg); border-top: 1px solid var(--border); display: block;">Serviços</span>
                  <div style="background: var(--surface); padding: 4px 0 12px;">
                    {#each selected.services as svc}
                      <div style="display: flex; align-items: baseline; gap: 12px; padding: 6px 20px;">
                        <span style="font-size: 13px; font-weight: 600; color: var(--text-muted); white-space: nowrap; flex-shrink: 0; width: 100px; overflow: hidden; text-overflow: ellipsis;">{svc.name}</span>
                        <span style="font-size: 15px; color: var(--text-secondary); flex: 1;">{svc.price_brl != null ? `R$${svc.price_brl.toFixed(2).replace('.', ',')}` : '—'}</span>
                      </div>
                    {/each}
                  </div>
                {/if}

                <!-- Section: Perfil -->
                {#if selected.target_audience || selected.brand_vibe || selected.quirks}
                  <span style="font-size: 13px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-muted); padding: 14px 20px 8px; background: var(--bg); border-top: 1px solid var(--border); display: block;">Perfil</span>
                  <div style="background: var(--surface); padding: 4px 0 12px;">
                    {#if selected.target_audience}
                      <div style="display: flex; align-items: baseline; gap: 12px; padding: 6px 20px;">
                        <span style="font-size: 13px; font-weight: 600; color: var(--text-muted); white-space: nowrap; flex-shrink: 0; width: 64px;">Público</span>
                        <span style="font-size: 15px; color: var(--text-secondary); flex: 1;">{selected.target_audience}</span>
                      </div>
                    {/if}
                    {#if selected.brand_vibe}
                      <div style="display: flex; align-items: baseline; gap: 12px; padding: 6px 20px;">
                        <span style="font-size: 13px; font-weight: 600; color: var(--text-muted); white-space: nowrap; flex-shrink: 0; width: 64px;">Estilo</span>
                        <span style="font-size: 15px; color: var(--text-secondary); flex: 1; font-style: italic;">{selected.brand_vibe}</span>
                      </div>
                    {/if}
                    {#if selected.quirks}
                      <div style="display: flex; align-items: baseline; gap: 12px; padding: 6px 20px;">
                        <span style="font-size: 13px; font-weight: 600; color: var(--text-muted); white-space: nowrap; flex-shrink: 0; width: 64px;">Obs.</span>
                        <span style="font-size: 15px; color: var(--text-secondary); flex: 1; white-space: pre-wrap;">{selected.quirks}</span>
                      </div>
                    {/if}
                  </div>
                {/if}

                <!-- Section: Sugestões de Perfil -->
                {#if suggestions.length > 0}
                  <button
                    onclick={() => { suggestionsOpen = !suggestionsOpen; }}
                    style="width: 100%; display: flex; align-items: center; justify-content: space-between; padding: 14px 20px 8px; background: var(--bg); border-top: 1px solid var(--border); border-left: none; border-right: none; border-bottom: none; cursor: pointer;"
                  >
                    <div style="display: flex; align-items: center; gap: 8px;">
                      <span style="font-size: 13px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--sage-dark);">Sugestões de perfil</span>
                      <span style="font-size: 12px; font-weight: 700; padding: 1px 7px; border-radius: 9999px; background: var(--sage); color: #fff;">{suggestions.length}</span>
                    </div>
                    <span style="font-size: 13px; color: var(--text-muted);">{suggestionsOpen ? '▴' : '▾'}</span>
                  </button>
                  {#if suggestionsOpen}
                    <div style="background: var(--surface);">
                      {#each suggestions as sug (sug.id)}
                        <div style="padding: 12px 20px; border-bottom: 1px solid var(--border);">
                          <span style="font-size: 13px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-muted);">
                            {sug.field === 'services' ? 'Serviço detectado' : sug.field === 'quirks' ? 'Diferencial detectado' : sug.field === 'target_audience' ? 'Público detectado' : 'Estilo detectado'}
                          </span>
                          <p style="font-size: 15px; color: var(--text-secondary); margin: 4px 0 10px; line-height: 1.5;">{sug.suggestion}</p>
                          <div style="display: flex; gap: 8px;">
                            <button
                              onclick={() => acceptSuggestion(sug)}
                              style="min-height: 40px; padding: 0 16px; border-radius: 9999px; font-size: 14px; font-weight: 600; background: var(--sage-pale); color: var(--sage-dark); border: 1px solid var(--sage-light);"
                            >Adicionar</button>
                            <button
                              onclick={() => dismissSuggestion(sug)}
                              style="min-height: 40px; padding: 0 16px; border-radius: 9999px; font-size: 14px; font-weight: 500; color: var(--text-muted); border: 1px solid var(--border-strong); background: none;"
                            >Ignorar</button>
                          </div>
                        </div>
                      {/each}
                    </div>
                  {/if}
                {/if}

                <!-- Section: Posts recentes -->
                {#if clientPosts.length > 0}
                  <span style="font-size: 13px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-muted); padding: 14px 20px 8px; background: var(--bg); border-top: 1px solid var(--border); display: block;">Posts recentes</span>
                  <div style="background: var(--surface);">
                    {#each clientPosts.slice(0, historyLimit) as post (post.id)}
                      {@const postExpanded = expandedPosts.has(post.id)}
                      <div style="padding: 12px 20px; border-bottom: 1px solid var(--border); display: flex; gap: 12px; align-items: flex-start;">
                        <span style="font-size: 14px; color: var(--text-muted); white-space: nowrap; padding-top: 2px; width: 48px; flex-shrink: 0;">
                          {new Date(post.created).toLocaleDateString("pt-BR", { day: "numeric", month: "short" })}
                        </span>
                        <div style="flex: 1; min-width: 0;">
                          <button
                            onclick={() => {
                              const next = new Set(expandedPosts);
                              if (postExpanded) next.delete(post.id); else next.add(post.id);
                              expandedPosts = next;
                            }}
                            style="font-size: 15px; color: var(--text-secondary); line-height: 1.5; text-align: left; width: 100%; background: none; border: none; padding: 0; {postExpanded ? '' : 'display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;'}"
                          >
                            {post.caption}
                          </button>
                          {#if !postExpanded}
                            <span style="font-size: 13px; color: var(--text-muted); margin-top: 2px; display: block;">ver mais</span>
                          {/if}
                          {#if postExpanded && post.production_note}
                            <p style="font-size: 14px; font-style: italic; margin-top: 4px; color: var(--text-muted);">{post.production_note}</p>
                          {/if}
                        </div>
                        <button
                          onclick={async () => {
                            await copyText(post.caption + (post.hashtags?.length ? '\n\n' + post.hashtags.join(' ') : ''));
                          }}
                          style="min-height: 48px; padding: 0 14px; border-radius: 9999px; font-size: 14px; font-weight: 500; color: var(--coral); border: 1px solid var(--coral-light); background: var(--coral-pale); white-space: nowrap; flex-shrink: 0;"
                        >Copiar</button>
                      </div>
                    {/each}
                    {#if clientPosts.length > historyLimit}
                      <button
                        onclick={() => { historyLimit = clientPosts.length; }}
                        style="display: block; width: 100%; min-height: 48px; padding: 0 20px; text-align: left; font-size: 14px; font-weight: 500; color: var(--coral); background: none; border: none;"
                      >
                        Ver todos ({clientPosts.length})
                      </button>
                    {/if}
                  </div>
                {/if}

                <!-- Section: Lembrete -->
                {#if nudgeTier || nudgeText}
                  <span style="font-size: 13px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; padding: 14px 20px 8px; display: block; {nudgeTier ? 'background: #FFF7ED; border-top: 1px solid #FDE68A; color: #92400E;' : 'background: var(--bg); border-top: 1px solid var(--border); color: var(--text-muted);'}">
                    {nudgeTier ? `⚠ Lembrete — ${clientHealth[selected!.id]?.daysSinceMsg} dias sem mensagem` : 'Mensagem'}
                  </span>
                  <div style="padding: 12px 20px 16px; border-bottom: 1px solid var(--border); {nudgeTier ? 'background: #FFFBEB;' : 'background: var(--surface);'}">
                    <textarea
                      bind:value={nudgeText}
                      rows={2}
                      class="w-full rounded-xl text-base outline-none border resize-none"
                      style="padding: 12px 16px; border-color: var(--border-strong); background: var(--surface); color: var(--text);"
                    ></textarea>
                    <div class="flex items-center gap-2 mt-2">
                      <button
                        onclick={sendNudge}
                        disabled={sendingNudge || !nudgeText.trim() || !!blockReason}
                        title={blockReason ?? undefined}
                        style="min-height: 48px; padding: 0 24px; border-radius: 9999px; font-size: 15px; font-weight: 600; background: #25D366; color: #fff; opacity: {sendingNudge || !nudgeText.trim() || blockReason ? '0.6' : '1'}; cursor: {sendingNudge || !nudgeText.trim() || blockReason ? 'not-allowed' : 'pointer'};"
                      >
                        {sendingNudge ? "Enviando..." : "Enviar lembrete"}
                      </button>
                      {#if blockReason && nudgeText.trim()}
                        <span class="text-sm" style="color: var(--text-muted)">{blockReason}</span>
                      {/if}
                      {#if sendNudgeError}
                        <span class="text-sm" style="color: var(--destructive)">{sendNudgeError}</span>
                      {/if}
                    </div>
                  </div>
                {/if}

                <!-- Section: Datas próximas -->
                {#if upcomingDates.length > 0}
                  <span style="font-size: 13px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-muted); padding: 14px 20px 8px; background: var(--bg); border-top: 1px solid var(--border); display: block;">Datas próximas</span>
                  <div style="background: var(--surface); padding: 8px 20px 16px; display: flex; flex-wrap: wrap; gap: 8px;">
                    {#each upcomingDates as sd}
                      <button
                        onclick={() => { prefillSeasonalMessage(sd.template); }}
                        style="min-height: 48px; padding: 0 16px; border-radius: 9999px; font-size: 14px; font-weight: 500; border: 1px solid var(--border-strong); color: var(--text-secondary); background: none;"
                      >
                        {sd.label} · {sd.daysUntil}d
                      </button>
                    {/each}
                  </div>
                {/if}

                <!-- Danger zone: Cancelar assinatura -->
                {#if selected.invite_status === 'active'}
                  <div style="margin: 16px 20px 24px; padding: 16px; border-radius: 16px; border: 1px solid #FCA5A5; background: #FFF5F5;">
                    <p style="font-size: 14px; color: #991B1B; margin-bottom: 10px; line-height: 1.5;">
                      Cancelar a assinatura desativa o acesso de {selected.name} ao rekan.
                    </p>
                    <button
                      onclick={cancelSubscription}
                      disabled={cancelling}
                      style="min-height: 48px; padding: 0 20px; border-radius: 9999px; font-size: 14px; font-weight: 600; border: 1.5px solid #EF4444; color: #EF4444; background: transparent; opacity: {cancelling ? '0.6' : '1'};"
                    >
                      {cancelling ? "Cancelando..." : "Cancelar assinatura"}
                    </button>
                  </div>
                {/if}

              </div>
            </div>
          {/if}

          <!-- Detail view: hidden on mobile when info screen is open -->
          <div class="{mobileView === 'info' ? 'hidden' : 'flex'} flex-col flex-1 overflow-hidden relative">
          <!-- Client header -->
          <div
            class="px-5 py-4 border-b shrink-0"
            style="border-color: var(--border); background: var(--surface)"
          >
            <div class="flex items-center gap-2">
              <button
                onclick={() => { history.back(); }}
                class="md:hidden flex items-center gap-1 py-2 pr-2 -ml-1 rounded-lg shrink-0 text-sm font-medium"
                style="color: var(--coral)"
              >
                <svg viewBox="0 0 20 20" fill="currentColor" width="18" height="18"><path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                Voltar
              </button>
              <!-- Tappable name opens info screen -->
              <button
                onclick={() => { mobileView = 'info'; history.pushState({ mobileView: 'info' }, ''); }}
                class="min-w-0 flex-1 flex items-center gap-3 text-left"
              >
                <!-- Avatar -->
                <div class="shrink-0 w-9 h-9 rounded-full overflow-hidden flex items-center justify-center text-sm font-semibold" style="background: var(--coral-pale); color: var(--coral);">
                  {#if profilePictureUrl(selected)}
                    <img src={profilePictureUrl(selected)!} alt={selected.name} class="w-full h-full object-cover" />
                  {:else}
                    {initials(selected)}
                  {/if}
                </div>
                <div class="min-w-0">
                  <h2 class="text-base font-semibold truncate" style="color: var(--text)">
                    {selected.name}
                  </h2>
                  <p class="text-sm" style="color: var(--text-secondary)">
                    {selected.type} — {selected.city}/{selected.state}
                    <span class="ml-1 text-xs" style="color: var(--text-muted)">›</span>
                  </p>
                </div>
              </button>
            </div>
          </div>

          <!-- Mobile ideas picker — full screen overlay (shows immediately on tap, then fills in) -->
          {#if generatingIdeas || ideaDrafts !== null}
            <div class="md:hidden absolute inset-0 flex flex-col z-10" style="background: var(--bg)">
              <div class="flex items-center gap-3 px-4 shrink-0" style="min-height: 60px; background: var(--surface); border-bottom: 1px solid var(--border);">
                {#if !generatingIdeas}
                  <button
                    onclick={() => { ideaDrafts = null; selectedIdeas = new Set(); }}
                    style="display: flex; align-items: center; gap: 4px; min-height: 60px; padding: 0 12px 0 0; font-size: 14px; font-weight: 500; color: var(--coral); flex-shrink: 0;"
                  >
                    <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20"><path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>
                    Voltar
                  </button>
                {/if}
                <span class="text-base font-semibold" style="color: var(--text)">
                  {generatingIdeas ? 'Gerando ideias...' : 'Selecione ideias'}
                </span>
              </div>
              {#if generatingIdeas}
                <div class="flex-1 flex flex-col items-center justify-center gap-4 p-8">
                  <svg class="animate-spin" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--coral)"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
                  <p class="text-base text-center" style="color: var(--text-muted)">Escrevendo 3 ideias<br>caprichadas pra você...</p>
                </div>
              {:else}
                <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
                  {#each ideaDrafts! as draft, i}
                    <button
                      onclick={() => {
                        if (selectedIdeas.has(i)) {
                          const next = new Set(selectedIdeas);
                          next.delete(i);
                          selectedIdeas = next;
                        } else {
                          selectedIdeas = new Set(selectedIdeas).add(i);
                        }
                      }}
                      class="rounded-2xl p-5 text-left transition-colors"
                      style="background: var(--surface); border: 2px solid {selectedIdeas.has(i) ? 'var(--coral)' : 'var(--border)'};"
                    >
                      <div class="flex items-start gap-3">
                        <div class="shrink-0 w-6 h-6 rounded-full flex items-center justify-center mt-0.5" style="border: 2px solid {selectedIdeas.has(i) ? 'var(--coral)' : 'var(--border)'}; background: {selectedIdeas.has(i) ? 'var(--coral)' : 'transparent'};">
                          {#if selectedIdeas.has(i)}
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                          {/if}
                        </div>
                        <div class="min-w-0 flex-1">
                          <p class="text-base leading-relaxed" style="color: var(--text); white-space: pre-wrap">{draft.caption}</p>
                          {#if draft.hashtags?.length}
                            <p class="text-sm mt-3" style="color: var(--text-muted)">{draft.hashtags.join(' ')}</p>
                          {/if}
                        </div>
                      </div>
                    </button>
                  {/each}
                </div>
                {#if selectedIdeas.size > 0}
                  <div class="shrink-0 p-4 flex gap-2" style="background: var(--surface); border-top: 1px solid var(--border);">
                    {#if selectedIdeas.size === 1}
                      <button
                        onclick={() => {
                          const idx = [...selectedIdeas][0];
                          result = ideaDrafts![idx];
                          isProactive = true;
                        }}
                        class="flex-1 rounded-full text-base font-semibold"
                        style="min-height: 52px; background: var(--coral); color: #fff;"
                      >Revisar e enviar</button>
                    {:else}
                      <button
                        onclick={sendSelectedIdeas}
                        disabled={sendingIdeas}
                        class="flex-1 rounded-full text-base font-semibold"
                        style="min-height: 52px; background: #25D366; color: #fff; opacity: {sendingIdeas ? '0.6' : '1'}"
                      >{sendingIdeas ? 'Enviando...' : `Enviar ${selectedIdeas.size} selecionadas`}</button>
                    {/if}
                    <button
                      onclick={() => { selectedIdeas = new Set(); }}
                      class="shrink-0 px-4 rounded-full text-base font-medium"
                      style="min-height: 52px; color: var(--destructive); background: var(--bg); border: 1px solid var(--border);"
                    >Cancelar</button>
                  </div>
                {/if}
              {/if}
            </div>
          {/if}

          <!-- Post review overlay -->
          {#if result && showReviewOverlay}
            <div class="absolute inset-0 flex flex-col z-10" style="background: var(--bg)">
              <div class="flex items-center gap-3 px-4 shrink-0" style="min-height: 60px; background: var(--surface); border-bottom: 1px solid var(--border);">
                <button
                  onclick={() => { if (ideaDrafts) { result = null; } else { showReviewOverlay = false; } }}
                  style="display: flex; align-items: center; gap: 4px; min-height: 60px; padding: 0 12px 0 0; font-size: 14px; font-weight: 500; color: var(--coral); flex-shrink: 0;"
                >
                  <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20"><path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>
                  Voltar
                </button>
                <span class="text-base font-semibold" style="color: var(--text)">Post gerado</span>
              </div>

              <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
                <div>
                  <div class="flex items-center justify-between mb-1">
                    <span class="text-sm font-medium" style="color: var(--text-muted)">Legenda</span>
                    <button onclick={() => copyWithFeedback("caption", editingCaption)} class="text-sm py-1" style="color: var(--coral)">
                      {copied.caption ? "Copiado!" : "Copiar"}
                    </button>
                  </div>
                  <textarea
                    bind:value={editingCaption}
                    class="w-full rounded-xl p-3 text-base leading-relaxed resize-none"
                    style="background: var(--surface); border: 1px solid var(--border); color: var(--text); field-sizing: content; min-height: 120px;"
                  ></textarea>
                </div>

                <div>
                  <div class="flex items-center justify-between mb-1">
                    <span class="text-sm font-medium" style="color: var(--text-muted)">Hashtags</span>
                    <button onclick={() => copyWithFeedback("hashtags", result!.hashtags.join(" "))} class="text-sm py-1" style="color: var(--coral)">
                      {copied.hashtags ? "Copiado!" : "Copiar"}
                    </button>
                  </div>
                  <p class="text-sm" style="color: var(--text-secondary)">
                    {result.hashtags.join(" ")}
                  </p>
                </div>

                {#if result.production_note}
                  <div>
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-sm font-medium" style="color: var(--text-muted)">Nota de produção</span>
                      <button onclick={() => copyWithFeedback("note", result!.production_note!)} class="text-sm py-1" style="color: var(--coral)">
                        {copied.note ? "Copiado!" : "Copiar"}
                      </button>
                    </div>
                    <p
                      class="text-sm italic mt-1"
                      style="color: var(--text-secondary); border-left: 2px solid var(--border-strong); padding-left: 0.75rem"
                    >
                      {result.production_note}
                    </p>
                  </div>
                {/if}
              </div>

              <div class="shrink-0 px-4 py-3 flex flex-col gap-2" style="background: var(--surface); border-top: 1px solid var(--border);">
                {#if blockReason}
                  <span class="text-sm" style="color: var(--text-muted)">{blockReason} — não é possível enviar agora.</span>
                {:else}
                  <button
                    onclick={sendViaWhatsApp}
                    disabled={sending}
                    class="w-full px-6 py-3 rounded-full text-base font-medium transition-opacity"
                    style="background: #25D366; color: #fff; opacity: {sending ? '0.6' : '1'}; cursor: {sending ? 'not-allowed' : 'pointer'}"
                  >
                    {sending ? "Enviando..." : "Enviar pelo WhatsApp"}
                  </button>
                  {#if sendError}
                    <span class="text-sm" style="color: var(--destructive)">{sendError}</span>
                  {/if}
                {/if}
                <button
                  onclick={() => { result = null; message = ""; isProactive = false; ideaDrafts = null; selectedIdeas = new Set(); }}
                  class="w-full py-2 text-sm font-medium"
                  style="color: var(--destructive)"
                >
                  Descartar
                </button>
              </div>
            </div>
          {/if}

          <!-- Message thread -->
          <div bind:this={threadEl} data-testid="message-thread" class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-2" style="background: var(--chat-bg)">
            {#if groupedMessages.length === 0}
              <p class="text-base text-center py-8" style="color: var(--text-muted)">
                Quando {selected.name} mandar mensagem, aparece aqui.
              </p>
            {:else}
              {#each groupedMessages as group}
                <div class="flex items-center gap-3 my-1">
                  <hr class="flex-1" style="border-color: var(--border)" />
                  <span class="text-sm shrink-0" style="color: var(--text-muted)">{group.label}</span>
                  <hr class="flex-1" style="border-color: var(--border)" />
                </div>
                {#each group.msgs as msg (msg.id)}
                  <div class="flex {msg.direction === 'outgoing' ? 'justify-end' : 'justify-start'}">
                    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
                    <div
                      class="rounded-2xl px-4 py-3 text-base"
                      style="max-width: 280px; background: {selectedMessages.has(msg.id) ? 'var(--coral-pale)' : msg.direction === 'outgoing' ? 'var(--coral-pale)' : 'var(--surface)'}; border: 1px solid {msg.direction === 'outgoing' ? 'var(--coral-light)' : 'var(--border)'}; color: var(--text); line-height: 1.5; {inputMode === 'generate' ? 'cursor: pointer;' : ''} {selectedMessages.has(msg.id) ? 'border-left: 3px solid var(--coral);' : ''}"
                      role={inputMode === 'generate' ? 'button' : undefined}
                      onclick={() => { if (inputMode === 'generate') toggleMessageSelection(msg.id); }}
                      onkeydown={(e) => { if (inputMode === 'generate' && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); toggleMessageSelection(msg.id); } }}
                    >
                      {#if msg.type === "audio"}
                        <span class="text-sm font-medium block mb-1" style="color: var(--text-muted)">Áudio transcrito</span>
                      {/if}

                      {#if msg.type === "image" && msg.media}
                        <img
                          src={mediaUrl(msg)}
                          alt="Imagem do cliente"
                          class="rounded-xl mb-2 max-w-full"
                          style="max-height: 240px"
                        />
                      {/if}

                      {#if msg.type === "video" && msg.media}
                        <!-- svelte-ignore a11y_media_has_caption -->
                        <!-- biome-ignore lint/a11y/useMediaCaption: client-uploaded video, no captions available -->
                        <video
                          src={mediaUrl(msg)}
                          controls
                          class="rounded-xl mb-2 max-w-full"
                          style="max-height: 240px"
                        >
                          Vídeo do cliente
                        </video>
                      {/if}

                      {#if msg.content}
                        <p class="whitespace-pre-wrap">{msg.content}</p>
                      {:else if msg.type === "audio"}
                        <p class="italic" style="color: var(--text-muted)">Transcrição indisponível</p>
                      {/if}

                      <span class="text-sm block mt-1" style="color: var(--text-muted)">
                        {new Date(msg.wa_timestamp || msg.created).toLocaleTimeString("pt-BR", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </span>
                    </div>
                  </div>
                {/each}
              {/each}
            {/if}
          </div>

          <!-- Unified input bar -->
          <div
            data-testid="input-bar"
            class="shrink-0 border-t px-3 md:px-4 py-3 flex flex-col gap-2"
            style="border-color: {inputMode === 'generate' ? 'var(--coral)' : 'var(--border)'}; background: {inputMode === 'generate' ? 'var(--coral-pale)' : 'var(--surface)'}; {inputMode === 'generate' ? 'border-top-width: 2px;' : ''}"
          >
            <!-- Idea drafts (desktop only) -->
            {#if ideaDrafts !== null}
              <div class="hidden md:flex flex-col gap-3 mb-2">
                <div class="flex items-center justify-between">
                  <span class="text-sm font-medium" style="color: var(--text-muted)">
                    Selecione ideias
                  </span>
                  <button
                    onclick={() => { ideaDrafts = null; selectedIdeas = new Set(); }}
                    class="text-sm py-1"
                    style="color: var(--text-muted)"
                  >Cancelar</button>
                </div>
                {#each ideaDrafts as draft, i}
                  <button
                    onclick={() => {
                      if (selectedIdeas.has(i)) {
                        const next = new Set(selectedIdeas);
                        next.delete(i);
                        selectedIdeas = next;
                      } else {
                        selectedIdeas = new Set(selectedIdeas).add(i);
                      }
                    }}
                    class="rounded-xl p-4 text-left transition-colors"
                    style="background: var(--bg); border: 2px solid {selectedIdeas.has(i) ? 'var(--coral)' : 'var(--border)'};"
                  >
                    <div class="flex items-start gap-3">
                      <div class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center mt-0.5" style="border: 2px solid {selectedIdeas.has(i) ? 'var(--coral)' : 'var(--border)'}; background: {selectedIdeas.has(i) ? 'var(--coral)' : 'transparent'};">
                        {#if selectedIdeas.has(i)}
                          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                        {/if}
                      </div>
                      <p
                        class="text-base leading-relaxed min-w-0 flex-1"
                        style="color: var(--text); display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden"
                      >
                        {draft.caption}
                      </p>
                    </div>
                  </button>
                {/each}
                {#if selectedIdeas.size > 0}
                  <div class="flex gap-2">
                    {#if selectedIdeas.size === 1}
                      <button
                        onclick={() => {
                          const idx = [...selectedIdeas][0];
                          result = ideaDrafts![idx];
                          isProactive = true;
                        }}
                        class="px-5 py-2.5 rounded-full text-sm font-medium"
                        style="background: var(--coral); color: #fff"
                      >Revisar e enviar</button>
                    {:else}
                      <button
                        onclick={sendSelectedIdeas}
                        disabled={sendingIdeas}
                        class="px-5 py-2.5 rounded-full text-sm font-medium"
                        style="background: #25D366; color: #fff; opacity: {sendingIdeas ? '0.6' : '1'}"
                      >{sendingIdeas ? 'Enviando...' : `Enviar ${selectedIdeas.size} selecionadas`}</button>
                    {/if}
                    <button
                      onclick={() => { selectedIdeas = new Set(); }}
                      class="px-4 py-2.5 rounded-full text-sm font-medium"
                      style="color: var(--destructive)"
                    >Limpar</button>
                  </div>
                {/if}
              </div>
            {/if}

            {#if inputMode === 'generate'}
              <p class="text-sm" style="color: var(--coral);">Toque nas mensagens que quer usar no post</p>
            {/if}
            {#if !blockReason}
              <!-- Action chips bar -->
              <div class="flex gap-2 items-center flex-wrap">
                {#if result && !showReviewOverlay}
                  <button
                    onclick={() => { showReviewOverlay = true; }}
                    class="text-sm px-3 py-1.5 rounded-full font-medium flex items-center gap-1.5"
                    style="background: var(--coral); color: #fff;"
                  >
                    Ver post gerado
                  </button>
                {/if}
                {#if inputMode === 'generate' && ideaDrafts === null}
                  {#if selectedMessages.size > 0}
                    <span class="text-sm px-3 py-1.5 rounded-full font-medium" style="background: var(--coral-pale); color: var(--coral);">
                      {selectedMessages.size} {selectedMessages.size === 1 ? 'mensagem selecionada' : 'mensagens selecionadas'}
                    </span>
                  {/if}
                  {#if recentContextIds.size > 0 && selectedMessages.size === 0}
                    <button
                      onclick={selectRecentMessages}
                      class="text-sm px-3 py-1.5 rounded-full font-medium"
                      style="background: var(--sage-pale); color: var(--text-secondary);"
                    >
                      Selecionar recentes
                    </button>
                  {/if}
                  {#if showGenerateIdeasButton}
                    <button
                      onclick={generateIdeas}
                      disabled={generatingIdeas || generating}
                      class="text-sm px-3 py-1.5 rounded-full font-medium flex items-center gap-1.5"
                      style="background: var(--sage-pale); color: var(--text-secondary); opacity: {generatingIdeas || generating ? '0.5' : '1'}"
                    >
                      {#if generatingIdeas}
                        <svg class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
                        Criando o post...
                      {:else}
                        3 ideias
                      {/if}
                    </button>
                  {/if}
                {/if}
                <button
                  onclick={() => { inputMode = inputMode === 'chat' ? 'generate' : 'chat'; message = ''; selectedMessages = new Set(); removeAttachment(); }}
                  class="ml-auto text-base px-4 py-1.5 min-h-12 rounded-full font-medium transition-colors flex items-center gap-1.5"
                  style="background: {inputMode === 'generate' ? '#25D366' : 'var(--coral)'}; color: #fff;"
                >
                  {#if inputMode === 'generate'}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                    Chat
                  {:else}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3l1.09 3.26L16.36 7.5l-3.27 1.15L12 12l-1.09-3.35L7.64 7.5l3.27-1.24L12 3z"/><path d="M5 16l.55 1.64L7.18 18.2 5.55 18.76 5 20.4l-.55-1.64L2.82 18.2l1.63-.56L5 16z"/><path d="M18 12l.55 1.64 1.63.56-1.63.56L18 16.4l-.55-1.64-1.63-.56 1.63-.56L18 12z"/></svg>
                    Post
                  {/if}
                </button>
              </div>
              <!-- Attachment preview -->
              {#if attachedPreview}
                <div class="flex items-center gap-2 px-1">
                  <div class="relative">
                    <img src={attachedPreview} alt="Anexo" class="w-16 h-16 rounded-lg object-cover border" style="border-color: var(--border-strong)" />
                    <button
                      onclick={removeAttachment}
                      class="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full flex items-center justify-center text-white text-xs"
                      style="background: var(--destructive)"
                      aria-label="Remover anexo"
                    >&times;</button>
                  </div>
                </div>
              {/if}
              <!-- Input row -->
              <div class="flex gap-2 items-center relative">
                <!-- Attach button -->
                <button
                  onclick={() => { showAttachMenu = !showAttachMenu; }}
                  disabled={sendingMedia}
                  class="shrink-0 w-10 h-10 rounded-full flex items-center justify-center transition-colors"
                  style="color: var(--text-muted); opacity: {sendingMedia ? '0.5' : '1'}"
                  aria-label="Anexar arquivo"
                >
                  {#if sendingMedia}
                    <svg class="animate-spin" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
                  {:else}
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
                  {/if}
                </button>
                <!-- Attach menu popup -->
                {#if showAttachMenu}
                  <!-- backdrop to close menu -->
                  <button class="fixed inset-0 z-10" onclick={() => { showAttachMenu = false; }} aria-label="Fechar menu"></button>
                  <div class="absolute bottom-12 left-0 z-20 rounded-xl shadow-lg border p-2 flex flex-col gap-1" style="background: var(--bg); border-color: var(--border-strong); min-width: 180px;">
                    <button
                      onclick={() => handleAttachFile("image/*")}
                      class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-left transition-colors hover:bg-black/5"
                      style="color: var(--text)"
                    >
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>
                      Galeria
                    </button>
                    <button
                      onclick={() => handleAttachFile("image/*", "environment")}
                      class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-left transition-colors hover:bg-black/5"
                      style="color: var(--text)"
                    >
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/></svg>
                      Camera
                    </button>
                  </div>
                {/if}
                <input
                  bind:value={message}
                  placeholder={inputMode === 'generate' ? 'Sobre o que é o post?' : 'Escreve aqui...'}
                  class="flex-1 min-w-0 px-3 md:px-4 py-3 rounded-xl text-base outline-none border"
                  style="border-color: var(--border-strong); background: var(--bg); color: var(--text); min-height: 48px;"
                  onkeydown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      if (inputMode === 'chat') sendQuickReply();
                      else if (message.trim() || selectedMessages.size > 0 || attachedFile) generate();
                    }
                  }}
                />
                {#if inputMode === 'generate'}
                  <button
                    onclick={generate}
                    disabled={generating || generatingIdeas || (!message.trim() && selectedMessages.size === 0 && !attachedFile)}
                    class="shrink-0 px-3 md:px-5 py-3 rounded-full text-sm font-medium transition-opacity flex items-center gap-2"
                    style="background: var(--coral); color: #fff; opacity: {generating || (!message.trim() && selectedMessages.size === 0 && !attachedFile) ? '0.6' : '1'}; cursor: {generating || (!message.trim() && selectedMessages.size === 0 && !attachedFile) ? 'not-allowed' : 'pointer'}"
                  >
                    {#if generating}
                      <svg class="animate-spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
                      Criando o post...
                    {:else}
                      Gerar
                    {/if}
                  </button>
                {:else}
                  <button
                    onclick={sendQuickReply}
                    disabled={sendingQuick || sendingMedia || (!message.trim() && !attachedFile)}
                    class="shrink-0 px-3 md:px-5 py-3 rounded-full text-sm font-medium transition-opacity"
                    style="background: #25D366; color: #fff; opacity: {sendingQuick || sendingMedia || (!message.trim() && !attachedFile) ? '0.6' : '1'}"
                  >
                    {sendingQuick ? "..." : "Enviar"}
                  </button>
                {/if}
              </div>
              {#if quickReplyError}
                <span class="text-sm" style="color: var(--destructive)">{quickReplyError}</span>
              {/if}
              {#if ideaError}
                <p class="text-sm" style="color: var(--destructive)">{ideaError}</p>
              {/if}
              {#if generateError}
                <p class="text-sm" style="color: var(--destructive)">{generateError}</p>
              {/if}
            {:else}
              <span class="text-sm" style="color: var(--text-muted)">{blockReason}</span>
            {/if}

          </div>
          </div>
        {:else}
          <div class="flex-1 flex items-center justify-center">
            <p class="text-base" style="color: var(--text-muted)">
              Escolhe uma cliente na lista pra começar.
            </p>
          </div>
        {/if}
      </div>
    </main>

    {#if !waConnected && waQR}
      <!-- QR overlay: non-blocking, appears on top of the main layout -->
      <div
        class="fixed inset-0 z-50 flex items-center justify-center"
        style="background: rgba(0,0,0,0.5)"
      >
        <div
          class="rounded-2xl p-8 text-center max-w-sm"
          style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
        >
          <h2 class="text-lg font-semibold mb-2" style="color: var(--text)">
            Conectar WhatsApp
          </h2>
          <p class="text-sm mb-6" style="color: var(--text-secondary)">
            Escaneie o QR code com o WhatsApp Business do Rekan.
          </p>
          <div class="bg-white p-4 rounded-xl inline-block">
            {#if qrDataUrl}
              <img src={qrDataUrl} alt="QR Code WhatsApp" width="256" height="256" />
            {:else}
              <div style="width: 256px; height: 256px" class="flex items-center justify-center">
                <span class="text-sm" style="color: var(--text-muted)">Conectando ao WhatsApp...</span>
              </div>
            {/if}
          </div>
          <p class="text-sm mt-4" style="color: var(--text-muted)">
            O QR code atualiza automaticamente.
          </p>
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Toast notification -->
{#if toastMessage}
  <div
    class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 px-4 py-3 rounded-xl text-sm font-medium shadow-lg"
    style="background: var(--surface); color: var(--text); border: 1px solid var(--border-strong);"
  >
    {toastMessage}
  </div>
{/if}

<style>
  @keyframes pulse {
    0%, 100% { transform: scale(1); opacity: 1; }
    50% { transform: scale(1.12); opacity: 0.75; }
  }
  @keyframes ring {
    0% { transform: scale(1); opacity: 0.6; }
    100% { transform: scale(1.7); opacity: 0; }
  }
  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.2; }
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  .mic-btn {
    width: 56px;
    height: 56px;
  }
  .rec-bar {
    height: 56px;
  }
  .rec-side-btn {
    width: 56px;
    height: 100%;
  }
  .rec-timer {
    font-size: 20px;
  }
  @media (max-width: 359px) {
    .rec-label {
      display: none;
    }
  }
  @media (min-width: 768px) {
    .mic-btn {
      width: 72px;
      height: 72px;
    }
    .rec-bar {
      height: 72px;
    }
    .rec-side-btn {
      width: 72px;
    }
    .rec-timer {
      font-size: 26px;
    }
  }
</style>

