<script lang="ts">
	import LogoMark from '$lib/components/LogoMark.svelte';

	interface Props {
		username: string;
		initial: string;
		avatarBg: string;
		postBg: string;
		postLine1: string;
		postLine2?: string;
		postEmoji?: string;
		caption: string;
		hashtags: string;
		likes: string;
	}

	let {
		username,
		initial,
		avatarBg,
		postBg,
		postLine1,
		postLine2,
		postEmoji,
		caption,
		hashtags,
		likes,
	}: Props = $props();
</script>

<div class="ig-head">
	<div class="ig-avatar" style="background: {avatarBg}">{initial}</div>
	<span class="ig-name">{username}</span>
	<span class="ig-dots">···</span>
</div>
<div class="ig-post" style="background: {postBg}">
	<LogoMark size={28} />
	<span class="ig-post-title">{postLine1}{#if postLine2}<br/>{postLine2}{/if}</span>
	{#if postEmoji}<span class="ig-post-emoji">{postEmoji}</span>{/if}
</div>
<div class="ig-actions">
	<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
	<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
	<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
	<span class="ig-spacer"></span>
	<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
</div>
<div class="ig-body">
	<p class="ig-likes">{likes} curtidas</p>
	<p class="ig-caption"><strong>{username}</strong> {caption}</p>
	<p class="ig-hashtags">{hashtags}</p>
</div>

<style>
	.ig-head {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 10px 12px 8px;
	}
	.ig-avatar {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-size: 10px;
		font-weight: 700;
		flex-shrink: 0;
	}
	.ig-name {
		font-size: 11px;
		font-weight: 700;
		color: var(--text);
	}
	.ig-dots {
		margin-left: auto;
		font-weight: 700;
		color: var(--text);
		font-size: 14px;
		letter-spacing: 2px;
	}
	.ig-post {
		width: 100%;
		aspect-ratio: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 24px;
		position: relative;
	}
	.ig-post :global(.logo-mark) {
		opacity: 0.2;
		margin-bottom: 12px;
	}
	.ig-post-title {
		font-family: var(--font-primary);
		font-size: 1.3rem;
		font-weight: 300;
		color: white;
		text-align: center;
		line-height: 1.25;
		text-shadow: 0 1px 8px rgba(0, 0, 0, 0.1);
	}
	.ig-post-emoji {
		font-size: 1.6rem;
		margin-top: 10px;
	}
	.ig-actions {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 12px 4px;
		color: var(--text);
	}
	.ig-actions svg {
		width: 18px;
		height: 18px;
	}
	.ig-spacer { flex: 1; }
	.ig-body {
		padding: 0 12px 14px;
	}
	.ig-likes {
		font-size: 10px;
		font-weight: 700;
		margin-bottom: 4px;
	}
	.ig-caption {
		font-size: 10px;
		line-height: 1.5;
		color: var(--text);
		margin-bottom: 4px;
		display: -webkit-box;
		line-clamp: 4;
		-webkit-line-clamp: 4;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.ig-caption strong { font-weight: 700; }
	.ig-hashtags {
		font-size: 9px;
		color: #00376b;
	}
</style>
