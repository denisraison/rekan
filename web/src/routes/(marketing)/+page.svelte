<script lang="ts">
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import { SectionLabel, PhoneFrame, IgPost } from '$lib/components/marketing';

	let commitment: 'mensal' | 'trimestral' = $state('mensal');

	const prices = {
		basico:       { mensal: 69_90, trimestral: 179_70 },
		parceiro:     { mensal: 108_90, trimestral: 299_70 },
		profissional: { mensal: 249_90, trimestral: 599_70 },
	} as const;

	function fmt(cents: number): string {
		return (cents / 100).toFixed(2).replace('.', ',');
	}

	function monthly(cents: number): string {
		return fmt(cents / 3);
	}

	function _reveal(node: HTMLElement) {
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
			postEmoji: '🎂',
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
			postEmoji: '💅',
			caption: 'Francesinha nunca sai de moda. Essa aqui ficou um arraso na @carol.santos ontem. Agenda de quinta tem horário. Chama no whats!',
			hashtags: '#manicure #francesinha #nailart #lunails',
			likes: '56',
		},
	];
</script>

<svelte:head>
	<title>Rekan — Conteúdo pro Instagram do seu negócio</title>
	<meta name="description" content="Legendas, hashtags e stories criados por IA, personalizados pro seu negócio. Feito para micro-empreendedores brasileiros." />
	<meta property="og:title" content="Rekan — Conteúdo pro Instagram do seu negócio" />
	<meta property="og:description" content="Legendas, hashtags e stories criados por IA, personalizados pro seu negócio. Feito para micro-empreendedores brasileiros." />
	<meta property="og:image" content="https://rekan.com.br/og-image.png" />
	<meta property="og:type" content="website" />
	<meta property="og:url" content="https://rekan.com.br" />
</svelte:head>

<!-- Nav -->
<nav class="nav">
	<div class="nav-inner">
		<LogoCombo height={26} />
		<div class="nav-links">
			<a href="#como-funciona">Como funciona</a>
			<a href="#exemplos">Exemplos</a>
			<a href="#preco">Preço</a>
			<!-- TODO: replace phone number -->
			<a href="https://wa.me/5500000000000?text=Oi,%20quero%20saber%20mais%20sobre%20o%20Rekan" target="_blank" rel="noopener" class="btn btn-primary btn-sm">Começar</a>
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
			<SectionLabel pill>PARA MICRO-EMPREENDEDORES</SectionLabel>
			<h1 class="hero-title">
				Post pronto.<br/>Você só <em>cola.</em>
			</h1>
			<p class="hero-sub">
				O Rekan escreve legenda, hashtags e story pro Instagram do seu negócio. Descreve o que você faz uma vez e sai postando.
			</p>
			<p class="hero-tags">Padarias · Salões · Studios · Barbearias · Lojas · Academias</p>
			<div class="hero-cta">
				<!-- TODO: replace phone number -->
				<a href="https://wa.me/5500000000000?text=Oi,%20quero%20saber%20mais%20sobre%20o%20Rekan" target="_blank" rel="noopener" class="btn btn-primary">Comece por R$ 69,90</a>
				<a href="#como-funciona" class="btn btn-ghost">Como funciona</a>
			</div>
		</div>
		<div class="hero-visual">
			<div class="hero-glow" aria-hidden="true"></div>
			<div class="hero-phone">
				<PhoneFrame width="300px">
					<IgPost
						username="padaria.donaelza"
						initial="D"
						avatarBg="#F97368"
						postBg="linear-gradient(145deg, #F97368, #f9a088)"
						postLine1="Bolo do dia"
						postEmoji="🎂"
						caption="Bolo de cenoura com cobertura de chocolate saindo do forno agora. Quem garante o último pedaço? Passa aqui: Rua das Flores, 123"
						hashtags="#padaria #bolodecenoura #donaelza #feitocomamor"
						likes="87"
					/>
				</PhoneFrame>
			</div>
		</div>
	</div>
</section>

<!-- Empathy -->
<section class="empathy" use:_reveal>
	<div class="empathy-inner">
		<p class="empathy-line">Ninguém abriu um negócio pra ficar pensando em legenda.</p>
		<p class="empathy-line">Você faz bolo, corta cabelo, dá aula.</p>
		<p class="empathy-line">O Instagram é só mais uma coisa na lista.</p>
		<p class="empathy-line empathy-question">E se alguém escrevesse pra você?</p>
		<p class="empathy-answer">O Rekan escreve.</p>
	</div>
</section>

