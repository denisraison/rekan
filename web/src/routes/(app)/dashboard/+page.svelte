<script lang="ts">
  import { onMount } from "svelte";
  import { pb } from "$lib/pb";
  import type { Business, Post, User } from "$lib/types";

  const trialLimit = 3;

  let user = $state<User | null>(null);
  let business = $state<Business | null>(null);
  let posts = $state<Post[]>([]);
  let loading = $state(true);
  let generating = $state(false);
  let generateError = $state("");
  let trialExhausted = $state(false);
  let subscribing = $state(false);
  let editStates = $state<
    Record<string, { caption: string; hashtags: string }>
  >({});

  let batches = $derived(groupByBatch(posts));
  let generationsLeft = $derived(
    user?.subscription_status === "trial"
      ? trialLimit - (user?.generations_used ?? 0)
      : null,
  );

  function groupByBatch(
    all: Post[],
  ): Array<{ batchId: string; date: string; posts: Post[] }> {
    const map = new Map<string, Post[]>();
    for (const post of all) {
      if (!map.has(post.batch_id)) map.set(post.batch_id, []);
      map.get(post.batch_id)!.push(post);
    }
    return [...map.entries()].map(([batchId, batchPosts]) => ({
      batchId,
      date: new Date(batchPosts[0].created).toLocaleDateString("pt-BR", {
        day: "numeric",
        month: "long",
        year: "numeric",
      }),
      posts: batchPosts,
    }));
  }

  onMount(async () => {
    try {
      await pb.collection("users").authRefresh();
      user = pb.authStore.model as unknown as User;
      const result = await pb.collection("businesses").getList<Business>(1, 1);
      business = result.items[0] ?? null;
      if (business) await loadPosts();
    } finally {
      loading = false;
    }
  });

  async function loadPosts() {
    if (!business) return;
    const result = await pb.collection("posts").getList<Post>(1, 100, {
      filter: `business = "${business.id}"`,
      sort: "-created",
    });
    posts = result.items;
  }

  async function _generate() {
    if (!business) return;
    generating = true;
    generateError = "";
    trialExhausted = false;
    try {
      await pb.send(`/api/businesses/${business.id}/posts:generate`, {
        method: "POST",
      });
      await pb.collection("users").authRefresh();
      user = pb.authStore.model as unknown as User;
      await loadPosts();
    } catch (err: unknown) {
      const e = err as { status?: number; data?: { message?: string } };
      if (e?.status === 402) {
        trialExhausted = true;
      } else {
        generateError =
          e?.data?.message ?? "Erro ao gerar conteúdo. Tente novamente.";
      }
    } finally {
      generating = false;
    }
  }

  async function _subscribe() {
    subscribing = true;
    try {
      const res = await pb.send("/api/subscriptions", {
        method: "POST",
        body: JSON.stringify({ billing_type: "PIX" }),
      });
      window.location.href = res.payment_url;
    } catch (err: unknown) {
      const e = err as { data?: { message?: string } };
      generateError =
        e?.data?.message ?? "Erro ao iniciar assinatura. Tente novamente.";
    } finally {
      subscribing = false;
    }
  }

  function _startEdit(post: Post) {
    editStates[post.id] = {
      caption: post.caption,
      hashtags: post.hashtags.join(" "),
    };
  }

  function _cancelEdit(postId: string) {
    delete editStates[postId];
  }

  async function _saveEdit(post: Post) {
    const state = editStates[post.id];
    if (!state) return;
    const hashtags = state.hashtags.split(/\s+/).filter(Boolean);
    await pb
      .collection("posts")
      .update(post.id, { caption: state.caption, hashtags, edited: true });
    const idx = posts.findIndex((p) => p.id === post.id);
    if (idx >= 0)
      posts[idx] = {
        ...posts[idx],
        caption: state.caption,
        hashtags,
        edited: true,
      };
    delete editStates[post.id];
  }

  async function _deletePost(postId: string) {
    if (!confirm("Excluir este post?")) return;
    await pb.collection("posts").delete(postId);
    posts = posts.filter((p) => p.id !== postId);
  }

  async function _logout() {
    pb.authStore.clear();
    window.location.href = "/login";
  }
</script>

