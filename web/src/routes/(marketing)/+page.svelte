<script lang="ts">
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import LogoMark from '$lib/components/LogoMark.svelte';

	function reveal(node: HTMLElement) {
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						node.classList.add('visible');
						observer.unobserve(node);
					}
				}
			},
			{ threshold: 0.05 }
		);
		observer.observe(node);
		return { destroy() { observer.disconnect(); } };
	}

	const examples = [
		{
			username: 'padaria.donaelza',
			initial: 'D',
			avatarBg: '#F97368',
			postBg: 'linear-gradient(145deg, #F97368, #f9a088)',
			postLine1: 'Bolo do dia',
			postLine2: '',
			postEmoji: '\ud83c\udf82',
			caption: 'Bolo de cenoura com cobertura de chocolate saindo do forno agora. Quem garante o último pedaço? Passa aqui: Rua das Flores, 123',
			hashtags: '#padaria #bolodecenoura #donaelza #feitocomamor',
			likes: '87',
		},
		{
			username: 'studio.viva',
			initial: 'V',
			avatarBg: '#87AA8C',
			postBg: 'linear-gradient(145deg, #87AA8C, #a8c49b)',
			postLine1: '30 min mudam',
			postLine2: 'seu dia',
			postEmoji: '',
			caption: 'Você não precisa de uma hora. 30 minutos de pilates já mudam seu dia: menos dor, mais disposição. Aula experimental grátis essa semana.',
			hashtags: '#pilates #bemestar #aulagratis #vidasaudavel',
			likes: '134',
		},
		{
			username: 'lu.nails',
			initial: 'L',
			avatarBg: '#d4827f',
			postBg: 'linear-gradient(145deg, #d4827f, #F97368)',
			postLine1: 'Agenda aberta',
			postLine2: 'essa semana',
			postEmoji: '\ud83d\udc85',
			caption: 'Francesinha nunca sai de moda. Essa aqui ficou um arraso na @carol.santos ontem. Agenda de quinta tem horário. Chama no whats!',
			hashtags: '#manicure #francesinha #nailart #lunails',
			likes: '56',
		},
	];
</script>

<svelte:head>
	<title>Rekan — Conteúdo pro Instagram do seu negócio</title>
	<meta name="description" content="Legendas, hashtags e stories criados por IA, personalizados pro seu negócio. Feito para micro-empreendedores brasileiros." />
</svelte:head>

<!-- Nav -->
<nav class="nav">
	<div class="nav-inner">
		<LogoCombo height={26} />
		<div class="nav-links">
			<a href="#como-funciona">Como funciona</a>
			<a href="#exemplos">Exemplos</a>
			<a href="#preco">Preço</a>
			<a href="/entrar" class="btn btn-primary btn-sm">Começar</a>
		</div>
	</div>
</nav>

<!-- Hero -->
<section class="hero">
	<div class="hero-grid" aria-hidden="true"></div>
	<div class="hero-mesh hero-mesh-1" aria-hidden="true"></div>
	<div class="hero-mesh hero-mesh-2" aria-hidden="true"></div>
	<div class="hero-content">
		<div class="hero-text">
			<span class="hero-label">PARA MICRO-EMPREENDEDORES</span>
			<h1 class="hero-title">
				Post pronto.<br/>Você só <em>cola.</em>
			</h1>
			<p class="hero-sub">
				O Rekan escreve legenda, hashtags e story pro Instagram do seu negócio. Descreve o que você faz uma vez e sai postando.
			</p>
			<p class="hero-tags">Padarias · Salões · Studios · Barbearias · Lojas · Academias</p>
			<div class="hero-cta">
				<a href="/entrar" class="btn btn-primary">Experimente grátis</a>
				<a href="#como-funciona" class="btn btn-ghost">Como funciona</a>
			</div>
		</div>
		<div class="hero-visual">
			<div class="hero-glow" aria-hidden="true"></div>
			<div class="hero-phone">
				<div class="phone-frame">
					<div class="phone-notch"></div>
					<div class="ig-head">
						<div class="ig-avatar" style="background: #F97368">D</div>
						<span class="ig-name">padaria.donaelza</span>
						<span class="ig-dots">···</span>
					</div>
					<div class="ig-post" style="background: linear-gradient(145deg, #F97368, #f9a088)">
						<LogoMark size={28} />
						<span class="ig-post-title">Bolo do dia</span>
						<span class="ig-post-emoji">🎂</span>
					</div>
					<div class="ig-actions">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
						<span class="ig-spacer"></span>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
					</div>
					<div class="ig-body">
						<p class="ig-likes">87 curtidas</p>
						<p class="ig-caption"><strong>padaria.donaelza</strong> Bolo de cenoura com cobertura de chocolate saindo do forno agora. Quem garante o último pedaço? Passa aqui: Rua das Flores, 123</p>
						<p class="ig-hashtags">#padaria #bolodecenoura #donaelza #feitocomamor</p>
					</div>
				</div>
			</div>
		</div>
	</div>
