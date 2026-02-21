<script lang="ts">
	import { pb } from '$lib/pb';
	import { goto } from '$app/navigation';
	import LogoCombo from '$lib/components/LogoCombo.svelte';

	let loading = $state(false);
	let error = $state('');

	async function loginWithGoogle() {
		loading = true;
		error = '';
		try {
			await pb.collection('users').authWithOAuth2({ provider: 'google' });
			// Check if user has a business already
			const businesses = await pb.collection('businesses').getList(1, 1);
			if (businesses.totalItems === 0) {
				goto('/onboarding');
			} else {
				goto('/dashboard');
			}
		} catch (e) {
			error = 'Não foi possível entrar. Tente novamente.';
		} finally {
			loading = false;
		}
	}
</script>

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
				Bem-vindo ao Rekan
			</h1>
			<p class="text-sm mb-6" style="color: var(--text-secondary)">
				Entre com sua conta Google para começar.
			</p>

			{#if error}
				<p class="text-sm mb-4 p-3 rounded-lg" style="color: #DC2626; background: #FEF2F2">
					{error}
				</p>
			{/if}

			<button
				onclick={loginWithGoogle}
				disabled={loading}
				class="w-full flex items-center justify-center gap-3 px-4 py-3 rounded-xl font-medium text-sm transition-opacity disabled:opacity-60"
				style="background: var(--primary); color: var(--primary-foreground); font-family: var(--font-primary)"
			>
				{#if loading}
					<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z" />
					</svg>
					Entrando...
				{:else}
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
						<path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
						<path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" />
						<path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
					</svg>
					Entrar com Google
				{/if}
			</button>
		</div>
	</div>
</div>