<!-- How it works -->
<section class="steps-section" id="como-funciona" use:_reveal>
	<div class="steps-inner">
		<SectionLabel>COMO FUNCIONA</SectionLabel>
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
<section class="showcase" id="exemplos" use:_reveal>
	<div class="showcase-inner">
		<SectionLabel>EXEMPLOS REAIS</SectionLabel>
		<h2 class="showcase-heading">Veja na prática.</h2>
		<p class="showcase-sub">Conteúdo gerado pelo Rekan. Nada genérico, tudo com a cara do negócio.</p>

		<div class="phones">
			{#each examples as ex, i}
				<div class="phone-slot" style="--i: {i}">
					<PhoneFrame>
						<IgPost
							username={ex.username}
							initial={ex.initial}
							avatarBg={ex.avatarBg}
							postBg={ex.postBg}
							postLine1={ex.postLine1}
							postLine2={ex.postLine2}
							postEmoji={ex.postEmoji}
							caption={ex.caption}
							hashtags={ex.hashtags}
							likes={ex.likes}
						/>
					</PhoneFrame>
				</div>
			{/each}
		</div>
	</div>
</section>

<!-- Pricing -->
<section class="pricing" id="preco" use:_reveal>
	<div class="pricing-inner">
		<SectionLabel>PREÇO</SectionLabel>
		<h2 class="pricing-heading">Três planos. Sem pegadinha.</h2>
		<p class="pricing-sub">Um social media manager cobra a partir de R$590/mês.</p>
		<p class="pricing-guarantee">Garantia de 30 dias. Sem contrato.</p>
		<div class="commitment-toggle">
			<button class="toggle-option" class:active={commitment === 'mensal'} onclick={() => { commitment = 'mensal'; }}>Mensal</button>
			<button class="toggle-option" class:active={commitment === 'trimestral'} onclick={() => { commitment = 'trimestral'; }}>
				Trimestral
				<span class="toggle-save">economize até 30%</span>
			</button>
		</div>
		<div class="pricing-grid">
			<div class="tier-card">
				<h3 class="tier-name">Básico</h3>
				<div class="tier-price">
					<span class="tier-currency">R$</span>
					<span class="tier-value">{commitment === 'mensal' ? fmt(prices.basico.mensal) : monthly(prices.basico.trimestral)}</span>
					<span class="tier-period">/mês</span>
				</div>
				{#if commitment === 'trimestral'}
					<p class="tier-total">R$ {fmt(prices.basico.trimestral)} por trimestre</p>
				{/if}
				<ul class="tier-features">
					<li>8 posts por mês</li>
					<li>Legendas + hashtags</li>
				</ul>
				<!-- TODO: replace phone number -->
				<a href="https://wa.me/5500000000000?text=Oi,%20quero%20o%20plano%20Básico%20{commitment}" target="_blank" rel="noopener" class="btn btn-ghost btn-block">Começar</a>
			</div>
			<div class="tier-card tier-card--featured">
				<span class="tier-badge">Mais popular</span>
				<h3 class="tier-name">Parceiro</h3>
				<div class="tier-price">
					<span class="tier-currency">R$</span>
					<span class="tier-value">{commitment === 'mensal' ? fmt(prices.parceiro.mensal) : monthly(prices.parceiro.trimestral)}</span>
					<span class="tier-period">/mês</span>
				</div>
				{#if commitment === 'mensal'}
					<p class="tier-original">de <s>R$149,90</s></p>
				{:else}
					<p class="tier-total">R$ {fmt(prices.parceiro.trimestral)} por trimestre</p>
				{/if}
				<ul class="tier-features">
					<li>12 posts por mês</li>
					<li>Legendas + hashtags</li>
					<li>Direção de foto e roteiro de reels</li>
					<li>Melhor horário pra postar</li>
				</ul>
				<!-- TODO: replace phone number -->
				<a href="https://wa.me/5500000000000?text=Oi,%20quero%20o%20plano%20Parceiro%20{commitment}" target="_blank" rel="noopener" class="btn btn-primary btn-block">Começar agora</a>
				<p class="tier-daily">{commitment === 'mensal' ? 'Menos de R$4 por dia' : 'Menos de R$3,50 por dia'}</p>
			</div>
			<div class="tier-card">
				<h3 class="tier-name">Profissional</h3>
				<div class="tier-price">
					<span class="tier-currency">R$</span>
					<span class="tier-value">{commitment === 'mensal' ? fmt(prices.profissional.mensal) : monthly(prices.profissional.trimestral)}</span>
					<span class="tier-period">/mês</span>
				</div>
				{#if commitment === 'trimestral'}
					<p class="tier-total">R$ {fmt(prices.profissional.trimestral)} por trimestre</p>
				{/if}
				<ul class="tier-features">
					<li>20 posts por mês</li>
					<li>Tudo do Parceiro</li>
					<li>Chamada mensal de estratégia</li>
					<li>Calendário de stories</li>
					<li>Resposta prioritária</li>
				</ul>
				<!-- TODO: replace phone number -->
				<a href="https://wa.me/5500000000000?text=Oi,%20quero%20o%20plano%20Profissional%20{commitment}" target="_blank" rel="noopener" class="btn btn-ghost btn-block">Começar</a>
			</div>
		</div>
	</div>
</section>

<!-- Final CTA -->
<section class="final-cta">
	<div class="final-cta-inner">
		<h2>Bora <em>postar?</em></h2>
		<!-- TODO: replace phone number -->
		<a href="https://wa.me/5500000000000?text=Oi,%20quero%20saber%20mais%20sobre%20o%20Rekan" target="_blank" rel="noopener" class="btn btn-white">Comece agora</a>
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
	.hero-phone :global(.phone-frame) {
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
	.phone-slot:hover :global(.phone-frame) {
		box-shadow: 0 20px 60px rgba(17, 17, 22, 0.16);
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
		margin-bottom: 8px;
		font-size: 0.95rem;
	}
	.pricing-guarantee {
		color: var(--text-muted);
		margin-bottom: 28px;
		font-size: 0.88rem;
	}
	.commitment-toggle {
		display: inline-flex;
		background: white;
		border: 1px solid var(--border);
		border-radius: var(--radius-full);
		padding: 4px;
		margin-bottom: 48px;
	}
	.toggle-option {
		padding: 9px 24px;
		border-radius: var(--radius-full);
		font-size: 0.85rem;
		font-weight: 500;
		font-family: var(--font-primary);
		color: var(--text-secondary);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all 200ms ease;
	}
	.toggle-option.active {
		background: var(--coral);
		color: white;
		box-shadow: 0 2px 8px rgba(249, 115, 104, 0.25);
	}
	.toggle-save {
		font-size: 0.72rem;
		font-weight: 600;
		margin-left: 6px;
		opacity: 0.75;
	}
	.pricing-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 20px;
		max-width: 960px;
		margin: 0 auto;
	}
	.tier-card {
		position: relative;
		background: white;
		border-radius: var(--radius-lg);
		padding: 36px 28px;
		text-align: center;
		border: 1px solid var(--border);
		display: flex;
		flex-direction: column;
	}
	.tier-card--featured {
		border: 2px solid var(--coral);
		box-shadow: var(--shadow-lg);
		padding-top: 44px;
	}
	.tier-badge {
		position: absolute;
		top: -13px;
		left: 50%;
		transform: translateX(-50%);
		background: var(--coral);
		color: white;
		font-size: 0.72rem;
		font-weight: 600;
		padding: 4px 16px;
		border-radius: var(--radius-full);
		letter-spacing: 0.03em;
		white-space: nowrap;
	}
	.tier-name {
		font-size: 1.05rem;
		font-weight: 600;
		margin-bottom: 16px;
		color: var(--text);
	}
	.tier-price {
		display: flex;
		align-items: baseline;
		justify-content: center;
		gap: 3px;
		margin-bottom: 8px;
	}
	.tier-currency {
		font-size: 1rem;
		font-weight: 300;
		color: var(--text-secondary);
		align-self: flex-start;
		margin-top: 10px;
	}
	.tier-value {
		font-size: 3rem;
		font-weight: 300;
		line-height: 1;
		letter-spacing: -0.03em;
		color: var(--text);
	}
	.tier-card--featured .tier-value {
		font-size: 3.5rem;
	}
	.tier-period {
		font-size: 0.85rem;
		font-weight: 400;
		color: var(--text-muted);
	}
	.tier-original,
	.tier-total {
		font-size: 0.82rem;
		color: var(--text-muted);
		margin-bottom: 8px;
	}
	.tier-original s {
		color: var(--text-secondary);
	}
	.tier-features {
		list-style: none;
		text-align: left;
		margin-bottom: 28px;
		flex: 1;
	}
	.tier-features li {
		padding: 10px 0;
		border-bottom: 1px solid var(--border);
		font-size: 0.85rem;
		color: var(--text-secondary);
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.tier-features li::before {
		content: '';
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--sage);
		flex-shrink: 0;
	}
	.tier-card--featured .tier-features li::before {
		background: var(--coral);
	}
	.tier-daily {
		font-size: 0.8rem;
		color: var(--text-muted);
		margin-top: 12px;
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
		.showcase-sub { padding: 0 24px; }
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
		.phone-slot :global(.phone-frame) { width: 256px; }
		.pricing { padding: 72px 24px; }
		.pricing-grid {
			grid-template-columns: 1fr;
			max-width: 380px;
		}
		.tier-card--featured { order: -1; }
		.final-cta { padding: 64px 24px; }
	}
</style>
