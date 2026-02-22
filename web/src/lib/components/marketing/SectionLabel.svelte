<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
		pill?: boolean;
	}

	let { children, pill = false }: Props = $props();
</script>

{#if pill}
	<span class="section-label pill">
		{@render children()}
	</span>
{:else}
	<span class="section-label">
		{@render children()}
	</span>
{/if}

<style>
	.section-label {
		display: block;
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		color: var(--coral);
		margin-bottom: 16px;
		text-align: center;
	}
	.pill {
		display: inline-block;
		padding: 6px 14px;
		background: var(--coral-pale);
		border-radius: var(--radius-full);
		margin-bottom: 20px;
		text-align: left;
	}
</style>
