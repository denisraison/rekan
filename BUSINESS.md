# Rekan Business Plan

**WhatsApp Content Partner for Brazilian Micro Entrepreneurs**

*February 2026*

---

## What Is Rekan

Rekan is an AI powered Instagram content service delivered entirely through WhatsApp. A micro entrepreneur (MEI) sends a quick message about their day ("fiz um bolo de casamento lindo, rosa e dourado") and gets back ready to post Instagram content within minutes: caption, hashtags, visual direction, reel ideas.

The name means "partner" in Indonesian. That's what we are. Not an app. Not a tool. A content partner that lives in the channel every MEI already uses 14 hours a day.

---

## Why This Business Exists

Elenice (founder) has a strong sales background and is excellent with people but has struggled to find stable employment in Brazil. Denis (technical co founder, staff engineer in Australia) built the AI content pipeline. Instead of ongoing financial support, the goal is to build a sustainable business that gives Elenice purpose, income, and independence.

**Target income:** R$2,000+/month net for Elenice.

---

## The Market

Brazil has 15.6 million active MEIs (Microempreendedor Individual) as of early 2025, representing over 73% of all formal businesses. The most common categories are hairdressers, clothing retailers, food vendors, and service providers. These are solo operators who know Instagram matters for their business but don't have time or skills to manage it.

Brazil has 160 million active social media users spending an average of 3 hours 42 minutes daily on platforms. Direct purchasing through Instagram and TikTok is growing at 127% annually in Brazil, outpacing global averages.

The opportunity is enormous: millions of people who need Instagram content but can't afford a social media manager and don't have time to learn another app.

---

## The Pricing Gap

There's a massive gap in the Brazilian market that nobody is serving well:

| Tier | Price Range | What You Get |
|---|---|---|
| DIY AI tools (GalilAI, Canva, ChatGPT) | R$0 to R$99/month | Self serve. You still do the work. |
| **Rekan sits here** | **R$69 to R$99/month** | **Done for you via WhatsApp. Zero friction.** |
| Junior social media manager | R$590 to R$1,500/month | Human creates and posts content for you. |
| Agency | R$2,000 to R$5,000+/month | Full service marketing. |

A MEI making R$6,750/month (the average) can't justify R$590 for a social media manager. But they also don't have the time or patience to open GalilAI, navigate the interface, generate content, edit it, and schedule it. That's still too much friction for someone cutting hair or baking cakes 10 hours a day.

Rekan fills the gap: affordable enough for any MEI, zero friction because it's just WhatsApp, and human relationship included.

---

## How We Got Here: Decisions Made

### Decision 1: Not a SaaS App (for now)

The original idea was a self serve app where MEIs would log in, generate posts, and schedule them. We decided against this as the primary model because:

- **GalilAI already exists** in this space, charges R$28 to R$99/month, is funded, and has a full engineering team in Florianópolis. Competing head to head with a SaaS requires marketing spend, customer support infrastructure, and constant feature development.
- **Elenice can't run a SaaS.** Who handles support tickets? Who writes landing page copy? Who runs acquisition ads? Denis has a full time job.
- **MEIs don't want another app.** They want someone to do it for them. The Brazilian market values personal relationships and "done for you" convenience enormously.

### Decision 2: Not a Pure Service Business Either

We considered a traditional service model where Elenice acts as a social media manager, using the AI pipeline behind the scenes to be more productive. This works but has limitations:

- It's essentially a job, not a scalable business.
- Elenice's time is the bottleneck. More clients = more hours.
- If she takes a week off, everything stops.

This model (Option 1) remains the fallback if the WhatsApp approach doesn't work. Elenice could charge R$149 to R$499/month per client with tiered packages and hit R$2,000/month with 10 to 15 clients.

### Decision 3: WhatsApp First Hybrid (what we're building)

The chosen model combines the best of both: the scalability of a product with the human touch of a service.

**Why WhatsApp:**
- Every MEI in Brazil is on WhatsApp all day. Zero adoption friction.
- No app to download, no login, no interface to learn.
- The interaction is natural: "tell me what you did today" → get back content.
- As automation increases, adding client number 50 doesn't require proportionally more of Elenice's time.

**Why this is different from everything else:**
- GalilAI requires 5 to 10 minutes of friction per session (open app, navigate, generate, edit, schedule).
- Rekan requires 30 seconds (send a WhatsApp message, get content back, copy paste and post).
- Nobody else is doing Instagram content delivery through WhatsApp with a human relationship layer.
- The "content direction" angle (what photo to take, how to frame it, reel scripts) is something no tool offers. This is expert level consulting powered by AI.

---

## The Product

### What the Client Experiences

1. Confeiteira Maria finishes a beautiful cake. She snaps a photo and sends it to the Rekan WhatsApp number: "Bolo de casamento que entreguei hoje, tema rosa e dourado, a noiva amou."
2. Within a few minutes, she gets back: a caption that sounds like her (not generic AI), relevant hashtags, a visual tip ("filma um reels de 10 segundos cortando a primeira fatia, usa áudio X que tá em alta"), and a suggested posting time.
3. Maria copies, pastes, posts. Done. 2 minutes of her day.

