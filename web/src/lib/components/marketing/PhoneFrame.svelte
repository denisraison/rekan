<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
		width?: string;
	}

	let { children, width = '268px' }: Props = $props();
</script>

<div class="phone-frame" style:width>
	<div class="phone-notch"></div>
	{@render children()}
</div>

<style>
	.phone-frame {
		background: white;
		border-radius: 36px;
		border: 3px solid var(--dark);
		overflow: hidden;
		box-shadow: var(--shadow-lg);
		transition: box-shadow 300ms ease;
	}
	.phone-notch {
		width: 80px;
		height: 20px;
		background: var(--dark);
		border-radius: 0 0 12px 12px;
		margin: 0 auto;
	}
</style>
