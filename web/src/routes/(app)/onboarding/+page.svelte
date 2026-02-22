<script lang="ts">
  import { goto } from "$app/navigation";
  import { pb } from "$lib/pb";
  import type { Service } from "$lib/types";

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

  let step = $state(1);
  let loading = $state(false);
  let error = $state("");
  let businessId = $state("");

  // Step 1
  let name = $state("");
  let type = $state("");
  let city = $state("");
  let uf = $state("");

  // Step 2
  let services: Service[] = $state([{ name: "", price_brl: 0 }]);

  // Step 3
  let target_audience = $state("");
  let brand_vibe = $state("");
  let quirks = $state("");

  function addService() {
    services = [...services, { name: "", price_brl: 0 }];
  }

  function removeService(i: number) {
    services = services.filter((_: Service, idx: number) => idx !== i);
  }

  async function submitStep1() {
    if (!name.trim() || !type || !city.trim() || !uf) {
      error = "Preencha todos os campos.";
      return;
    }
    error = "";
    loading = true;
    try {
      const record = await pb.collection("businesses").create({
        user: pb.authStore.record!.id,
        name: name.trim(),
        type,
        city: city.trim(),
        state: uf,
        onboarding_step: 1,
      });
      businessId = record.id;
      step = 2;
    } catch (_e) {
      error = "Erro ao salvar. Tente novamente.";
    } finally {
      loading = false;
    }
  }

  async function submitStep2() {
    const valid = services.every(
      (s: Service) => s.name.trim() && s.price_brl > 0,
    );
    if (!valid || services.length === 0) {
      error = "Adicione pelo menos um serviço com nome e preço.";
      return;
    }
    error = "";
    loading = true;
    try {
      await pb.collection("businesses").update(businessId, {
        services,
        onboarding_step: 2,
      });
      step = 3;
    } catch (_e) {
      error = "Erro ao salvar. Tente novamente.";
    } finally {
      loading = false;
    }
  }

  async function submitStep3(skip = false) {
    error = "";
    loading = true;
    try {
      await pb.collection("businesses").update(businessId, {
        target_audience: skip
          ? "Público geral"
          : target_audience.trim() || "Público geral",
        brand_vibe: skip
          ? "Profissional e acolhedor"
          : brand_vibe.trim() || "Profissional e acolhedor",
        quirks: skip ? "" : quirks.trim(),
        onboarding_step: 3,
      });
      goto("/dashboard");
    } catch (_e) {
      error = "Erro ao salvar. Tente novamente.";
      loading = false;
    }
  }
</script>