### What Happens Behind the Scenes

Phase 1 (now): Elenice receives the WhatsApp message, runs it through the existing AI pipeline (already built in Go with PocketBase backend and content evaluation system), reviews the output, and sends it back manually.

Phase 2 (later): The whatsmeow bot automates the intake and response. Elenice only handles onboarding, monthly check ins, and edge cases.

### Elenice's Role

- **Sales and acquisition:** Finds new clients through her network, WhatsApp groups, local businesses, Sebrae events.
- **Onboarding:** 15 minute WhatsApp call with each new client to understand their business, voice, and audience.
- **Relationship management:** Monthly check ins, handling questions, being the human face of Rekan.
- **Quality review:** Reviews AI generated content before it goes out (Phase 1), handles anything the AI gets wrong.

---

## Pricing

Pricing is still being finalised but the range is R$49 to R$99/month. This positions Rekan:

- Above DIY tools (justifiable because we do the work for you)
- Far below human social media managers (10x cheaper)
- Affordable for any MEI

| Price Point | Clients Needed for R$2,000 | Elenice's Monthly Hours |
|---|---|---|
| R$49/month | ~41 clients | ~15 hours (maintenance) + acquisition time |
| R$79/month | ~26 clients | ~15 hours (maintenance) + acquisition time |
| R$99/month | ~21 clients | ~15 hours (maintenance) + acquisition time |

Each client requires roughly 15 to 20 minutes per month of Elenice's time once onboarded (Phase 1 will be more since she's doing manual processing).

### Monthly Cost Structure

- AI pipeline costs (API calls, hosting): R$50 to R$100
- Canva Pro: R$35
- ChatGPT Plus (backup): R$100
- MEI tax: R$81
- **Total overhead: approximately R$270 to R$320/month**

---

## Target Niche

Don't try to serve "all MEIs" at launch. Pick one vertical Elenice knows best: confeiteiras, cabeleireiras, or personal trainers. Build the AI prompts and templates specifically for that niche. Become known as "the Instagram partner for confeiteiras." Then expand.

Starting niche helps because:
- Content templates are reusable across similar businesses
- Word of mouth spreads fast within professional communities
- Elenice becomes genuinely expert in what works for that niche

---

## Competition

### GalilAI (primary competitor)

Based in Florianópolis. Charges R$28 to R$99/month. Generates posts with captions, hashtags, brand colours, auto publishing. Strong reviews. Growing.

**What GalilAI proves:** The market exists and people will pay for AI content in Brazil. This validates our idea.

**What GalilAI doesn't do:** Content direction (what photo to take, how to frame it), reel scripting, human relationship, WhatsApp delivery. Their users still have to sit down, open the app, navigate it, and manage the process.

### Generic AI Tools (Canva, ChatGPT, Predis.ai)

These are general purpose. Not localised for Brazilian MEIs. Require the user to know what to ask for. No relationship, no guidance.

### Human Social Media Managers

Start at R$590/month. Too expensive for most MEIs. But they offer the human touch that tools don't. Rekan offers that same human touch at a fraction of the price because AI does the production work.

### Our Moat

The combination of human relationship + AI speed + content direction delivered through WhatsApp. Each piece alone isn't unique. The combination is. And Elenice's sales ability and genuine care for small business owners is the competitive advantage that can't be copied by a tool.

---

## Risks and Mitigation

### Client Acquisition

The tech works. The market exists. But finding and converting those first 20 to 30 clients requires Elenice to consistently prospect. If she gets discouraged after 5 "no"s, the business stalls. Denis needs to be the support system here.

**Mitigation:** Start with businesses Elenice already knows. Use the "semana de prova" (7 day free trial) to reduce risk for new clients.

### Proving ROI

MEIs don't track metrics. "Followers up 20%" doesn't matter to a confeiteira. The proof they care about: "I got 3 new clients this month from Instagram."

**Mitigation:** Help clients set up simple tracking. A specific WhatsApp link in their bio. Asking new customers "how did you find us?"

### Churn

Some clients will leave after 1 to 2 months because they don't see immediate results or think they can do it themselves now. Plan for 20 to 30% monthly churn.

**Mitigation:** Keep the acquisition pipeline active. Focus on results clients can feel. Monthly check ins to show progress and adjust.

### Elenice Capacity

If she reaches 40+ clients doing manual processing, she'll be overloaded.

**Mitigation:** Phase 2 automation via whatsmeow bot. Also, this is a good problem to have. Solve it when you get there.

---

## Future Pivots

### Option 4: Sell to Social Media Managers (B2B)

Brazil has thousands of freelance social media managers charging R$500 to R$3,000/month per client. If the Rekan pipeline can help a social media manager serve 20 clients instead of 8 in the same time, they'd pay R$200 to R$300/month for the tool.

This is not the focus now but is a natural evolution once the pipeline is proven with real clients. The service business validates the product. If Phase 1 works, Phase 2 could be packaging the system for other people doing what Elenice does.

### Self Serve App

If demand warrants it, the pipeline can eventually be wrapped in a web UI for clients who want DIY at a lower price point. The WhatsApp engine is the same engine that would power a self serve product. But this comes later, if at all.