</section>

<!-- Empathy -->
<section class="empathy" use:reveal>
	<div class="empathy-inner">
		<p class="empathy-line">Ninguém abriu um negócio pra ficar pensando em legenda.</p>
		<p class="empathy-line">Você faz bolo, corta cabelo, dá aula.</p>
		<p class="empathy-line">O Instagram é só mais uma coisa na lista.</p>
		<p class="empathy-line empathy-question">E se alguém escrevesse pra você?</p>
		<p class="empathy-answer">O Rekan escreve.</p>
	</div>
</section>

<!-- How it works -->
<section class="steps-section" id="como-funciona" use:reveal>
	<div class="steps-inner">
		<span class="section-label">COMO FUNCIONA</span>
		<h2 class="steps-heading">Três passos.<br/>Dois minutos.</h2>
		<div class="steps">
			<div class="step-card">
				<div class="step-accent"></div>
				<span class="step-num">01</span>
				<h3>Conta sobre seu negócio</h3>
				<p>O que você faz, quem compra, como você fala. Preenche uma vez e pronto.</p>
			</div>
			<div class="step-card">
				<div class="step-accent"></div>
				<span class="step-num">02</span>
				<h3>Escolhe o tipo de post</h3>
				<p>Promoção, dica do dia, novidade, engajamento. Você decide o objetivo, a IA escreve.</p>
			</div>
			<div class="step-card">
				<div class="step-accent"></div>
				<span class="step-num">03</span>
				<h3>Cola no Instagram</h3>
				<p>Legenda, hashtags e story, tudo junto. Copia e publica.</p>
			</div>
		</div>
	</div>
</section>

