<script lang="ts">
	import LogoMark from './LogoMark.svelte';

	let { height = 28 }: { height?: number } = $props();
</script>

<a href="/" class="logo-combo">
	<LogoMark size={height} />
	<span class="logo-text" style:font-size="{Math.round(height * 0.64)}px">rekan</span>
</a>

<style>
	.logo-combo {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		text-decoration: none;
	}
	.logo-combo:hover { text-decoration: none; }
	.logo-text {
		font-family: 'Urbanist', sans-serif;
		font-weight: 300;
		letter-spacing: 0.05em;
		color: var(--text, #111116);
		line-height: 1;
	}
</style>
