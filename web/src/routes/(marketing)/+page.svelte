<script lang="ts">
	import LogoCombo from '$lib/components/LogoCombo.svelte';
	import { SectionLabel } from '$lib/components/marketing';
	import { waLink } from '$lib/whatsapp';

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
			image: '/demo/demo-padaria.png',
			caption: 'Semana passada o forno de baixo apagou no meio da madrugada. 4h da manhã, 3 formas de bolo de cenoura dentro, a resistência simplesmente parou. Dona Elza olhou pra mim e disse: "Tira tudo, passa pro de cima, e reza." Queimei o dedo. O bolo? Saiu perfeito.',
			hashtags: '#bolodecenoura #padariaartesanal #lapasp #bastidores #vidadepadeiro',
		},
		{
			username: 'studio.viva',
			image: '/demo/demo-studio.png',
			caption: 'Semana passada uma aluna chegou aqui segurando a lombar com as duas mãos. Ela tinha 3 hérnias e já tinha desistido de academia, yoga, tudo. Depois de 2 meses no Studio Viva Pilates, ela me mandou esse áudio: "Ju, hoje eu amarrei meu tênis sem sentar." Parece pouco né? Mas quem convive com dor crônica sabe que isso é TUDO.',
			hashtags: '#pilatespinheiros #dorlombar #reabilitação #herniaDeDisco #pilatesreabilitação',
		},
		{
			username: 'lu.nails',
			image: '/demo/demo-nails.png',
			caption: 'Ontem uma cliente sentou na cadeira e disse: "Lu, eu quero aquela francesinha que ninguém faz direito". Eu ouço isso pelo menos 3 vezes por semana. A real é que francesinha parece simples mas é técnica pura. Eu me formei em nail design justamente porque queria dominar esse detalhe. R$45 e sai com acabamento que parece filtro, mas é real.',
			hashtags: '#Francesinha #NailDesigner #EsmalteriaSantoAndré #NailArt #LuNails',
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
		<LogoCombo height={44} />
		<div class="nav-links">
			<a href="#exemplos">Exemplos</a>
			<a href="#preco">Preço</a>
			<a href="{waLink('Oi, quero saber mais sobre o Rekan')}" target="_blank" rel="noopener" class="btn btn-primary btn-sm">Começar</a>
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
				<a href="{waLink('Oi, quero saber mais sobre o Rekan')}" target="_blank" rel="noopener" class="btn btn-primary">Comece por R$ 69,90</a>
				<a href="#exemplos" class="btn btn-ghost">Ver exemplos</a>
			</div>
		</div>
		<div class="hero-visual">
			<div class="wa-window">
				<div class="wa-header">
					<div class="wa-avatar" aria-hidden="true">
						<img src="/brand/logo-mark.svg" alt="" />
					</div>
					<div class="wa-header-text">
						<span class="wa-contact-name">Rekan</span>
						<span class="wa-contact-status">online</span>
					</div>
				</div>
				<div class="wa-body">
					<div class="wa-body-bg" aria-hidden="true"></div>
					<div class="wa-date-chip">hoje</div>
					<div class="wa-bubble wa-bubble--out">
						<p>hoje fiz 60 brigadeiros e entreguei na casa de três clientes 🍫</p>
						<span class="wa-time">14:32 <span class="wa-ticks">✓✓</span></span>
					</div>
					<div class="wa-bubble wa-bubble--in">
						<div class="wa-post-label">POST PRONTO +</div>
						<p>60 brigadeiros saindo da cozinha da Doces da Flavinha hoje 🍫</p>
						<p>Um por um, enrolado na mão, sem conservante nenhum. Separei em três encomendas e fui eu mesma entregar na casa de cada cliente aqui em BH.</p>
						<p>Tem dia que a cozinha fica assim: panela suja, bancada cheia de forminha e eu já pensando na próxima leva. Mas quando a cliente abre a porta e já sente o cheiro do chocolate, vale cada brigadeiro.</p>
						<p>Quer encomendar? Chama no zap, link tá na bio 📱</p>
						<p class="wa-hashtags">#DocesdaFlavinha #BrigadeiroArtesanal #DoceriaEmBH #Savassi #BeloHorizonte #SemConservantes #FeitoEmCasa</p>
						<span class="wa-time">14:33</span>
					</div>
				</div>
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

<!-- Content Showcase -->
<section class="showcase" id="exemplos" use:_reveal>
	<div class="showcase-inner">
		<SectionLabel>EXEMPLOS REAIS</SectionLabel>
		<h2 class="showcase-heading">Conteúdo com a<br/>cara do negócio.</h2>
		<p class="showcase-sub">A padaria mandou uma foto do bolo. O Rekan escreveu. Ela só colou no Instagram.</p>

		<div class="posts">
			{#each examples as ex, i}
				<article class="post-item">
					<p class="post-username">@{ex.username}</p>
					<div class="post-image-wrap">
						<img
							src={ex.image}
							alt="Post gerado pelo Rekan para {ex.username}"
							class="post-image"
							loading="lazy"
						/>
					</div>
					<p class="post-caption">{ex.caption}</p>
					<p class="post-tags">{ex.hashtags}</p>
				</article>
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
				<a href={waLink(`Oi, quero o plano Básico ${commitment}`)} target="_blank" rel="noopener" class="btn btn-ghost btn-block">Começar</a>
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
				<a href={waLink(`Oi, quero o plano Parceiro ${commitment}`)} target="_blank" rel="noopener" class="btn btn-primary btn-block">Começar agora</a>
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
					<li>16 posts por mês</li>
					<li>Tudo do Parceiro</li>
					<li>Chamada mensal de estratégia</li>
					<li>Calendário de stories</li>
					<li>Resposta prioritária</li>
				</ul>
				<a href={waLink(`Oi, quero o plano Profissional ${commitment}`)} target="_blank" rel="noopener" class="btn btn-ghost btn-block">Começar</a>
			</div>
		</div>
	</div>
</section>

<!-- Final CTA -->
<section class="final-cta">
	<div class="final-cta-inner">
		<h2>Bora <em>postar?</em></h2>
		<a href="{waLink('Oi, quero saber mais sobre o Rekan')}" target="_blank" rel="noopener" class="btn btn-white">Comece agora</a>
	</div>
</section>

<!-- Footer -->
<footer class="footer">
	<div class="footer-inner">
		<div class="footer-logo" style="--mark-coral: rgba(255,255,255,0.5); --mark-sage: rgba(255,255,255,0.3); --text: rgba(255,255,255,0.6)">
			<LogoCombo height={24} />
		</div>
		<p>Feito no Brasil, para empreendedores brasileiros.</p>
		<a href="/termos" class="footer-link">Termos de Uso</a>
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

	/* Hero visual: WhatsApp window */
	.hero-visual {
		display: flex;
		align-items: center;
		justify-content: center;
		animation: fadeUp 0.6s ease 0.15s both;
	}
	.hero-visual .wa-window {
		width: 100%;
		box-shadow: 0 32px 80px rgba(17, 17, 22, 0.18), 0 8px 24px rgba(17, 17, 22, 0.1);
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

	/* ============ WHATSAPP WINDOW ============ */
	.wa-window {
		width: 100%;
		max-width: 560px;
		border-radius: 16px;
		overflow: hidden;
		box-shadow: 0 32px 80px rgba(0, 0, 0, 0.5), 0 8px 24px rgba(0, 0, 0, 0.3);
	}
	.wa-header {
		background: #0d4f43;
		padding: 14px 20px;
		display: flex;
		align-items: center;
		gap: 14px;
	}
	.wa-avatar {
		width: 46px;
		height: 46px;
		border-radius: 50%;
		overflow: hidden;
		flex-shrink: 0;
		background: #F5F2ED;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 8px;
	}
	.wa-avatar img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
	.wa-header-text {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.wa-contact-name {
		font-size: 1.05rem;
		font-weight: 600;
		color: white;
		letter-spacing: 0.01em;
	}
	.wa-contact-status {
		font-size: 0.8rem;
		color: rgba(255, 255, 255, 0.6);
	}
	.wa-body {
		background: #ece5dd;
		padding: 16px 12px 20px;
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.wa-body-bg {
		position: absolute;
		inset: 0;
		opacity: 0.04;
		background-image: url("data:image/svg+xml,%3Csvg width='20' height='20' viewBox='0 0 20 20' xmlns='http://www.w3.org/2000/svg'%3E%3Ccircle cx='10' cy='10' r='1' fill='%23000'/%3E%3C/svg%3E");
		background-size: 20px 20px;
		pointer-events: none;
	}
	.wa-date-chip {
		align-self: center;
		background: rgba(255, 255, 255, 0.75);
		color: #555;
		font-size: 0.7rem;
		font-weight: 500;
		padding: 3px 10px;
		border-radius: 6px;
		margin-bottom: 4px;
		position: relative;
	}
	.wa-bubble {
		max-width: 88%;
		padding: 8px 10px 6px;
		border-radius: 8px;
		font-size: 0.82rem;
		line-height: 1.5;
		color: #111;
		position: relative;
	}
	.wa-bubble p {
		margin-bottom: 6px;
		color: #111;
		font-size: 0.82rem;
	}
	.wa-bubble p:last-of-type {
		margin-bottom: 0;
	}
	.wa-bubble--out {
		background: #d9fdd3;
		align-self: flex-end;
		border-bottom-right-radius: 2px;
	}
	.wa-bubble--in {
		background: white;
		align-self: flex-start;
		border-bottom-left-radius: 2px;
	}
	.wa-post-label {
		font-size: 0.65rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--sage-dark, #6B8A61);
		margin-bottom: 8px;
		text-transform: uppercase;
	}
	.wa-hashtags {
		color: #128c7e;
		font-size: 0.76rem;
		margin-top: 6px;
	}
	.wa-time {
		display: block;
		text-align: right;
		font-size: 0.65rem;
		color: #999;
		margin-top: 4px;
	}
	.wa-ticks {
		color: #4fc3f7;
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

	/* Editorial post grid */
	.posts {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 32px;
		align-items: start;
		padding: 8px 0 24px;
	}
	.post-item {
		transition: transform 0.35s ease;
	}
	.post-item:nth-child(1) { transform: rotate(-1.5deg) translateY(12px); }
	.post-item:nth-child(2) { transform: scale(1.03); }
	.post-item:nth-child(3) { transform: rotate(1.5deg) translateY(12px); }
	.post-item:hover { transform: scale(1.02) translateY(-4px); }
	.post-username {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--text-muted);
		margin-bottom: 10px;
		letter-spacing: 0.02em;
	}
	.post-image-wrap {
		border-radius: 12px;
		overflow: hidden;
		box-shadow: 0 8px 32px rgba(17, 17, 22, 0.1), 0 2px 8px rgba(17, 17, 22, 0.06);
		margin-bottom: 16px;
		transition: box-shadow 0.35s ease;
	}
	.post-item:hover .post-image-wrap {
		box-shadow: 0 20px 56px rgba(17, 17, 22, 0.18);
	}
	.post-image {
		width: 100%;
		aspect-ratio: 4 / 5;
		object-fit: cover;
		display: block;
	}
	.post-caption {
		font-size: 0.82rem;
		line-height: 1.6;
		color: var(--text-secondary);
		display: -webkit-box;
		line-clamp: 3;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
		margin-bottom: 8px;
	}
	.post-tags {
		font-size: 0.73rem;
		color: var(--coral);
		font-weight: 500;
		opacity: 0.85;
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
	.footer-link {
		color: rgba(255, 255, 255, 0.28);
		font-size: 0.72rem;
		text-decoration: underline;
		margin-top: 8px;
		display: inline-block;
	}
	.footer-link:hover {
		color: rgba(255, 255, 255, 0.45);
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
	@media (prefers-reduced-motion: reduce) {
		.hero-mesh-1, .hero-mesh-2 { animation: none; }
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
		.wa-window { border-radius: 12px; }
		.showcase { padding: 72px 0; }
		.showcase-heading,
		.showcase-sub { padding: 0 24px; }
		.posts {
			grid-template-columns: 1fr;
			gap: 40px;
			padding: 8px 24px 24px;
			max-width: 380px;
			margin: 0 auto;
		}
		.post-item:nth-child(1),
		.post-item:nth-child(2),
		.post-item:nth-child(3) { transform: none; }
		.pricing { padding: 72px 24px; }
		.pricing-grid {
			grid-template-columns: 1fr;
			max-width: 380px;
		}
		.tier-card--featured { order: -1; }
		.final-cta { padding: 64px 24px; }
	}
</style>