<!-- Content Showcase -->
<section class="showcase" id="exemplos" use:reveal>
	<div class="showcase-inner">
		<span class="section-label">EXEMPLOS REAIS</span>
		<h2 class="showcase-heading">Veja na prática.</h2>
		<p class="showcase-sub">Conteúdo gerado pelo Rekan. Nada genérico, tudo com a cara do negócio.</p>

		<div class="phones">
			{#each examples as ex, i}
				<div class="phone-slot" style="--i: {i}">
					<div class="phone-frame">
						<div class="phone-notch"></div>
						<div class="ig-head">
							<div class="ig-avatar" style="background: {ex.avatarBg}">{ex.initial}</div>
							<span class="ig-name">{ex.username}</span>
							<span class="ig-dots">···</span>
						</div>
						<div class="ig-post" style="background: {ex.postBg}">
							<LogoMark size={28} />
							<span class="ig-post-title">{ex.postLine1}{#if ex.postLine2}<br/>{ex.postLine2}{/if}</span>
							{#if ex.postEmoji}<span class="ig-post-emoji">{ex.postEmoji}</span>{/if}
						</div>
						<div class="ig-actions">
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
							<span class="ig-spacer"></span>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
						</div>
						<div class="ig-body">
							<p class="ig-likes">{ex.likes} curtidas</p>
							<p class="ig-caption"><strong>{ex.username}</strong> {ex.caption}</p>
							<p class="ig-hashtags">{ex.hashtags}</p>
						</div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</section>

<!-- Pricing -->
<section class="pricing" id="preco" use:reveal>
	<div class="pricing-inner">
		<span class="section-label">PREÇO</span>
		<h2 class="pricing-heading">Um plano. Sem pegadinha.</h2>
		<p class="pricing-sub">Um preço. Sem letras miúdas.</p>
		<div class="price-card">
			<div class="price-amount">
				<span class="price-currency">R$</span>
				<span class="price-value">29</span>
				<span class="price-period">/mês</span>
			</div>
			<ul class="price-list">
				<li>30 legendas por mês</li>
				<li>10 textos pra stories</li>
				<li>Tom personalizado pro seu negócio</li>
				<li>Sem contrato, cancele quando quiser</li>
			</ul>
			<a href="/entrar" class="btn btn-primary btn-block">Começar agora</a>
		</div>
	</div>
</section>

<!-- Final CTA -->
<section class="final-cta">
	<div class="final-cta-inner">
		<h2>Bora <em>postar?</em></h2>
		<a href="/entrar" class="btn btn-white">Comece grátis</a>
	</div>
</section>

<!-- Footer -->
<footer class="footer">
	<div class="footer-inner">
		<div class="footer-logo" style="--mark-coral: rgba(255,255,255,0.5); --mark-sage: rgba(255,255,255,0.3); --text: rgba(255,255,255,0.6)">
			<LogoCombo height={24} />
		</div>
		<p>Feito no Brasil, para empreendedores brasileiros.</p>
	</div>
</footer>

<style>
	/* ============ NAV ============ */
	.nav {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 100;
		background: rgba(250, 250, 247, 0.8);
		backdrop-filter: blur(24px);
		-webkit-backdrop-filter: blur(24px);
		border-bottom: 1px solid var(--border);
	}
	.nav-inner {
		max-width: 1120px;
		margin: 0 auto;
		padding: 14px 24px;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.nav-links {
		display: flex;
		align-items: center;
		gap: 32px;
	}
	.nav-links a {
		font-size: 0.85rem;
		font-weight: 500;
		color: var(--text-secondary);
		letter-spacing: 0.01em;
		transition: color 150ms;
	}
	.nav-links a:hover {
		color: var(--text);
		text-decoration: none;
	}

	/* ============ BUTTONS ============ */
	.btn {
		display: inline-block;
		font-family: var(--font-primary);
		font-weight: 500;
		font-size: 0.9rem;
		padding: 13px 30px;
		border-radius: var(--radius-full);
		transition: all 200ms ease;
		cursor: pointer;
		text-decoration: none;
		border: none;
		letter-spacing: 0.01em;
	}
	.btn:hover { text-decoration: none; }
	.btn-primary {
		background: var(--coral);
		color: white;
		box-shadow: 0 4px 16px rgba(249, 115, 104, 0.3);
	}
	.btn-primary:hover {
		background: var(--coral-dark);
		transform: translateY(-2px);
		box-shadow: 0 8px 28px rgba(249, 115, 104, 0.35);
	}
	.btn-ghost {
		background: transparent;
		color: var(--text);
		border: 1.5px solid var(--border-strong);
	}
	.btn-ghost:hover {
		border-color: var(--text);
		transform: translateY(-2px);
	}
	.btn-white {
		background: white;
		color: var(--coral);
		font-weight: 600;
	}
	.btn-white:hover {
		transform: translateY(-2px);
		box-shadow: 0 8px 28px rgba(0, 0, 0, 0.12);
	}
	.btn-sm { padding: 8px 20px; font-size: 0.8rem; }
	.btn-block { display: block; width: 100%; text-align: center; }

	/* ============ SECTION LABEL ============ */
	.section-label {
		display: block;
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		color: var(--coral);
		margin-bottom: 16px;
		text-align: center;
	}

	/* ============ HERO ============ */
	.hero {
		position: relative;
		min-height: 100vh;
		min-height: 100dvh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 120px 24px 80px;
		overflow: hidden;
		background: var(--bg);
	}
	.hero-grid {
		position: absolute;
		inset: 0;
		background-image: radial-gradient(circle, rgba(17, 17, 22, 0.06) 1px, transparent 1px);
		background-size: 28px 28px;
		pointer-events: none;
	}
	.hero-mesh {
		position: absolute;
		border-radius: 50%;
		pointer-events: none;
	}
	.hero-mesh-1 {
		width: 800px;
		height: 800px;
		background: radial-gradient(circle, rgba(249, 115, 104, 0.22) 0%, rgba(249, 115, 104, 0.08) 35%, transparent 60%);
		top: -20%;
		right: -10%;
		animation: meshFloat 28s ease-in-out infinite;
	}
	.hero-mesh-2 {
		width: 650px;
		height: 650px;
		background: radial-gradient(circle, rgba(135, 170, 140, 0.18) 0%, rgba(135, 170, 140, 0.06) 35%, transparent 60%);
		bottom: -15%;
		left: -10%;
		animation: meshFloat 34s ease-in-out infinite reverse;
	}
	.hero-content {
		position: relative;
		z-index: 1;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 48px;
		align-items: center;
		max-width: 1120px;
		width: 100%;
	}
	.hero-text {
		animation: fadeUp 0.6s ease both;
	}
	.hero-label {
		display: inline-block;
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		color: var(--coral);
		margin-bottom: 20px;
		padding: 6px 14px;
		background: var(--coral-pale);
		border-radius: var(--radius-full);
	}
	.hero-title {
		font-size: clamp(2.6rem, 5.5vw, 4.2rem);
		font-weight: 300;
		line-height: 1.06;
		letter-spacing: -0.025em;
		margin-bottom: 24px;
	}
	.hero-title em {
		font-style: italic;
		color: var(--coral);
	}
	.hero-sub {
		font-size: 1.05rem;
		font-weight: 400;
		line-height: 1.65;
		color: var(--text-secondary);
		max-width: 440px;
		margin-bottom: 16px;
	}
	.hero-tags {
		font-size: 0.78rem;
		font-weight: 500;
		letter-spacing: 0.04em;
		color: var(--text-muted);
		margin-bottom: 32px;
	}
	.hero-cta {
		display: flex;
		gap: 12px;
	}

	/* Hero visual column */
	.hero-visual {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 420px;
		animation: fadeUp 0.6s ease 0.15s both;
	}
	.hero-glow {
		position: absolute;
		width: 420px;
		height: 420px;
		background: radial-gradient(circle,
			rgba(249, 115, 104, 0.25) 0%,
			rgba(253, 220, 218, 0.18) 30%,
			rgba(135, 170, 140, 0.1) 55%,
			transparent 70%
		);
		border-radius: 50%;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		filter: blur(40px);
	}
	.hero-phone {
		position: relative;
		z-index: 1;
		transform: rotate(2deg);
	}
	.hero-phone .phone-frame {
		width: 300px;
		box-shadow: 0 24px 64px rgba(17, 17, 22, 0.14), 0 4px 12px rgba(17, 17, 22, 0.06);
	}

	/* ============ EMPATHY ============ */
	.empathy {
		background: var(--dark);
		padding: 88px 24px 96px;
		text-align: center;
		position: relative;
	}
	.empathy::before {
		content: '';
		position: absolute;
		inset: 0;
		background-image: radial-gradient(circle, rgba(255, 255, 255, 0.03) 1px, transparent 1px);
		background-size: 24px 24px;
		pointer-events: none;
	}
	.empathy-inner {
		max-width: 560px;
		margin: 0 auto;
		position: relative;
	}
	.empathy-line,
	.empathy-answer {
		font-family: var(--font-accent);
		color: rgba(255, 255, 255, 0.7);
		font-size: clamp(1.2rem, 2.5vw, 1.55rem);
		font-weight: 300;
		line-height: 1.85;
		opacity: 0.15;
		transform: translateY(8px);
		transition: opacity 0.5s ease, transform 0.5s ease;
	}
	.empathy:global(.visible) .empathy-line,
	.empathy:global(.visible) .empathy-answer {
		opacity: 1;
		transform: none;
	}
	.empathy:global(.visible) .empathy-line:nth-child(1) { transition-delay: 0s; }
	.empathy:global(.visible) .empathy-line:nth-child(2) { transition-delay: 0.15s; }
	.empathy:global(.visible) .empathy-line:nth-child(3) { transition-delay: 0.3s; }
	.empathy:global(.visible) .empathy-question { transition-delay: 0.45s; }
	.empathy:global(.visible) .empathy-answer { transition-delay: 0.75s; }
	.empathy-question {
		font-style: italic;
		color: rgba(255, 255, 255, 0.4);
	}
	.empathy-answer {
		margin-top: 36px;
		font-size: clamp(1.5rem, 3vw, 1.9rem);
		font-weight: 400;
		color: var(--coral-light);
	}

	/* ============ STEPS ============ */
	.steps-section {
		padding: 100px 24px;
		background: var(--sage-pale);
		transform: translateY(16px);
		transition: transform 0.6s ease;
	}
	.steps-section:global(.visible) {
		transform: none;
	}
	.steps-inner {
		max-width: 1000px;
		margin: 0 auto;
	}
	.steps-heading {
		font-size: clamp(1.8rem, 4vw, 2.8rem);
		font-weight: 300;
		text-align: center;
		margin-bottom: 60px;
		line-height: 1.15;
		letter-spacing: -0.02em;
	}
	.steps {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 20px;
	}
	.step-card {
		position: relative;
		background: white;
		border: 1px solid rgba(135, 170, 140, 0.2);
		border-radius: var(--radius-lg);
		padding: 36px 28px;
		overflow: hidden;
		transition: transform 300ms ease, box-shadow 300ms ease;
	}
	.step-card:hover {
		transform: translateY(-4px);
		box-shadow: var(--shadow-lg);
	}
	.step-accent {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 3px;
	}
	.step-card:nth-child(1) .step-accent { background: var(--coral); }
	.step-card:nth-child(2) .step-accent { background: var(--sage); }
	.step-card:nth-child(3) .step-accent { background: var(--coral-light); }
	.step-num {
		font-size: 3rem;
		font-weight: 200;
		line-height: 1;
		display: block;
		margin-bottom: 20px;
	}
	.step-card:nth-child(1) .step-num { color: var(--coral); }
	.step-card:nth-child(2) .step-num { color: var(--sage); }
	.step-card:nth-child(3) .step-num { color: var(--coral-light); }
	.step-card h3 {
		font-size: 1.05rem;
		font-weight: 600;
		margin-bottom: 8px;
		color: var(--text);
	}
	.step-card p {
		font-size: 0.9rem;
		color: var(--text-secondary);
		line-height: 1.6;
	}

	/* ============ SHOWCASE ============ */
	.showcase {
		padding: 100px 24px;
		background: white;
		transform: translateY(16px);
		transition: transform 0.6s ease;
	}
	.showcase:global(.visible) {
		transform: none;
	}
	.showcase-inner {
		max-width: 1120px;
		margin: 0 auto;
	}
	.showcase-heading {
		font-size: clamp(1.8rem, 4vw, 2.8rem);
		font-weight: 300;
		text-align: center;
		margin-bottom: 12px;
		letter-spacing: -0.02em;
	}
	.showcase-sub {
		text-align: center;
		color: var(--text-secondary);
		max-width: 440px;
		margin: 0 auto 56px;
		font-size: 0.95rem;
	}
	.phones {
		display: flex;
		justify-content: center;
		align-items: flex-start;
		padding: 20px 0;
	}
	.phone-slot {
		transition: transform 0.4s ease;
	}
	.phone-slot:nth-child(1) {
		transform: rotate(-5deg) translateY(32px);
		z-index: 1;
		margin-right: -16px;
	}
	.phone-slot:nth-child(2) {
		z-index: 3;
		transform: scale(1.04);
	}
	.phone-slot:nth-child(3) {
		transform: rotate(5deg) translateY(32px);
		z-index: 1;
		margin-left: -16px;
	}
	.phone-frame {
		width: 268px;
		background: white;
		border-radius: 36px;
		border: 3px solid var(--dark);
		overflow: hidden;
		box-shadow: var(--shadow-lg);
		transition: box-shadow 300ms ease;
	}
	.phone-slot:hover .phone-frame {
		box-shadow: 0 20px 60px rgba(17, 17, 22, 0.16);
	}
	.phone-notch {
		width: 80px;
		height: 20px;
		background: var(--dark);
		border-radius: 0 0 12px 12px;
		margin: 0 auto;
	}
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

	/* ============ PRICING ============ */
	.pricing {
		padding: 100px 24px;
		background: var(--coral-pale);
		transform: translateY(16px);
		transition: transform 0.6s ease;
	}
	.pricing:global(.visible) {
		transform: none;
	}
	.pricing-inner {
		max-width: 1120px;
		margin: 0 auto;
		text-align: center;
	}
	.pricing-heading {
		font-size: clamp(1.8rem, 4vw, 2.8rem);
		font-weight: 300;
		margin-bottom: 8px;
		letter-spacing: -0.02em;
	}
	.pricing-sub {
		color: var(--text-secondary);
		margin-bottom: 48px;
		font-size: 0.95rem;
	}
	.price-card {
		position: relative;
		max-width: 380px;
		margin: 0 auto;
		background: white;
		border-radius: var(--radius-lg);
		padding: 48px 36px;
		text-align: center;
		box-shadow: var(--shadow-lg);
	}
	.price-amount {
		display: flex;
		align-items: baseline;
		justify-content: center;
		gap: 4px;
		margin-bottom: 36px;
	}
	.price-currency {
		font-size: 1.2rem;
		font-weight: 300;
		color: var(--text-secondary);
		align-self: flex-start;
		margin-top: 14px;
	}
	.price-value {
		font-size: 4.5rem;
		font-weight: 300;
		line-height: 1;
		letter-spacing: -0.04em;
		color: var(--text);
	}
	.price-period {
		font-size: 0.9rem;
		font-weight: 400;
		color: var(--text-muted);
	}
	.price-list {
		list-style: none;
		text-align: left;
		margin-bottom: 36px;
	}
	.price-list li {
		padding: 12px 0;
		border-bottom: 1px solid var(--border);
		font-size: 0.9rem;
		color: var(--text-secondary);
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.price-list li::before {
		content: '';
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--sage);
		flex-shrink: 0;
	}

	/* ============ FINAL CTA ============ */
	.final-cta {
		background: linear-gradient(145deg, var(--coral), var(--coral-dark));
		padding: 96px 24px;
		text-align: center;
		position: relative;
		overflow: hidden;
	}
	.final-cta-inner {
		max-width: 600px;
		margin: 0 auto;
		position: relative;
	}
	.final-cta h2 {
		font-size: clamp(2rem, 5vw, 3.2rem);
		font-weight: 300;
		color: white;
		margin-bottom: 28px;
		letter-spacing: -0.02em;
	}
	.final-cta h2 em {
		font-style: italic;
	}

	/* ============ FOOTER ============ */
	.footer {
		background: var(--dark);
		padding: 44px 24px;
		text-align: center;
	}
	.footer-inner {
		max-width: 1120px;
		margin: 0 auto;
	}
	.footer-logo {
		display: flex;
		justify-content: center;
		margin-bottom: 14px;
	}
	.footer-inner p {
		color: rgba(255, 255, 255, 0.28);
		font-size: 0.78rem;
	}

	/* ============ ANIMATIONS ============ */
	@keyframes fadeUp {
		from { opacity: 0; transform: translateY(20px); }
		to { opacity: 1; transform: translateY(0); }
	}
	@keyframes meshFloat {
		0%, 100% { transform: translate(0, 0); }
		33% { transform: translate(30px, -20px); }
		66% { transform: translate(-20px, 15px); }
	}
	@keyframes floatCard {
		0%, 100% { transform: translateY(0) rotate(var(--r, 2deg)); }
		50% { transform: translateY(-10px) rotate(var(--r, 2deg)); }
	}

	/* ============ RESPONSIVE ============ */
	@media (max-width: 768px) {
		.nav-links a:not(.btn) { display: none; }
		.hero { min-height: auto; padding: 110px 24px 56px; }
		.hero-content {
			grid-template-columns: 1fr;
			text-align: center;
		}
		.hero-visual { display: none; }
		.hero-sub { margin-left: auto; margin-right: auto; }
		.hero-cta { justify-content: center; flex-wrap: wrap; }
		.empathy { padding: 56px 24px 64px; }
		.steps-section { padding: 72px 24px; }
		.steps-heading { margin-bottom: 40px; }
		.steps {
			grid-template-columns: 1fr;
			gap: 16px;
		}
		.showcase { padding: 72px 0; }
		.showcase-heading,
		.showcase-sub,
		.section-label { padding: 0 24px; }
		.phones {
			justify-content: flex-start;
			overflow-x: auto;
			scroll-snap-type: x mandatory;
			gap: 16px;
			padding: 20px 24px;
			-webkit-overflow-scrolling: touch;
		}
		.phone-slot {
			flex: 0 0 256px;
			scroll-snap-align: center;
		}
		.phone-slot:nth-child(1),
		.phone-slot:nth-child(2),
		.phone-slot:nth-child(3) {
			transform: none;
			margin: 0;
		}
		.phone-frame { width: 256px; }
		.pricing { padding: 72px 24px; }
		.price-card { padding: 36px 24px; }
		.final-cta { padding: 64px 24px; }
	}
</style>