<div class="min-h-screen" style="background: var(--bg)">
  <header
    class="border-b px-6 py-4 flex items-center justify-between"
    style="background: var(--surface); border-color: var(--border)"
  >
    <span
      class="font-semibold"
      style="color: var(--text); font-family: var(--font-primary)">Rekan</span
    >
    <button
      onclick={logout}
      class="text-sm"
      style="color: var(--text-secondary); font-family: var(--font-primary)"
    >
      Sair
    </button>
  </header>

  <main class="max-w-2xl mx-auto px-6 py-12">
    {#if loading}
      <p class="text-sm" style="color: var(--text-muted)">Carregando...</p>
    {:else if business}
      <div class="mb-8">
        <h1
          class="text-2xl font-semibold mb-1"
          style="color: var(--text); font-family: var(--font-primary)"
        >
          {business.name}
        </h1>
        <p class="text-sm" style="color: var(--text-secondary)">
          {business.type} · {business.city}, {business.state}
        </p>
      </div>

      {#if user?.subscription_status === "trial" && generationsLeft !== null}
        <div
          class="rounded-xl px-4 py-3 mb-6 flex items-center justify-between"
          style="background: var(--sage-pale); border: 1px solid var(--border)"
        >
          <p
            class="text-sm"
            style="color: var(--text-secondary); font-family: var(--font-primary)"
          >
            {generationsLeft > 0
              ? `${generationsLeft} geração${generationsLeft === 1 ? "" : "ões"} gratuita${generationsLeft === 1 ? "" : "s"} restante${generationsLeft === 1 ? "" : "s"}`
              : "Período de teste encerrado"}
          </p>
          {#if generationsLeft === 0}
            <button
              onclick={subscribe}
              disabled={subscribing}
              class="text-sm font-medium px-4 py-1.5 rounded-full transition-opacity"
              style="background: var(--coral); color: #fff; font-family: var(--font-primary); opacity: {subscribing
                ? '0.6'
                : '1'}; cursor: {subscribing ? 'not-allowed' : 'pointer'}"
            >
              {subscribing ? "Aguarde..." : "Assinar"}
            </button>
          {/if}
        </div>
      {/if}

      {#if user?.subscription_status === "past_due"}
        <div
          class="rounded-xl px-4 py-3 mb-6 flex items-center justify-between"
          style="background: #fff3cd; border: 1px solid #ffc107"
        >
          <p
            class="text-sm"
            style="color: #856404; font-family: var(--font-primary)"
          >
            Pagamento pendente. Renove sua assinatura para continuar.
          </p>
          <button
            onclick={subscribe}
            disabled={subscribing}
            class="text-sm font-medium px-4 py-1.5 rounded-full ml-4"
            style="background: var(--coral); color: #fff; font-family: var(--font-primary)"
          >
            {subscribing ? "Aguarde..." : "Renovar"}
          </button>
        </div>
      {/if}

      {#if user?.subscription_status === "cancelled"}
        <div
          class="rounded-xl px-4 py-3 mb-6 flex items-center justify-between"
          style="background: var(--surface); border: 1px solid var(--border)"
        >
          <p
            class="text-sm"
            style="color: var(--text-secondary); font-family: var(--font-primary)"
          >
            Assinatura cancelada.
          </p>
          <button
            onclick={subscribe}
            disabled={subscribing}
            class="text-sm font-medium px-4 py-1.5 rounded-full ml-4"
            style="background: var(--coral); color: #fff; font-family: var(--font-primary)"
          >
            {subscribing ? "Aguarde..." : "Reativar"}
          </button>
        </div>
      {/if}

      <div class="mb-10">
        {#if trialExhausted}
          <div
            class="rounded-xl p-5"
            style="background: var(--surface); border: 1px solid var(--border)"
          >
            <p
              class="text-sm mb-3"
              style="color: var(--text); font-family: var(--font-primary)"
            >
              Período de teste encerrado. Assine para continuar gerando
              conteúdo.
            </p>
            <button
              onclick={subscribe}
              disabled={subscribing}
              class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity"
              style="background: var(--coral); color: #fff; font-family: var(--font-primary); opacity: {subscribing
                ? '0.6'
                : '1'}; cursor: {subscribing ? 'not-allowed' : 'pointer'}"
            >
              {subscribing ? "Aguarde..." : "Assinar — R$ 19 no primeiro mês"}
            </button>
          </div>
        {:else}
          <button
            onclick={generate}
            disabled={generating}
            class="px-5 py-2.5 rounded-full text-sm font-medium transition-opacity"
            style="background: var(--coral); color: #fff; font-family: var(--font-primary); opacity: {generating
              ? '0.6'
              : '1'}; cursor: {generating ? 'not-allowed' : 'pointer'}"
          >
            {generating ? "Gerando..." : "Gerar posts"}
          </button>
          {#if generateError}
            <p class="mt-3 text-sm" style="color: var(--destructive)">
              {generateError}
            </p>
          {/if}
        {/if}
      </div>

      {#if batches.length > 0}
        <div class="space-y-10">
          {#each batches as batch}
            <section>
              <p
                class="text-xs font-medium uppercase tracking-widest mb-4"
                style="color: var(--text-muted)"
              >
                {batch.date}
              </p>
              <div class="space-y-4">
                {#each batch.posts as post (post.id)}
                  {@const editing = editStates[post.id]}
                  <div
                    class="rounded-2xl p-6"
                    style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
                  >
                    <div class="flex items-center gap-2 mb-3">
                      <span
                        class="text-xs font-medium px-2 py-0.5 rounded-full"
                        style="background: var(--coral-pale); color: var(--coral-dark)"
                      >
                        {post.role}
                      </span>
                      {#if post.edited}
                        <span
                          class="text-xs px-2 py-0.5 rounded-full"
                          style="background: var(--sage-pale); color: var(--sage-dark)"
                        >
                          editado
                        </span>
                      {/if}
                    </div>

                    {#if editing}
                      <textarea
                        bind:value={editing.caption}
                        rows={6}
                        class="w-full rounded-lg px-3 py-2 text-sm resize-none mb-3"
                        style="border: 1px solid var(--border-strong); color: var(--text); font-family: var(--font-primary); background: var(--bg)"
                      ></textarea>
                      <input
                        bind:value={editing.hashtags}
                        placeholder="#hashtag1 #hashtag2"
                        class="w-full rounded-lg px-3 py-2 text-sm mb-4"
                        style="border: 1px solid var(--border-strong); color: var(--text-secondary); font-family: var(--font-primary); background: var(--bg)"
                      />
                      <div class="flex gap-2">
                        <button
                          onclick={() => saveEdit(post)}
                          class="px-4 py-1.5 rounded-full text-sm font-medium"
                          style="background: var(--coral); color: #fff; font-family: var(--font-primary)"
                        >
                          Salvar
                        </button>
                        <button
                          onclick={() => cancelEdit(post.id)}
                          class="px-4 py-1.5 rounded-full text-sm"
                          style="color: var(--text-secondary); font-family: var(--font-primary)"
                        >
                          Cancelar
                        </button>
                      </div>
                    {:else}
                      <p
                        class="text-sm leading-relaxed mb-3"
                        style="color: var(--text)"
                      >
                        {post.caption}
                      </p>
                      {#if post.hashtags?.length}
                        <p
                          class="text-xs mb-4"
                          style="color: var(--text-muted)"
                        >
                          {post.hashtags.join(" ")}
                        </p>
                      {/if}
                      {#if post.production_note}
                        <p
                          class="text-xs italic mb-4"
                          style="color: var(--text-muted); border-left: 2px solid var(--border-strong); padding-left: 0.75rem"
                        >
                          {post.production_note}
                        </p>
                      {/if}
                      <div class="flex gap-3">
                        <button
                          onclick={() => startEdit(post)}
                          class="text-xs"
                          style="color: var(--text-secondary); font-family: var(--font-primary)"
                        >
                          Editar
                        </button>
                        <button
                          onclick={() => deletePost(post.id)}
                          class="text-xs"
                          style="color: var(--destructive); font-family: var(--font-primary)"
                        >
                          Excluir
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            </section>
          {/each}
        </div>
      {:else}
        <div
          class="rounded-2xl p-8 text-center"
          style="background: var(--surface); border: 1px solid var(--border); box-shadow: var(--shadow-sm)"
        >
          <p class="text-sm" style="color: var(--text-muted)">
            Nenhum post gerado ainda. Clique em "Gerar posts" para começar.
          </p>
        </div>
      {/if}
    {:else}
      <p class="text-sm" style="color: var(--text-muted)">
        Nenhum negócio encontrado.
      </p>
    {/if}
  </main>
</div>
