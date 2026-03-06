<script lang="ts">
	import { goto } from '$app/navigation';
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import { pb } from '$lib/pb';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');

	async function login() {
		if (!email.trim() || !password) {
			error = 'Preencha email e senha.';
			return;
		}
		loading = true;
		error = '';
		try {
			await pb.collection('users').authWithPassword(email.trim(), password);
			goto('/operador');
		} catch {
			error = 'Email ou senha incorretos.';
		} finally {
			loading = false;
		}
	}

</script>

<svelte:head>
	<title>Entrar — Rekan</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center" style="background: var(--bg)">
	<div class="w-full max-w-sm px-6">
		<div class="flex justify-center mb-10">
			<LogoCombo />
		</div>

		<div
			class="rounded-2xl p-8"
			style="background: var(--surface); box-shadow: var(--shadow-md); border: 1px solid var(--border)"
		>
			<h1 class="text-xl font-semibold mb-2" style="color: var(--text); font-family: var(--font-primary)">
				Entrar no Rekan
			</h1>
			<p class="text-sm mb-6" style="color: var(--text-secondary)">
				Acesso restrito para operadores.
			</p>

			{#if error}
				<p class="text-sm mb-4 p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">
					{error}
				</p>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); login(); }} class="flex flex-col gap-4">
				<label class="flex flex-col gap-1.5">
					<span class="text-sm font-medium" style="color: var(--text)">Email</span>
					<input
						bind:value={email}
						type="email"
						autocomplete="email"
						class="px-3 py-2.5 rounded-xl text-sm outline-none border"
						style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
					/>
				</label>

				<label class="flex flex-col gap-1.5">
					<span class="text-sm font-medium" style="color: var(--text)">Senha</span>
					<input
						bind:value={password}
						type="password"
						autocomplete="current-password"
						class="px-3 py-2.5 rounded-xl text-sm outline-none border"
						style="border-color: var(--border-strong); background: var(--surface); color: var(--text); font-family: var(--font-primary)"
					/>
				</label>

				<button
					type="submit"
					disabled={loading}
					class="w-full py-3 rounded-xl font-medium text-sm transition-opacity disabled:opacity-60"
					style="background: var(--primary); color: var(--primary-foreground); font-family: var(--font-primary)"
				>
					{loading ? 'Entrando...' : 'Entrar'}
				</button>
			</form>
		</div>
	</div>
</div>
