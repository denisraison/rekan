<script lang="ts">
  import QRCode from "qrcode";
  import { onDestroy, onMount } from "svelte";
  import { pb } from "$lib/pb";
  import { readSSE } from "$lib/sse";
  import type {
    Business,
    GeneratedPost,
    Message,
    Post,
    Service,
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
    "AC",
    "AL",
    "AP",
    "AM",
    "BA",
    "CE",
    "DF",
    "ES",
    "GO",
    "MA",
    "MT",
    "MS",
    "MG",
    "PA",
    "PB",
    "PR",
    "PE",
    "PI",
    "RJ",
    "RN",
    "RS",
    "RO",
    "RR",
    "SC",
    "SP",
    "SE",
    "TO",
  ];

  const NUDGE_TEMPLATES = [
    {
      minDays: 5,
      maxDays: 7,
      template:
        "Oi {name}, como foi a semana? Tem algo legal pra gente postar?",
    },
    {
      minDays: 8,
      maxDays: 14,
      template:
        "{name}, tudo bem? Faz um tempinho que a gente nao posta. Bora preparar algo novo?",
    },
    {
      minDays: 15,
      maxDays: Infinity,
      template:
        "{name}, vi que faz um tempo! Quer retomar? Posso te mandar ideias de conteudo pra essa semana.",
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
      month: 2,
      day: 14,
      label: "Carnaval",
      niches: [
        "Salão de Beleza",
        "Barbearia",
        "Personal Trainer",
        "Nail Designer",
      ],
      template: "{name}, Carnaval ta chegando! Vamos preparar posts especiais?",
    },
    {
      month: 3,
      day: 8,
      label: "Dia da Mulher",
      niches: [
        "Salão de Beleza",
        "Nail Designer",
        "Confeitaria",
        "Loja de Roupas",
      ],
      template:
        "{name}, Dia da Mulher vem ai! Que tal um post com promo especial?",
    },
    {
      month: 4,
      day: 5,
      label: "Páscoa",
      niches: ["Confeitaria", "Restaurante", "Hamburgueria", "Loja de Açaí"],
      template:
        "{name}, Pascoa ta chegando! Vamos montar os posts das encomendas?",
    },
    {
      month: 5,
      day: 10,
      label: "Dia das Mães",
      niches: [
        "Salão de Beleza",
        "Confeitaria",
        "Nail Designer",
        "Loja de Roupas",
        "Restaurante",
      ],
      template:
        "{name}, Dia das Maes daqui a pouco! Bora preparar posts de presente e promo?",
    },
    {
      month: 6,
      day: 12,
      label: "Dia dos Namorados",
      niches: [
        "Confeitaria",
        "Restaurante",
        "Hamburgueria",
        "Salão de Beleza",
        "Loja de Roupas",
      ],
      template:
        "{name}, Dia dos Namorados vem ai! Vamos criar posts romanticos pro seu negocio?",
    },
    {
      month: 6,
      day: 13,
      label: "Festas Juninas",
      niches: ["Confeitaria", "Restaurante", "Hamburgueria", "Banda Musical"],
      template: "{name}, Junho ta ai! Vamos postar algo com tema junino?",
    },
    {
      month: 9,
      day: 1,
      label: "Dia do Educador Físico",
      niches: ["Personal Trainer"],
      template:
        "{name}, vem ai o Dia do Educador Fisico! Bora fazer um post especial?",
    },
    {
      month: 10,
      day: 1,
      label: "Início do Verão",
      niches: ["Personal Trainer", "Loja de Açaí"],
      template:
        "{name}, verao chegando! Momento perfeito pra postar sobre preparacao e resultados.",
    },
    {
      month: 10,
      day: 12,
      label: "Dia das Crianças",
      niches: ["Confeitaria", "Pet Shop", "Loja de Roupas"],
      template:
        "{name}, Dia das Criancas ta perto! Vamos criar posts com ofertas kids?",
    },
    {
      month: 12,
      day: 19,
      label: "Dia do Cabeleireiro",
      niches: ["Salão de Beleza", "Barbearia"],
      template:
        "{name}, Dia do Cabeleireiro chegando! Que tal um post especial celebrando a profissao?",
    },
    {
      month: 12,
      day: 25,
      label: "Natal",
      niches: [],
      template:
        "{name}, Natal chegando! Vamos preparar posts com ofertas e mensagem de final de ano?",
    },
    {
      month: 12,
      day: 31,
      label: "Réveillon",
      niches: [
        "Salão de Beleza",
        "Barbearia",
        "Nail Designer",
        "Personal Trainer",
        "Loja de Roupas",
      ],
      template:
        "{name}, Reveillon vem ai! Bora postar sobre agendamento e preparacao?",
    },
  ];

  let clients = $state<Business[]>([]);
  let selectedId = $state<string | null>(null);
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

  // Invite
  let inviteUrl = $state("");
  let inviteCopied = $state(false);
  let cancelling = $state(false);

  // Generation
  let message = $state("");
  let generating = $state(false);
  let generateError = $state("");
  let result = $state<GeneratedPost | null>(null);
  let copied = $state<string | null>(null);
  let sending = $state(false);
  let sendError = $state("");

  // Nudge / engagement
  let clientFilter = $state<"todos" | "inativos">("todos");
  let nudgeText = $state("");
  let sendingNudge = $state(false);
  let sendNudgeError = $state("");

  // Monthly summary
  let summaryText = $state("");
  let sendingSummary = $state(false);
  let sendSummaryError = $state("");

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

  // Latest incoming message for "Gerar post" pre-fill
  let latestIncoming = $derived.by(() => {
    const incoming = threadMessages.filter(
      (m) => m.direction === "incoming" && m.content,
    );
    return incoming.length > 0 ? incoming[incoming.length - 1] : null;
  });
  let latestIncomingText = $derived(latestIncoming?.content ?? "");

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

  let filteredClients = $derived(
    clientFilter === "inativos"
      ? sortedClients.filter((c) => {
          const h = clientHealth[c.id];
          return h && h.daysSinceMsg >= 5;
        })
      : sortedClients,
  );

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

  onMount(async () => {
    const [clientsRes] = await Promise.all([
      pb.collection("businesses").getList<Business>(1, 200, { sort: "name" }),
    ]);
    clients = clientsRes.items;
    loading = false;

    // Load all messages and posts
    await Promise.all([loadMessages(), loadPosts()]);

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

    connectWhatsAppStream();
  });

  let waAbortController: AbortController | null = null;
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

  onDestroy(() => {
    unsubscribeMessages?.();
    unsubscribeBusinesses?.();
    unsubscribePosts?.();
    waAbortController?.abort();
  });

  async function connectWhatsAppStream() {
    waAbortController = new AbortController();
    try {
      const res = await fetch(`${pb.baseUrl}/api/whatsapp/stream`, {
        headers: { Authorization: pb.authStore.token },
        signal: waAbortController.signal,
      });
      if (!res.body) return;
      await readSSE(res.body, (data: any) => {
        waConnected = data.connected;
        waQR = data.qr || "";
        waChecking = false;
      });
    } catch (err) {
      if (err instanceof Error && err.name === "AbortError") return;
      waConnected = false;
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

  function selectClient(id: string) {
    selectedId = id;
    result = null;
    generateError = "";
    sendNudgeError = "";
    sendSummaryError = "";
    // Mark as seen
    lastSeen = { ...lastSeen, [id]: new Date().toISOString() };
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
      nudgeText = tier.template.replace("{name}", client.name.split(" ")[0]);
    } else {
      nudgeText = "";
    }
    // Monthly summary
    summaryText = "";
    if (!client) return;
    const now = new Date();
    const thisMonthStart = new Date(now.getFullYear(), now.getMonth(), 1);
    const lastMonthStart = new Date(now.getFullYear(), now.getMonth() - 1, 1);
    const clientPosts = posts.filter((p) => p.business === id);
    const thisMonth = clientPosts.filter(
      (p) => new Date(p.created) >= thisMonthStart,
    ).length;
    const lastMonth = clientPosts.filter((p) => {
      const d = new Date(p.created);
      return d >= lastMonthStart && d < thisMonthStart;
    }).length;
    if (thisMonth === 0) return;
    const firstName = client.name.split(" ")[0];
    const monthName = now.toLocaleDateString("pt-BR", { month: "long" });
    let text = `*${firstName}, resumo de ${monthName}:* a gente criou *${thisMonth} post${thisMonth > 1 ? "s" : ""}* pro seu Instagram`;
    if (lastMonth > 0) {
      text += ` (contra ${lastMonth} em ${new Date(now.getFullYear(), now.getMonth() - 1, 1).toLocaleDateString("pt-BR", { month: "long" })})`;
    }
    text += `. Mes que vem vamos manter esse ritmo!`;
    summaryText = text;
  }

  function prefillGenerate() {
    message = latestIncomingText;
  }

  function mediaUrl(msg: Message): string {
    return pb.files.getURL(
      { id: msg.id, collectionId: msg.collectionId },
      msg.media,
    );
  }

  // --- Client form logic (unchanged) ---

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
  }

  function openNewForm() {
    resetForm();
    showForm = true;
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
    showForm = true;
  }

  function closeForm() {
    showForm = false;
    resetForm();
  }

  function addService() {
    formServices = [...formServices, { name: "", price_brl: 0 }];
  }

  function removeService(i: number) {
    formServices = formServices.filter((_: Service, idx: number) => idx !== i);
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

  async function copyInviteUrl() {
    await navigator.clipboard.writeText(inviteUrl);
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
    if (!selectedId || !message.trim()) return;
    generating = true;
    generateError = "";
    result = null;
    try {
      const payload: Record<string, string> = { message: message.trim() };
      if (latestIncoming && message.trim() === latestIncoming.content) {
        payload.message_id = latestIncoming.id;
      }
      const res = await pb.send(
        `/api/businesses/${selectedId}/posts:generateFromMessage`,
        {
          method: "POST",
          body: JSON.stringify(payload),
        },
      );
      result = res as GeneratedPost;
    } catch (err: unknown) {
      const e = err as { data?: { message?: string } };
      generateError =
        e?.data?.message ?? "Erro ao gerar conteúdo. Tente novamente.";
    } finally {
      generating = false;
    }
  }

  async function sendViaWhatsApp() {
    if (!selectedId || !result) return;
    sending = true;
    sendError = "";
    try {
      await pb.send("/api/messages:send", {
        method: "POST",
        body: JSON.stringify({
          business_id: selectedId,
          caption: result.caption,
          hashtags: result.hashtags.join(" "),
          production_note: result.production_note || "",
        }),
      });
      result = null;
      message = "";
    } catch {
      sendError = "Erro ao enviar. Tente novamente.";
    } finally {
      sending = false;
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

  async function sendSummary() {
    if (!selectedId || !summaryText.trim()) return;
    sendingSummary = true;
    sendSummaryError = "";
    try {
      await pb.send("/api/messages:send", {
        method: "POST",
        body: JSON.stringify({
          business_id: selectedId,
          caption: summaryText.trim(),
          hashtags: "",
          production_note: "",
        }),
      });
      summaryText = "";
    } catch {
      sendSummaryError = "Erro ao enviar resumo. Tente novamente.";
    } finally {
      sendingSummary = false;
    }
  }

  function prefillSeasonalMessage(template: string) {
    if (!selected) return;
    nudgeText = template.replace("{name}", selected.name.split(" ")[0]);
  }

  async function copyText(text: string, label: string) {
    await navigator.clipboard.writeText(text);
    copied = label;
    setTimeout(() => {
      copied = null;
    }, 2000);
  }
</script>

<div class="min-h-screen flex flex-col" style="background: var(--bg)">
  <header
    class="border-b px-6 py-4 flex items-center justify-between shrink-0"
    style="background: var(--surface); border-color: var(--border)"
  >
    <span
      class="font-semibold"
      style="color: var(--text); font-family: var(--font-primary)"
    >
      Rekan — Operador
    </span>
    <a
      href="/operador/whatsapp"
      class="text-xs px-2 py-1 rounded-full"
      style="background: {waConnected
        ? '#DEF7EC'
        : '#FDE8E8'}; color: {waConnected ? '#03543F' : '#9B1C1C'}"
    >
      WhatsApp {waConnected ? "conectado" : "desconectado"}
    </a>
  </header>

  {#if loading}
    <p class="text-sm p-6" style="color: var(--text-muted)">Carregando...</p>
  {:else if !waConnected && waQR}
    <!-- QR Code pairing screen -->
    <main class="flex-1 flex items-center justify-center p-6">
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
            <img
              src={qrDataUrl}
              alt="QR Code WhatsApp"
              width="256"
              height="256"
            />
          {:else}
            <div
              style="width: 256px; height: 256px"
              class="flex items-center justify-center"
            >
              <span class="text-sm" style="color: var(--text-muted)"
                >Carregando...</span
              >
            </div>
          {/if}
        </div>
        <p class="text-xs mt-4" style="color: var(--text-muted)">
          O QR code atualiza automaticamente.
        </p>
      </div>
    </main>
  {:else}
    <!-- Main operator layout -->
    <main class="flex-1 flex overflow-hidden">
      <!-- Left: Client list -->
      <div
        class="w-72 border-r flex flex-col shrink-0"
        style="border-color: var(--border); background: var(--surface)"
      >
        <div
          class="flex items-center justify-between p-4 border-b"
          style="border-color: var(--border)"
        >
          <div class="flex gap-1">
            <button
              onclick={() => {
                clientFilter = "todos";
              }}
              class="text-xs font-medium px-2.5 py-1 rounded-full transition-colors"
              style="background: {clientFilter === 'todos'
                ? 'var(--coral)'
                : 'transparent'}; color: {clientFilter === 'todos'
                ? '#fff'
                : 'var(--text-secondary)'}"
            >
              Todos
            </button>
            <button
              onclick={() => {
                clientFilter = "inativos";
              }}
              class="text-xs font-medium px-2.5 py-1 rounded-full transition-colors"
              style="background: {clientFilter === 'inativos'
                ? 'var(--coral)'
                : 'transparent'}; color: {clientFilter === 'inativos'
                ? '#fff'
                : 'var(--text-secondary)'}"
            >
              Inativos{inactiveCount > 0 ? ` (${inactiveCount})` : ""}
            </button>
          </div>
          <button
            onclick={openNewForm}
            class="text-xs font-medium px-2.5 py-1 rounded-full"
            style="background: var(--coral); color: #fff"
          >
            Novo
          </button>
        </div>

        <div class="flex-1 overflow-y-auto">
          {#if clients.length === 0}
            <p class="text-sm p-4" style="color: var(--text-muted)">
              Nenhum cliente cadastrado.
            </p>
          {:else}
            {#each filteredClients as client (client.id)}
              {@const unread = unreadCounts[client.id] || 0}
              {@const health = clientHealth[client.id]}
              <button
                onclick={() => selectClient(client.id)}
                class="w-full text-left px-4 py-3 border-b transition-colors"
                style="background: {selectedId === client.id
                  ? 'var(--coral-pale)'
                  : 'transparent'}; border-color: var(--border); color: var(--text)"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2 min-w-0">
                    {#if health}
                      <span
                        class="w-2 h-2 rounded-full shrink-0"
                        style="background: {health.color}"
                      ></span>
                    {/if}
                    <span class="font-medium text-sm truncate"
                      >{client.name}</span
                    >
                    {#if client.invite_status && client.invite_status !== 'draft'}
                      <span
                        class="text-xs px-1.5 py-0.5 rounded-full shrink-0"
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
                        <span class="text-xs shrink-0" title="Aceito há mais de 48h sem pagamento">&#9888;</span>
                      {/if}
                    {/if}
                  </div>
                  {#if unread > 0}
                    <span
                      class="text-xs font-bold px-1.5 py-0.5 rounded-full shrink-0 ml-2"
                      style="background: var(--coral); color: #fff"
                    >
                      {unread}
                    </span>
                  {/if}
                </div>
                <div class="flex items-center justify-between mt-0.5">
                  <span class="text-xs" style="color: var(--text-muted)"
                    >{client.type} — {client.city}/{client.state}</span
                  >
                  {#if health}
                    <span class="text-xs" style="color: var(--text-muted)">
                      {health.daysSinceMsg < 999
                        ? `${health.daysSinceMsg}d`
                        : ""}{health.postsThisMonth > 0
                        ? ` · ${health.postsThisMonth} posts`
                        : ""}
                    </span>
                  {/if}
                </div>
              </button>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Right: Thread or form -->
      <div class="flex-1 flex flex-col overflow-hidden">
        {#if showForm}
          <div class="flex-1 overflow-y-auto p-6">
            <div
              class="max-w-xl rounded-2xl p-6"
              style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
            >
              <h2 class="text-lg font-semibold mb-4" style="color: var(--text)">
                {editingId ? "Editar cliente" : "Novo cliente"}
              </h2>

              {#if formError}
                <p
                  class="text-sm mb-4 p-3 rounded-lg"
                  style="color: #DC2626; background: #FEF2F2"
                >
                  {formError}
                </p>
              {/if}

              <div class="flex flex-col gap-4">
                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Nome do cliente</span
                  >
                  <input
                    bind:value={formClientName}
                    placeholder="Ex: Ana Silva"
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  />
                </label>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Email do cliente</span
                  >
                  <input
                    bind:value={formClientEmail}
                    type="email"
                    placeholder="ana@email.com"
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  />
                </label>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Nome do negócio</span
                  >
                  <input
                    bind:value={formName}
                    placeholder="Nome do negócio"
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  />
                </label>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Tipo de negócio</span
                  >
                  <select
                    bind:value={formType}
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  >
                    <option value="">Selecione...</option>
                    {#each BUSINESS_TYPES as t}
                      <option value={t}>{t}</option>
                    {/each}
                  </select>
                </label>

                <div class="flex gap-3">
                  <label class="flex flex-col gap-1.5 flex-1">
                    <span class="text-sm font-medium" style="color: var(--text)"
                      >Cidade</span
                    >
                    <input
                      bind:value={formCity}
                      placeholder="Ex: São Paulo"
                      class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                      style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                    />
                  </label>
                  <label class="flex flex-col gap-1.5 w-24">
                    <span class="text-sm font-medium" style="color: var(--text)"
                      >Estado</span
                    >
                    <select
                      bind:value={formState}
                      class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                      style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                    >
                      <option value="">UF</option>
                      {#each STATES as stateCode}
                        <option value={stateCode}>{stateCode}</option>
                      {/each}
                    </select>
                  </label>
                </div>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Telefone WhatsApp</span
                  >
                  <input
                    bind:value={formPhone}
                    placeholder="5511999998888"
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  />
                </label>

                <div>
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Serviços</span
                  >
                  <div class="flex flex-col gap-2 mt-1.5">
                    {#each formServices as service, i}
                      <div class="flex gap-2 items-center">
                        <input
                          bind:value={service.name}
                          placeholder="Nome do serviço"
                          class="flex-1 px-3 py-2.5 rounded-xl text-sm outline-none border"
                          style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                        />
                        <div class="relative w-28">
                          <span
                            class="absolute left-3 top-1/2 -translate-y-1/2 text-sm"
                            style="color: var(--text-muted)">R$</span
                          >
                          <input
                            type="number"
                            bind:value={service.price_brl}
                            min="0"
                            class="w-full pl-9 pr-3 py-2.5 rounded-xl text-sm outline-none border"
                            style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                          />
                        </div>
                        {#if formServices.length > 1}
                          <button
                            onclick={() => removeService(i)}
                            class="p-1"
                            style="color: var(--text-muted)"
                            aria-label="Remover"
                          >
                            <svg
                              viewBox="0 0 20 20"
                              fill="currentColor"
                              width="16"
                              height="16"
                              ><path
                                fill-rule="evenodd"
                                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                                clip-rule="evenodd"
                              /></svg
                            >
                          </button>
                        {/if}
                      </div>
                    {/each}
                    <button
                      onclick={addService}
                      class="text-sm font-medium mt-1"
                      style="color: var(--primary)">+ Adicionar serviço</button
                    >
                  </div>
                </div>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Público-alvo <span
                      class="font-normal"
                      style="color: var(--text-muted)">— opcional</span
                    ></span
                  >
                  <input
                    bind:value={formTargetAudience}
                    placeholder="Ex: Mulheres de 25 a 40 anos"
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  />
                </label>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Estilo da marca <span
                      class="font-normal"
                      style="color: var(--text-muted)">— opcional</span
                    ></span
                  >
                  <input
                    bind:value={formBrandVibe}
                    placeholder="Ex: Descontraido e moderno"
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  />
                </label>

                <label class="flex flex-col gap-1.5">
                  <span class="text-sm font-medium" style="color: var(--text)"
                    >Diferenciais <span
                      class="font-normal"
                      style="color: var(--text-muted)">— opcional</span
                    ></span
                  >
                  <textarea
                    bind:value={formQuirks}
                    placeholder="Ex: Atendimento por WhatsApp, parcela em 3x"
                    rows={2}
                    class="px-3 py-2.5 rounded-xl text-sm outline-none border resize-none"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  ></textarea>
                </label>
              </div>

              {#if inviteUrl}
                <div class="mt-4 rounded-xl p-4" style="background: var(--sage-pale); border: 1px solid var(--border)">
                  <p class="text-sm font-medium mb-2" style="color: var(--text)">Convite enviado!</p>
                  <div class="flex items-center gap-2">
                    <input
                      readonly
                      value={inviteUrl}
                      class="flex-1 px-3 py-2 rounded-lg text-xs outline-none border"
                      style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                    />
                    <button
                      onclick={copyInviteUrl}
                      class="px-3 py-2 rounded-lg text-xs font-medium"
                      style="background: var(--coral); color: #fff"
                    >
                      {inviteCopied ? "Copiado!" : "Copiar"}
                    </button>
                  </div>
                </div>
              {/if}

              <div class="flex gap-3 mt-6">
                <button
                  onclick={closeForm}
                  class="px-5 py-2.5 rounded-full text-sm font-medium border"
                  style="border-color: var(--border-strong); color: var(--text-secondary)"
                  >Cancelar</button
                >
                <button
                  onclick={saveClient}
                  disabled={formSaving}
                  class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity"
                  style="background: var(--coral); color: #fff; opacity: {formSaving
                    ? '0.6'
                    : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}"
                >
                  {formSaving ? "Salvando..." : "Salvar"}
                </button>
                <button
                  onclick={saveAndInvite}
                  disabled={formSaving}
                  class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity"
                  style="background: #25D366; color: #fff; opacity: {formSaving
                    ? '0.6'
                    : '1'}; cursor: {formSaving ? 'not-allowed' : 'pointer'}"
                >
                  {formSaving ? "Salvando..." : "Salvar e Enviar Convite"}
                </button>
              </div>
            </div>
          </div>
        {:else if selected}
          <!-- Client header -->
          <div
            class="px-6 py-4 border-b flex items-center justify-between shrink-0"
            style="border-color: var(--border); background: var(--surface)"
          >
            <div>
              <h2 class="text-sm font-semibold" style="color: var(--text)">
                {selected.name}
              </h2>
              <p class="text-xs" style="color: var(--text-secondary)">
                {selected.type} — {selected.city}/{selected.state}
              </p>
            </div>
            <div class="flex items-center gap-2">
              {#if selected.type === 'Desconhecido'}
                <button
                  onclick={() => openEditForm(selected!)}
                  class="text-xs px-3 py-1.5 rounded-full font-medium"
                  style="background: #25D366; color: #fff"
                >Criar conta</button>
              {/if}
              {#if selected.invite_status === 'active'}
                <button
                  onclick={cancelSubscription}
                  disabled={cancelling}
                  class="text-xs px-3 py-1.5 rounded-full border transition-opacity"
                  style="border-color: #EF4444; color: #EF4444; opacity: {cancelling ? '0.6' : '1'}"
                >
                  {cancelling ? "Cancelando..." : "Cancelar assinatura"}
                </button>
              {/if}
              <button
                onclick={() => openEditForm(selected!)}
                class="text-xs px-3 py-1.5 rounded-full border"
                style="border-color: var(--border-strong); color: var(--text-secondary)"
                >Editar</button
              >
            </div>
          </div>

          <!-- Engagement panel (nudge + seasonal) -->
          {#if nudgeTier || upcomingDates.length > 0 || nudgeText || summaryText}
            <div
              class="shrink-0 px-6 py-3 border-b"
              style="border-color: var(--border); background: var(--bg)"
            >
              {#if nudgeTier || nudgeText}
                <div class="mb-2">
                  <span
                    class="text-xs font-medium uppercase tracking-widest"
                    style="color: var(--text-muted)"
                  >
                    {#if nudgeTier}
                      Lembrete · {clientHealth[selected!.id]?.daysSinceMsg} dias sem
                      mensagem
                    {:else}
                      Mensagem
                    {/if}
                  </span>
                  <textarea
                    bind:value={nudgeText}
                    rows={2}
                    class="w-full mt-1.5 px-3 py-2 rounded-xl text-sm outline-none border resize-none"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  ></textarea>
                  <div class="flex items-center gap-2 mt-1.5">
                    <button
                      onclick={sendNudge}
                      disabled={sendingNudge ||
                        !nudgeText.trim() ||
                        !waConnected ||
                        !selected?.phone}
                      class="px-3 py-1.5 rounded-full text-xs font-medium transition-opacity"
                      style="background: #25D366; color: #fff; opacity: {sendingNudge ||
                      !nudgeText.trim() ||
                      !waConnected ||
                      !selected?.phone
                        ? '0.6'
                        : '1'}; cursor: {sendingNudge ||
                      !nudgeText.trim() ||
                      !waConnected ||
                      !selected?.phone
                        ? 'not-allowed'
                        : 'pointer'}"
                    >
                      {sendingNudge ? "Enviando..." : "Enviar lembrete"}
                    </button>
                    {#if sendNudgeError}
                      <span class="text-xs" style="color: var(--destructive)"
                        >{sendNudgeError}</span
                      >
                    {/if}
                  </div>
                </div>
              {/if}

              {#if upcomingDates.length > 0}
                <div
                  class={nudgeTier || nudgeText ? "pt-2 border-t" : ""}
                  style="border-color: var(--border)"
                >
                  <span
                    class="text-xs font-medium uppercase tracking-widest"
                    style="color: var(--text-muted)"
                  >
                    Datas próximas
                  </span>
                  <div class="flex flex-wrap gap-1.5 mt-1.5">
                    {#each upcomingDates as sd}
                      <button
                        onclick={() => prefillSeasonalMessage(sd.template)}
                        class="text-xs px-2.5 py-1 rounded-full border transition-colors hover:bg-(--coral-pale)"
                        style="border-color: var(--border-strong); color: var(--text-secondary)"
                        title={sd.template.replace(
                          "{name}",
                          selected!.name.split(" ")[0],
                        )}
                      >
                        {sd.label} · {sd.daysUntil}d
                      </button>
                    {/each}
                  </div>
                </div>
              {/if}

              {#if summaryText}
                <div
                  class={nudgeTier || nudgeText || upcomingDates.length > 0
                    ? "pt-2 border-t"
                    : ""}
                  style="border-color: var(--border)"
                >
                  <span
                    class="text-xs font-medium uppercase tracking-widest"
                    style="color: var(--text-muted)"
                  >
                    Resumo mensal · {clientHealth[selected!.id]
                      ?.postsThisMonth ?? 0} posts este mês
                  </span>
                  <textarea
                    bind:value={summaryText}
                    rows={3}
                    class="w-full mt-1.5 px-3 py-2 rounded-xl text-sm outline-none border resize-none"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text)"
                  ></textarea>
                  <div class="flex items-center gap-2 mt-1.5">
                    <button
                      onclick={sendSummary}
                      disabled={sendingSummary ||
                        !summaryText.trim() ||
                        !waConnected ||
                        !selected?.phone}
                      class="px-3 py-1.5 rounded-full text-xs font-medium transition-opacity"
                      style="background: #25D366; color: #fff; opacity: {sendingSummary ||
                      !summaryText.trim() ||
                      !waConnected ||
                      !selected?.phone
                        ? '0.6'
                        : '1'}; cursor: {sendingSummary ||
                      !summaryText.trim() ||
                      !waConnected ||
                      !selected?.phone
                        ? 'not-allowed'
                        : 'pointer'}"
                    >
                      {sendingSummary ? "Enviando..." : "Enviar resumo"}
                    </button>
                    {#if sendSummaryError}
                      <span class="text-xs" style="color: var(--destructive)"
                        >{sendSummaryError}</span
                      >
                    {/if}
                  </div>
                </div>
              {/if}
            </div>
          {/if}

          <!-- Message thread -->
          <div class="flex-1 overflow-y-auto px-6 py-4 flex flex-col gap-3">
            {#if threadMessages.length === 0}
              <p
                class="text-sm text-center py-8"
                style="color: var(--text-muted)"
              >
                Nenhuma mensagem ainda.
              </p>
            {:else}
              {#each threadMessages as msg (msg.id)}
                <div
                  class="flex {msg.direction === 'outgoing'
                    ? 'justify-end'
                    : 'justify-start'}"
                >
                  <div
                    class="max-w-md rounded-2xl px-4 py-2.5 text-sm"
                    style="background: {msg.direction === 'outgoing'
                      ? 'var(--coral-pale)'
                      : 'var(--surface)'}; border: 1px solid {msg.direction ===
                    'outgoing'
                      ? 'var(--coral-light)'
                      : 'var(--border)'}; color: var(--text)"
                  >
                    {#if msg.type === "audio"}
                      <span
                        class="text-xs font-medium block mb-1"
                        style="color: var(--text-muted)">Áudio transcrito</span
                      >
                    {/if}

                    {#if msg.type === "image" && msg.media}
                      <img
                        src={mediaUrl(msg)}
                        alt="Imagem do cliente"
                        class="rounded-xl mb-2 max-w-full"
                        style="max-height: 300px"
                      />
                    {/if}

                    {#if msg.content}
                      <p class="whitespace-pre-wrap">{msg.content}</p>
                    {:else if msg.type === "audio"}
                      <p class="italic" style="color: var(--text-muted)">
                        Transcrição indisponível
                      </p>
                    {/if}

                    <span
                      class="text-xs block mt-1"
                      style="color: var(--text-muted)"
                    >
                      {new Date(
                        msg.wa_timestamp || msg.created,
                      ).toLocaleTimeString("pt-BR", {
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
                    </span>
                  </div>
                </div>
              {/each}
            {/if}
          </div>

          <!-- Generate panel (bottom) -->
          <div
            class="shrink-0 border-t px-6 py-4"
            style="border-color: var(--border); background: var(--surface)"
          >
            <div class="flex gap-2 items-end">
              <div class="flex-1">
                <textarea
                  bind:value={message}
                  placeholder="Mensagem do cliente para gerar post..."
                  rows={2}
                  class="w-full px-3 py-2.5 rounded-xl text-sm outline-none border resize-none"
                  style="border-color: var(--border-strong); background: var(--bg); color: var(--text)"
                ></textarea>
              </div>
              <div class="flex flex-col gap-1.5">
                {#if latestIncomingText && message !== latestIncomingText}
                  <button
                    onclick={prefillGenerate}
                    class="text-xs px-3 py-1.5 rounded-full border"
                    style="border-color: var(--border-strong); color: var(--text-secondary)"
                  >
                    Usar última msg
                  </button>
                {/if}
                <button
                  onclick={generate}
                  disabled={generating || !message.trim()}
                  class="px-4 py-2.5 rounded-full text-sm font-medium transition-opacity"
                  style="background: var(--coral); color: #fff; opacity: {generating ||
                  !message.trim()
                    ? '0.6'
                    : '1'}; cursor: {generating || !message.trim()
                    ? 'not-allowed'
                    : 'pointer'}"
                >
                  {generating ? "Gerando..." : "Gerar post"}
                </button>
              </div>
            </div>

            {#if generateError}
              <p class="mt-2 text-sm" style="color: var(--destructive)">
                {generateError}
              </p>
            {/if}

            {#if result}
              <div
                class="mt-4 rounded-xl p-4"
                style="background: var(--bg); border: 1px solid var(--border)"
              >
                <div class="mb-3">
                  <div class="flex items-center justify-between mb-1">
                    <span
                      class="text-xs font-medium uppercase tracking-widest"
                      style="color: var(--text-muted)">Legenda</span
                    >
                    <button
                      onclick={() => copyText(result!.caption, "caption")}
                      class="text-xs"
                      style="color: var(--coral)"
                    >
                      {copied === "caption" ? "Copiado!" : "Copiar"}
                    </button>
                  </div>
                  <p
                    class="text-sm leading-relaxed whitespace-pre-wrap"
                    style="color: var(--text)"
                  >
                    {result.caption}
                  </p>
                </div>

                <div class="mb-3">
                  <div class="flex items-center justify-between mb-1">
                    <span
                      class="text-xs font-medium uppercase tracking-widest"
                      style="color: var(--text-muted)">Hashtags</span
                    >
                    <button
                      onclick={() =>
                        copyText(result!.hashtags.join(" "), "hashtags")}
                      class="text-xs"
                      style="color: var(--coral)"
                    >
                      {copied === "hashtags" ? "Copiado!" : "Copiar"}
                    </button>
                  </div>
                  <p class="text-xs" style="color: var(--text-secondary)">
                    {result.hashtags.join(" ")}
                  </p>
                </div>

                {#if result.production_note}
                  <div>
                    <span
                      class="text-xs font-medium uppercase tracking-widest"
                      style="color: var(--text-muted)">Nota de produção</span
                    >
                    <p
                      class="text-xs italic mt-1"
                      style="color: var(--text-secondary); border-left: 2px solid var(--border-strong); padding-left: 0.75rem"
                    >
                      {result.production_note}
                    </p>
                  </div>
                {/if}

                {#if waConnected && selected?.phone}
                  <div
                    class="flex items-center gap-2 mt-4 pt-3 border-t"
                    style="border-color: var(--border)"
                  >
                    <button
                      onclick={sendViaWhatsApp}
                      disabled={sending}
                      class="px-4 py-2 rounded-full text-sm font-medium transition-opacity"
                      style="background: #25D366; color: #fff; opacity: {sending
                        ? '0.6'
                        : '1'}; cursor: {sending ? 'not-allowed' : 'pointer'}"
                    >
                      {sending ? "Enviando..." : "Enviar pelo WhatsApp"}
                    </button>
                    {#if sendError}
                      <span class="text-xs" style="color: var(--destructive)"
                        >{sendError}</span
                      >
                    {/if}
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {:else}
          <div class="flex-1 flex items-center justify-center">
            <p class="text-sm" style="color: var(--text-muted)">
              Selecione um cliente para ver as mensagens.
            </p>
          </div>
        {/if}
      </div>
    </main>
  {/if}
</div>