---

## Phase 1 Runbook for Elenice

This is the step by step plan for getting started. No automation needed. Elenice does everything manually using the tools already built.

### Week 1 to 2: Preparation

**Set up the Rekan WhatsApp Business number.** This is the business line, separate from personal. Use WhatsApp Business app (free). Set up the profile with Rekan logo, description, and business hours.

**Create the "cardápio" (menu of services).** A simple WhatsApp message she can forward to prospects:

> *Rekan: Seu parceiro de conteúdo no Instagram*
>
> Você manda uma mensagem sobre o que fez hoje no trabalho. A gente devolve o post pronto: legenda, hashtags, dica de foto e ideia de reels. Tudo pelo WhatsApp.
>
> Plano mensal: R$XX/mês
> Teste grátis por 7 dias
>
> Quer experimentar? Me manda uma mensagem!

**Choose the starting niche.** Elenice picks the vertical she knows best and feels most comfortable selling to.

**Prepare 3 to 5 example posts** for that niche using the existing pipeline. These are samples she can show prospects: "look at the kind of content we create for [confeiteiras/cabeleireiras/etc]."

### Week 2 to 4: First Clients (Free Trials)

**Goal: Get 5 businesses to try the 7 day free trial ("semana de prova").**

How to find them:
- Businesses Elenice already knows personally
- Local businesses in her neighbourhood she visits as a customer
- WhatsApp groups for empreendedores in her city
- Instagram: search for local MEIs in the chosen niche, look for profiles with inconsistent or low quality content, DM them
- Sebrae events and communities (free to attend)

**The pitch (WhatsApp or in person):**

> "Oi [nome], tudo bem? Eu tô começando um serviço de conteúdo pro Instagram focado em [confeiteiras/etc]. Funciona assim: você me manda uma foto ou conta o que fez hoje, e eu devolvo o post prontinho, com legenda, hashtags, e até ideia de reels. Tudo pelo WhatsApp. Quero te oferecer uma semana grátis pra você testar. Sem compromisso nenhum. Quer experimentar?"

**During the trial:**
1. Client sends a message about their work (photo, voice note, or text)
2. Elenice runs it through the AI pipeline
3. Elenice reviews the output, adjusts if needed
4. Sends back the complete content package via WhatsApp
5. Deliver 2 to 3 posts during the 7 day trial

**End of trial conversion:**

> "E aí [nome], gostou dos posts essa semana? Vi que o post de [X] teve bastante engajamento. Se quiser continuar, o plano é R$XX por mês. Posso te mandar conteúdo toda semana."

### Month 2 to 3: First Paying Clients

**Goal: Convert 5 to 10 trial clients into paying subscribers.**

**Consider introductory pricing** for the first 3 months (e.g. R$49 instead of R$79) to build the initial base and get testimonials.

**Establish a weekly rhythm:**
- Monday: Receive messages from clients about their week ahead / recent work
- Tuesday/Wednesday: Process through pipeline, review, send back content
- Thursday/Friday: Follow up with clients who haven't sent anything ("Oi [nome], como foi a semana? Tem algo legal pra gente postar?")

**Collect testimonials.** After 2 to 4 weeks, ask happy clients: "Posso usar seu depoimento no meu Instagram?" Screenshot their positive WhatsApp messages (with permission).

**Track simple metrics per client:**
- Posts delivered per week
- Client responsiveness (are they sending content regularly?)
- Any feedback on what works/doesn't
- New followers or clients they mention getting from Instagram

### Month 3+: Growth and Optimisation

**Continue prospecting.** Never stop. Even with 10 clients, keep the pipeline active because churn will happen.

**Ask for referrals.** "Você conhece alguma [confeiteira/cabeleireira] que também precisa de ajuda com Instagram? Se indicar alguém, dou 1 semana grátis pra vocês duas."

**Increase prices gradually** as demand grows. First clients at introductory price, new clients at full price.

**Consider expanding niches** once the first vertical is stable and you have 10+ clients.

**Start documenting patterns:** Which types of content get the most engagement? Which photo directions work best? What time of day works for posting in this niche? This becomes the knowledge base that makes the AI pipeline even better over time.

---

## Success Criteria

| Milestone | Timeline | Target |
|---|---|---|
| WhatsApp Business set up, samples ready | Week 2 | Done |
| First 5 free trials running | Week 4 | 5 businesses trying the service |
| First paying clients | Month 2 | 3 to 5 paying R$49 to R$79/month |
| Sustainable income | Month 4 to 6 | 15 to 25 clients, R$1,500+/month |
| Target income reached | Month 6 to 9 | 25 to 40 clients, R$2,000+/month net |

---

## What Denis Does

- Maintains and improves the AI content pipeline
- Helps Elenice troubleshoot technical issues
- Builds the whatsmeow automation when manual volume justifies it (Phase 2)
- Acts as business advisor and emotional support (the "no"s will be hard)
- Handles anything that requires technical knowledge
- Does NOT do the client facing work. That's Elenice's strength.

---

*Rekan: seu parceiro de conteúdo.*