<div class="min-h-screen py-12 px-4" style="background: var(--bg)">
  <div class="max-w-lg mx-auto">
    <!-- Progress -->
    <div class="flex items-center gap-2 mb-8">
      {#each [1, 2, 3] as s}
        <div
          class="h-1 flex-1 rounded-full transition-colors"
          style="background: {s <= step
            ? 'var(--primary)'
            : 'var(--border-strong)'}"
        ></div>
      {/each}
    </div>

    {#if step === 1}
      <div
        class="rounded-2xl p-8"
        style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
      >
        <h1
          class="text-xl font-semibold mb-1"
          style="color: var(--text); font-family: var(--font-primary)"
        >
          Sobre o seu negócio
        </h1>
        <p class="text-sm mb-6" style="color: var(--text-secondary)">
          Etapa 1 de 3
        </p>

        {#if error}
          <p
            class="text-sm mb-4 p-3 rounded-lg"
            style="color: #DC2626; background: #FEF2F2"
          >
            {error}
          </p>
        {/if}

        <div class="flex flex-col gap-4">
          <label class="flex flex-col gap-1.5">
            <span class="text-sm font-medium" style="color: var(--text)"
              >Nome do negócio</span
            >
            <input
              bind:value={name}
              placeholder="Ex: Salão da Ana"
              class="px-3 py-2.5 rounded-xl text-sm outline-none border transition-colors"
              style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
            />
          </label>

          <label class="flex flex-col gap-1.5">
            <span class="text-sm font-medium" style="color: var(--text)"
              >Tipo de negócio</span
            >
            <select
              bind:value={type}
              class="px-3 py-2.5 rounded-xl text-sm outline-none border transition-colors"
              style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
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
                bind:value={city}
                placeholder="Ex: São Paulo"
                class="px-3 py-2.5 rounded-xl text-sm outline-none border transition-colors"
                style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
              />
            </label>

            <label class="flex flex-col gap-1.5 w-24">
              <span class="text-sm font-medium" style="color: var(--text)"
                >Estado</span
              >
              <select
                bind:value={uf}
                class="px-3 py-2.5 rounded-xl text-sm outline-none border transition-colors"
                style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
              >
                <option value="">UF</option>
                {#each STATES as stateCode}
                  <option value={stateCode}>{stateCode}</option>
                {/each}
              </select>
            </label>
          </div>
        </div>

        <button
          onclick={submitStep1}
          disabled={loading}
          class="w-full mt-6 py-3 rounded-xl font-medium text-sm transition-opacity disabled:opacity-60"
          style="background: var(--primary); color: var(--primary-foreground); font-family: var(--font-primary)"
        >
          {loading ? "Salvando..." : "Continuar"}
        </button>
      </div>
    {:else if step === 2}
      <div
        class="rounded-2xl p-8"
        style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
      >
        <h1
          class="text-xl font-semibold mb-1"
          style="color: var(--text); font-family: var(--font-primary)"
        >
          Seus serviços
        </h1>
        <p class="text-sm mb-6" style="color: var(--text-secondary)">
          Etapa 2 de 3
        </p>

        {#if error}
          <p
            class="text-sm mb-4 p-3 rounded-lg"
            style="color: #DC2626; background: #FEF2F2"
          >
            {error}
          </p>
        {/if}

        <div class="flex flex-col gap-3">
          {#each services as service, i}
            <div class="flex gap-2 items-start">
              <div class="flex-1 flex gap-2">
                <input
                  bind:value={service.name}
                  placeholder="Nome do serviço"
                  class="flex-1 px-3 py-2.5 rounded-xl text-sm outline-none border"
                  style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
                />
                <div class="relative w-32">
                  <span
                    class="absolute left-3 top-1/2 -translate-y-1/2 text-sm"
                    style="color: var(--text-muted)">R$</span
                  >
                  <input
                    type="number"
                    bind:value={service.price_brl}
                    placeholder="0"
                    min="0"
                    class="w-full pl-9 pr-3 py-2.5 rounded-xl text-sm outline-none border"
                    style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
                  />
                </div>
              </div>
              {#if services.length > 1}
                <button
                  onclick={() => removeService(i)}
                  class="mt-2.5 p-1.5 rounded-lg transition-colors"
                  style="color: var(--text-muted)"
                  aria-label="Remover serviço"
                >
                  <svg
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    width="16"
                    height="16"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clip-rule="evenodd"
                    />
                  </svg>
                </button>
              {/if}
            </div>
          {/each}

          <button
            onclick={addService}
            class="flex items-center gap-1.5 text-sm font-medium mt-1 transition-opacity"
            style="color: var(--primary); font-family: var(--font-primary)"
          >
            <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
              <path
                fill-rule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clip-rule="evenodd"
              />
            </svg>
            Adicionar serviço
          </button>
        </div>

        <button
          onclick={submitStep2}
          disabled={loading}
          class="w-full mt-6 py-3 rounded-xl font-medium text-sm transition-opacity disabled:opacity-60"
          style="background: var(--primary); color: var(--primary-foreground); font-family: var(--font-primary)"
        >
          {loading ? "Salvando..." : "Continuar"}
        </button>
      </div>
    {:else if step === 3}
      <div
        class="rounded-2xl p-8"
        style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
      >
        <h1
          class="text-xl font-semibold mb-1"
          style="color: var(--text); font-family: var(--font-primary)"
        >
          Personalidade do negócio
        </h1>
        <p class="text-sm mb-6" style="color: var(--text-secondary)">
          Etapa 3 de 3 — opcional, você pode pular
        </p>

        {#if error}
          <p
            class="text-sm mb-4 p-3 rounded-lg"
            style="color: #DC2626; background: #FEF2F2"
          >
            {error}
          </p>
        {/if}

        <div class="flex flex-col gap-4">
          <label class="flex flex-col gap-1.5">
            <span class="text-sm font-medium" style="color: var(--text)"
              >Público-alvo</span
            >
            <input
              bind:value={target_audience}
              placeholder="Ex: Mulheres de 25 a 40 anos"
              class="px-3 py-2.5 rounded-xl text-sm outline-none border"
              style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
            />
          </label>

          <label class="flex flex-col gap-1.5">
            <span class="text-sm font-medium" style="color: var(--text)"
              >Estilo da marca</span
            >
            <input
              bind:value={brand_vibe}
              placeholder="Ex: Descontraído e moderno"
              class="px-3 py-2.5 rounded-xl text-sm outline-none border"
              style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
            />
          </label>

          <label class="flex flex-col gap-1.5">
            <span class="text-sm font-medium" style="color: var(--text)">
              Diferenciais
              <span class="font-normal" style="color: var(--text-muted)"
                >— opcional</span
              >
            </span>
            <textarea
              bind:value={quirks}
              placeholder="Ex: Atendimento por WhatsApp, parcela em 3x sem juros"
              rows="3"
              class="px-3 py-2.5 rounded-xl text-sm outline-none border resize-none"
              style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
            ></textarea>
          </label>
        </div>

        <div class="flex gap-3 mt-6">
          <button
            onclick={() => submitStep3(true)}
            disabled={loading}
            class="flex-1 py-3 rounded-xl font-medium text-sm border transition-opacity disabled:opacity-60"
            style="border-color: var(--border-strong); color: var(--text-secondary); font-family: var(--font-primary)"
          >
            Pular
          </button>
          <button
            onclick={() => submitStep3(false)}
            disabled={loading}
            class="flex-1 py-3 rounded-xl font-medium text-sm transition-opacity disabled:opacity-60"
            style="background: var(--primary); color: var(--primary-foreground); font-family: var(--font-primary)"
          >
            {loading ? "Salvando..." : "Concluir"}
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
