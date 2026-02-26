# PEP-010: Pricing Strategy

**Status:** In Progress (not yet live)
**Date:** 2026-02-24

## Context

Rekan currently has a flat pricing structure: R$19.90 first month, R$108.90/month after. One tier, no commitment options, no price anchoring. The customer sees R$108.90 in isolation and compares it to the only alternative they know: doing nothing (free).

This is a problem. R$108.90 is objectively cheap (10x less than a social media manager, less than R$4/day), but it doesn't feel cheap because there's no reference frame. A confeiteira who's never paid for content help has no way to evaluate whether R$108.90 is a bargain or a splurge.

Inspired by Jason Cohen's MicroConf talk on self-funded pricing: the goal isn't to lower the price. It's to structure the pricing so the customer looks at R$108.90 and thinks "this is obviously the smart choice." Three levers to pull: tiers (anchor against more expensive options), commitment discounts (reward longer sign-ups), and framing (anchor against the real alternative, a R$590 social media manager).

### Constraints

- Elenice sells through WhatsApp, not a pricing page. The "pricing table" is a message she sends. It must be scannable in a chat bubble.
- MEIs have irregular income. Large upfront payments (annual at R$1,000+) are unrealistic for most. Quarterly is the sweet spot.
- The low-risk entry point is a 30-day money-back guarantee at full price, not a discounted first month. Elenice's personal WhatsApp relationship provides the trust; the guarantee removes the remaining risk.
- Tier differences must be real and deliverable. No phantom features. If Profissional includes a monthly strategy call, Elenice must actually do the call.
- Payments must be fully automatic after the first authorization. No QR codes to scan on renewal, no reminders to send. The customer pays once, authorizes future debits, and never thinks about payment again.

### Principles from Cohen's talk applied here

See `docs/self-funded-playbook.md` for the full reference of all principles and how they map to Rekan. The ones directly driving pricing decisions:

1. **Three tiers, highlight the middle.** The bottom exists to make the middle obvious. The top exists for price anchoring and for MEIs who genuinely want more.
2. **Name the top tier "Profissional."** Business owners self-select into the professional/business tier automatically. (Cohen via Erica Douglas: "call the most expensive one the Business plan and people with businesses will sign up for that automatically.")
3. **Strikethrough pricing.** Show a higher "normal" price crossed out. This is not deceptive; R$149.90 becomes the listed price, and current customers get a founders' discount. (Cohen: "We wrote $79 and crossed it out and said $49. Sales went up.")
4. **Longer commitment = better deal.** Quarterly prepay gives Rekan cash upfront and reduces the monthly churn touchpoint. The MEI feels rewarded for committing.
5. **Coupons only on longer commitments.** Referral and ad coupons push toward quarterly, not monthly.
6. **Anchor against the real alternative.** Every pricing touchpoint should remind the MEI that a social media manager costs R$590+/month.
7. **Money-back guarantee over free trial.** Cohen switched WP Engine from a 15-day free trial to a 60-day money-back guarantee. Signups went up. People said "15 days wasn't enough time, but 60 days made me sign up." Collect full price from day 1, offer a generous refund window.
8. **ARPU is the most important metric when small.** More important than churn rate. Moving ARPU from R$108.90 to R$130 with a few Profissional clients matters more than reducing churn by 5%.
9. **Raise prices until something bad happens.** Cohen doubled a company from $19 to $49 per month, signups didn't change. Then he told them to double again. Keep raising until you see resistance.
10. **Boutique sympathy.** "If you say 'I am just one person really making a go of it,' that actually lets you charge more money because people think that's awesome." Elenice IS this. The "preco de lançamento" framing leans into it.
11. **Land and expand.** Start a client on Basico or Parceiro. Once they see results, upsell to the next tier. Cohen's SmartBear averaged $12k first transaction but $60k total over time. Same principle at micro scale: R$108.90/month client becomes R$249.90/month after 3 months of visible results.

## Wave 1: Launch (tiers, commitments, Pix Automatico)

Everything needed to go live. Three tiers, commitment plans, automatic payments via Pix Automatico, cardapio for Elenice. One release, one sales script.

### Tier design

| | Basico | **Parceiro** | Profissional |
|---|---|---|---|
| Monthly (listed) | R$69,90 | ~~R$149,90~~ **R$108,90** | R$249,90 |
| Posts/month | 8 | 12 | 16 |
| Legendas + hashtags | Yes | Yes | Yes |
| Direcao de foto/video | No | **Yes** | Yes |
| Roteiros de reels | No | **Yes** | Yes |
| Melhor horario pra postar | No | **Yes** | Yes |
| Chamada mensal de estrategia (30 min) | No | No | **Yes** |
| Calendario de stories | No | No | **Yes** |
| Resposta prioritaria (mesma hora) | No | No | **Yes** |

**Why these tiers:**

- **Basico** is intentionally unattractive. Captions and hashtags only, no creative direction. That's roughly what ChatGPT gives you. It exists so Parceiro looks like the obvious upgrade. Some MEIs will pick it, and that's fine: R$69.90/month for 8 AI-generated captions is still profitable with near-zero marginal cost.
- **Parceiro** is the real product. This is what Rekan actually does today. Photo direction, reels scripts, posting times. The strikethrough from R$149.90 to R$108.90 is the "preco de lançamento" (launch price), a real discount for early customers that creates urgency and gratitude.
- **Profissional** costs 2.3x Parceiro but only adds one 30-min call/month, a story calendar (template, not custom daily), and faster response. The marginal cost to Elenice is ~45 minutes/month per Profissional client. At R$249.90, that's extremely high-margin. It also makes R$108.90 look moderate by comparison. The "priority response" feature is Cohen's premium support trick: "you just make two queues and work them in a different order. Those people simply have faster response times. It's pretty much free money because you want to do all the tickets anyway." Elenice is already responding to all clients; Profissional clients just go first.

### Commitment pricing

| Plan | Mensal | Trimestral |
|---|---|---|
| Basico | R$69,90 | R$179,70 (R$59,90/mes) |
| **Parceiro** | **R$149,90** | **R$299,70 (R$99,90/mes)** |
| Profissional | R$249,90 | R$599,70 (R$199,90/mes) |

Note: the monthly Parceiro price is R$149.90 (full price). The "preco de lançamento" R$108.90 only applies to the monthly plan during the launch period, which makes the trimestral at R$99.90/month look like an even better deal. The MEI thinks: "I can pay R$149.90/month (or R$108.90 with the launch discount), OR I can commit to 3 months and pay R$99.90/month. Easy choice."

**Why trimestral is the sweet spot:** R$299.70 is roughly one week of average MEI income (R$6,750/month). That's a real but manageable commitment.

**The "infinite marketing budget" unlock.** Cohen: at WP Engine, 25% of signups chose annual prepay. That gave 3x the cash flow and meant they collected more cash each month than they spent on acquisition. "The marketing budget at WP Engine is not limited by money." For Rekan: if even 30% of clients choose trimestral, Elenice collects R$299.70 upfront per trimestral vs. R$108.90 per monthly. That's 2.75x the first-month cash. Combined with PEP-009 ad spend (R$40-55 acquisition cost per trial), trimestral clients pay back their acquisition cost on day 1. This means every real earned from ads can be immediately reinvested into more ads. The constraint shifts from cash to Elenice's capacity, which is where it should be.

### 30-day money-back guarantee (replaces R$19.90 first month)

Charge full price from day 1. Offer a 30-day money-back guarantee via Pix (instant, no fees). "Se em 30 dias voce nao sentir a diferenca no seu Instagram, devolvemos tudo pelo Pix. Sem perguntas."

**Why this replaces the R$19.90 first month:**

- **No month-2 price shock.** The R$19.90 → R$108.90 jump (5.5x) was the highest churn risk in the old model. With a guarantee, there is no price jump. The client already accepted R$108.90 on day 1.
- **5.5x more cash collected on day 1.** R$108.90 vs. R$19.90. With 10 signups in a month, that's R$1,089 vs. R$199 in the bank.
- **Signals confidence.** "Devolvo tudo se nao gostar" says Rekan believes in its own product. A discounted trial says "try it cheap because maybe you won't like it."
- **Almost nobody asks for refunds.** Cohen's WP Engine data: switched from trial to guarantee, signups went up, refund requests were negligible.
- **30 days is plenty.** Rekan delivers content within the first week. By day 14 the confeiteira has posted multiple times and can see the difference on her grid. Unlike hosting (where migration takes weeks), content value is visible in days. 60 days would just extend uncertainty for Elenice with no benefit to the client.
- **Pix makes it frictionless.** Refunding via Pix is instant and free. No credit card disputes, no payment processor delays. The guarantee is real and easy to honor.
- **Elenice's WhatsApp conversation IS the trust.** By the time she mentions price, the confeiteira already knows Elenice by name, has described her business, has seen example content, and feels heard. The guarantee is a safety net on top of an already warm relationship.

**The pitch:** "O Parceiro custa R$108,90 por mes. Menos de R$4 por dia. E se em 30 dias voce nao sentir a diferenca no seu Instagram, devolvo tudo pelo Pix na hora. Pode testar sem risco."

**What about ad-sourced leads?** Strangers from Meta ads don't have the personal trust from Elenice's conversation yet. But the 30-day money-back guarantee serves the same purpose: it removes risk without devaluing the service. Elenice qualifies them via WhatsApp first, then offers the standard deal. The guarantee is the low barrier; no special coupon needed.

### Payment infrastructure: Pix Automatico

The old subscription flow (Asaas `POST /subscriptions` with `BillingType: PIX`) generates a new QR code each billing cycle that the customer must manually scan and pay. This is a churn risk: the customer forgets, doesn't see the notification, or just doesn't feel like opening their banking app. Every renewal is a decision point where they can drift away.

Pix Automatico eliminates this. The customer authorizes recurring debits once. All future charges are auto-debited from their bank account on the due date. No QR codes, no scanning, no reminders. Payment becomes invisible, like a credit card subscription but via Pix.

**Pre-requisite:** The Asaas production account must have a Pix key (EVP) registered before creating authorizations. Without it, the API returns "Chave Pix nao encontrada." Create one via `POST /pix/addressKeys` with `{"type":"EVP"}` or through the Asaas dashboard.

**How it works on Asaas:**

1. **Authorization (once, on signup):** Call the Asaas authorization endpoint. Get back a QR code that combines the first charge + recurring debit authorization. Customer scans once. First payment is collected, and authorization becomes `ACTIVE`.

2. **Recurring charges (automatic):** For each future billing cycle, create a charge via `POST /payments` with the `pixAutomaticAuthorizationId` field set to the stored authorization ID. Asaas auto-debits the customer's bank account on the due date. The charge must be created 2-10 business days before the due date.

3. **PocketBase cron (scheduler):** A daily cron job checks which businesses have a charge due in 5 business days and creates the Asaas charge for them. This replaces Asaas's built-in subscription scheduler. The cron runs once per day, queries businesses by `next_charge_date`, and creates charges in batch.

**What this replaces:**

| Old (subscription + PIX) | New (Pix Automatico) |
|---|---|
| `POST /subscriptions` creates recurring billing | `POST` authorization endpoint on signup |
| Asaas generates charges + QR codes each cycle | Cron creates charges, Asaas auto-debits |
| Customer scans QR code every month/quarter | Customer authorizes once, never thinks about it |
| Store `subscription_id` per business | Store `pix_authorization_id` per business |
| Webhook: `PAYMENT_CONFIRMED` | Same + `PIX_AUTOMATIC_RECURRING_AUTHORIZATION_*` events |

**DB schema changes (businesses collection):**

- Remove: `subscription_id`
- Add: `authorization_id` (text, from Asaas authorization response, unique index)
- Add: `customer_id` (text, Asaas customer ID, persists across retries)
- Add: `tier` (select: `basico`, `parceiro`, `profissional`)
- Add: `commitment` (select: `mensal`, `trimestral`)
- Add: `next_charge_date` (date, when the next charge is due)
- Add: `charge_pending` (bool, set before creating Asaas charge, cleared on webhook confirmation)
- Add: `qr_payload` (text, Pix copia-e-cola string for inline QR code rendering)
- Note: `charge_amount` was dropped in favor of runtime computation via `pricing.Price(tier, commitment)`

**Webhook changes:**

New events to handle:
- `PIX_AUTOMATIC_RECURRING_AUTHORIZATION_ACTIVATED`: authorization confirmed, mark business as `active`
- `PIX_AUTOMATIC_RECURRING_AUTHORIZATION_CANCELLED`: customer or merchant cancelled, mark as `cancelled`
- `PIX_AUTOMATIC_RECURRING_AUTHORIZATION_REFUSED`: QR code expired without payment, mark as `payment_failed`
- `PIX_AUTOMATIC_RECURRING_PAYMENT_INSTRUCTION_REFUSED`: charge failed (insufficient funds, bank error), notify Elenice via WhatsApp
- `PAYMENT_CONFIRMED`: charge collected successfully, update `next_charge_date`

Remove old events: `PAYMENT_OVERDUE`, `SUBSCRIPTION_DELETED` (no longer using subscriptions).

**Invite accept flow (new):**

1. Client submits CPF/CNPJ + selected tier + selected commitment
2. Backend creates Asaas customer (same as before)
3. Backend creates Pix Automatico authorization (replaces subscription creation)
4. Backend stores `pix_authorization_id`, `tier`, `commitment`, `next_charge_date`, `charge_amount`
5. Frontend redirects to Asaas payment page (same UX as before, customer scans QR code once)
6. Webhook confirms authorization is active, business becomes `active`

**Cron job details:**

- Runs daily (e.g. 08:00 BRT)
- Queries: businesses where `invite_status = active` AND `next_charge_date` is within the next 5 business days AND no pending charge exists for that period
- For each: `POST /payments` with `pixAutomaticAuthorizationId`, `value` = `charge_amount`, `dueDate` = `next_charge_date`
- On success: mark charge as pending (to avoid duplicates)
- On `PAYMENT_CONFIRMED` webhook: advance `next_charge_date` by 1/3/6 months depending on `commitment`

### The cardapio (WhatsApp message)

- [x] Write the cardapio message in pt-BR. Three WhatsApp messages: anchor + value prop, tiers table, guarantee + soft close. Added to `docs/guia-de-vendas.md` under "Cardapio (mensagem pro WhatsApp)".
- [x] Add the social media manager anchor at the top: "Um social media manager cobra a partir de R$590/mes." Present in cardapio message 1 and marketing page pricing section.
- [x] Add daily price reframe for Parceiro: "Menos de R$4 por dia." Present in cardapio message 2 and marketing page Parceiro card.
- [x] Include the boutique/founder angle: "O Rekan e um servico pequeno e pessoal. Eu conheco seu negocio, acompanho toda semana e cobro quando voce esquece de mandar conteudo. Nao e ferramenta, e parceiro." In cardapio message 1.
- [x] Update `docs/guia-de-vendas.md` with the new cardapio and objection handling for "Por que tem tres planos?"
- [x] Update BUSINESS.md: replaced old R$19 first-month cardapio with reference to guia-de-vendas, updated pricing table to 3 tiers, changed "7 day trial" to "first week".
- [x] Update cardapio to include commitment options (mensal/trimestral prices). Added trimestral column to plans table, added trimestral pitch to cardapio message 3, added note about strikethrough pricing strategy.
- [x] Write the "trimestral pitch" for when the prospect is interested. Added to cardapio message 3 and "E caro" objection section.

### Code changes

- [x] Delete old subscription code: `CreateSubscription`, `UpdateSubscription`, `GetSubscription`, `CancelSubscription` from `asaas/client.go`. Delete `CreateSubscriptionReq` and `Subscription` types. Also removed `get()` and `put()` helpers.
- [x] Add Pix Automatico methods to `asaas/client.go`: `CreateAuthorization(ctx, req) (Authorization, error)`, `CreateCharge(ctx, req) (Charge, error)`, `CancelAuthorization(ctx, id) error`.
- [x] Add DB migration: new fields `authorization_id`, `customer_id`, `tier`, `commitment`, `next_charge_date`, `charge_pending`, `qr_payload` on businesses. Unique index on `authorization_id`. Remove `subscription_id`. Note: `charge_amount` was replaced by runtime lookup via `pricing.Price(tier, commitment)`.
- [x] Rewrite `invite.go` InviteAccept: create authorization instead of subscription, accept tier + commitment from request body. Uses DB transaction to atomically claim the invite (`invited` -> `accepted`) preventing duplicate Asaas authorizations. Reuses existing `customer_id` on retry. Returns `qr_payload` instead of `payment_url`. Reverts to `invited` on failure so user can retry.
- [x] Update `convite/[token]/+page.svelte`: shows tier, commitment, and calculated price. Renders Pix QR code inline (via `qrcode` npm package) instead of redirecting to Asaas payment page. Polls for status change every 5s. Confirmation page (`confirmacao/+page.svelte`) gutted to a redirect shim.
- [x] Rewrite `webhooks.go`: handles all Pix Automatico events (`AUTHORIZATION_ACTIVATED`, `_REFUSED`, `_CANCELLED`, `_EXPIRED`, `PAYMENT_CONFIRMED`, `PAYMENT_INSTRUCTION_REFUSED`, `PAYMENT_INSTRUCTION_CANCELLED`). All handlers are idempotent. Business lookup by `authorization_id` instead of `subscription_id`.
- [x] Add PocketBase cron job: `billing.CreatePendingCharges()` runs daily at 10:00. Queries businesses due within 7 days, sets `charge_pending = true` before calling Asaas (crash safety), rolls back on failure. Registered in `main.go`. Nil-client guard for dev environments.
- [x] Update `(marketing)/+page.svelte`: added mensal/trimestral toggle with dynamic prices. Trimestral view shows effective per-month price prominently with total below. WhatsApp CTA includes selected commitment.
- [x] Update tests: `invite_test.go` and `webhooks_test.go` rewritten for new flow. New `billing/charges_test.go` (5 cases: charge creation, skip pending, skip far dates, skip non-active, nil client). New `asaas/sandbox_test.go` for real Asaas sandbox integration (skips without `ASAAS_SANDBOX_KEY`).

**Additional work done (not in original checklist):**

- [x] New `api/internal/pricing/` package: three-tier pricing matrix with `Price(tier, commitment)` lookup, `Months` map, validators. Replaces hardcoded `PriceParceiro` constant.
- [x] New `api/internal/domain/` package: centralizes collection names, invite statuses, webhook event names, message types, billing type, post source. All handlers refactored to use these constants instead of raw strings.
- [x] Route renamed: `/api/businesses/{id}/subscription:cancel` -> `/api/businesses/{id}/authorization:cancel`.
- [x] Frontend types updated: `Business` interface reflects new fields (`authorization_id`, `customer_id`, `tier`, `commitment`, `next_charge_date`, `charge_pending`). New `Tier` and `Commitment` type aliases.

**Gate:** End-to-end test in Asaas sandbox: create authorization, confirm first payment, cron creates next charge, auto-debit succeeds, webhook updates next_charge_date. Test with all 3 tiers and all 3 commitment periods.

## Wave 2: Coupons, referrals, and ad integration

Use pricing structure to amplify acquisition channels from PEP-009.

### Referral program

Current plan (BUSINESS.md): "indica alguem, 1 semana gratis pra voces duas." This is weak. Cohen says pay affiliates a lot because it's worth it.

New structure:
- Referred person gets the standard offer: full price + 30-day money-back guarantee. No special discount. The friend's recommendation + the guarantee is enough trust.
- Referrer gets 1 free month, but only after the referred person stays past the 30-day guarantee window. This aligns incentives: the referrer recommends Rekan to people who'll actually use it, not just anyone.
- Implementation: Elenice tracks "Client A referred Client B" in a spreadsheet. After 30 days, if B is still active, Elenice opens PocketBase admin and pushes Client A's `next_charge_date` forward by one month. The billing cron skips the cycle automatically (it only picks up businesses where `next_charge_date` is within 7 days). Zero code changes, zero complexity.

Elenice's pitch: "Voce conhece alguem que tambem precisa de ajuda com Instagram? Se voce indicar e a pessoa assinar e ficar, voce ganha 1 mes gratis."

### Ad coupons (PEP-009 integration)

Meta ads drive click-to-WhatsApp conversations. Ad-sourced leads are strangers with no prior relationship. They get the same offer as everyone: full price + 30-day money-back guarantee. Elenice qualifies them first via WhatsApp, then pushes toward trimestral for the better per-month price.

- [ ] Elenice qualifies them ("Voce e confeiteira? Me conta do seu trabalho.").
- [ ] If qualified, offers the standard deal: full price + 30-day money-back guarantee. Same offer as everyone else. No special coupon.
- [ ] Push toward trimestral: "O plano trimestral sai por R$99,90/mes. E se em 30 dias voce nao gostar, devolvo tudo." The guarantee removes risk for strangers; the trimestral discount rewards commitment.
- [ ] Economics: cost per WhatsApp conversation target from PEP-009 is R$5-8. At 15% conversion, acquisition cost is ~R$40-55 per client. Trimestral Parceiro pays R$299,70 upfront, covering acquisition cost on day 1.

### Pricing in ad creatives

- [ ] Update PEP-009 copy guidelines: prospecting ads never show price. Retargeting ads show the anchor: "Social media manager: R$590/mes. Rekan: a partir de R$69,90/mes." The "a partir de" is the Basico price, which makes even the Parceiro tier feel mid-range.
- [ ] For retargeting ads specifically, test showing the trimestral price: "R$99,90/mes no plano trimestral" with a CTA to WhatsApp.

**Gate:** Referral program has generated at least 5 referred clients. Ad-sourced clients are converting to trimestral at >30% rate. Track acquisition cost per channel and payback period.

## Wave 3: Price testing and iteration

Once the structure is live and there are 15+ clients across tiers:

### Track and measure

- [ ] Track tier distribution. If >80% pick Parceiro, the tiers are working as designed. If >30% pick Basico, the gap between Basico and Parceiro is too large or the value difference isn't clear. Adjust.
- [ ] Track commitment distribution. If <20% pick trimestral, test making the monthly price higher (R$169,90) to widen the gap.
- [ ] Track ARPU monthly. This is the north star metric when small (Cohen: "more important than cancellation rate"). Target: R$120+ ARPU across all tiers.
- [ ] Track acquisition cost per channel (organic/referral/Meta ads) and payback period per commitment type.

### Raise prices until something breaks

Cohen's rule: "double it and see what happens. If signups don't change, double it again." Applied to Rekan:

- [ ] Test removing the "preco de lançamento" for new clients. If signups don't drop, R$149,90/month becomes the real price and ARPU jumps ~40%.
- [ ] Test a R$299,90 Parceiro monthly price with an aggressive trimestral discount (R$149,90/month). Cohen's Capital Factory story: raise the monthly, discount the quarterly heavily. If signups don't change, revenue goes up for monthly payers and trimestral becomes irresistible.
- [ ] Test raising Profissional to R$349,90. Cohen: "people who decide they always want the most expensive thing will pay it." If even 2-3 clients stay at the higher price, that's +R$300/month for zero extra work.
- [ ] If raising prices causes some Basico clients to leave, that may be fine. Cohen: "raising prices changes the clientele. Some people who are not really serious will drop off." Fewer low-value clients means more time for high-value ones.

### Optimize the tiers

- [ ] Evaluate whether Profissional clients actually use the strategy call. If most skip it, consider replacing it with something lower-effort (async voice note review of their grid, curated content calendar doc).
- [ ] Track refund rate on the 30-day money-back guarantee. If >15% request refunds, investigate why (bad fit? overpromising in sales? content quality?). Cohen's data suggests refund rates should be well under 10%.
- [ ] Consider adding a "Parceiro+" at R$179.90 between Parceiro and Profissional if there's demand for more posts but not a strategy call. Four tiers can work if the differences are clear.

**Gate:** Data from 15+ clients over 2+ months. Clear picture of which tier and commitment length performs best. ARPU tracked and trending up. Pricing adjusted based on evidence, not guesses.

## Consequences

- Elenice's sales conversations become slightly more complex (3 tiers + commitment options vs. one price). The cardapio must be clear enough that it doesn't slow down the WhatsApp flow. If it confuses prospects, simplify back to 2 tiers.
- The R$19.90 first month is gone entirely. Everyone gets the same offer: full price + 30-day money-back guarantee. The barrier to trying Rekan is now R$69.90 (Basico) or R$108.90 (Parceiro with founder discount). Some prospects who would have tried at R$19.90 won't try at R$69.90. The trade-off is worth it: the guarantee removes risk without devaluing the service, there's no month-2 price shock, and Rekan collects 5.5x more cash on day 1.
- **Payments are invisible after signup.** Pix Automatico means the customer authorizes once and never thinks about payment again. This eliminates the biggest churn vector for MEIs with irregular schedules: forgetting or postponing a manual Pix payment. Every renewal that would have been a "do I still want this?" moment becomes a non-event.
- Trimestral clients give Rekan 3 months of runway per sign-up. With 15 trimestral Parceiro clients, that's R$4,495 in the bank covering 3 months. This changes the business from "will we make rent this month" to "we have a quarter of visibility." That psychological shift matters enormously for Elenice's confidence and for reinvesting in ads.
- The Profissional tier creates a natural upsell path. A confeiteira who's been on Parceiro for 3 months and seeing results is a warm lead for "quer levar pro proximo nivel?" at R$249.90. No new acquisition cost, just more revenue from existing clients.
- BUSINESS.md target of 21 clients at R$108.90 for R$2,000/month changes. With tiers: 10 Parceiro trimestral (R$99.90) + 3 Profissional (R$249.90) + 5 Basico (R$69.90) = R$999 + R$749.70 + R$349.50 = R$2,098/month. Fewer total clients needed if the mix includes Profissional. More resilient because revenue isn't uniform.
- Price anchoring against social media managers (R$590+) reframes every future conversation about "is Rekan worth it?" The answer becomes self-evident. This is the single most important framing change and costs nothing to implement.
- **Land and expand becomes the growth engine.** Cohen's SmartBear: $12k first transaction, $60k lifetime. For Rekan: a client starts on Basico (R$69.90) or Parceiro (R$108.90), sees results after 2-3 months, and Elenice suggests Profissional (R$249.90). No acquisition cost on the upgrade. This is why the Profissional tier must genuinely deliver more value, not just be a price anchor. The strategy call is the differentiator that justifies the jump.
- **The success dilemma arrives earlier with tiers.** Cohen warns: if the business works, it grows, and then you're managing people instead of building. With 3 tiers, Profissional clients require more of Elenice's time (strategy calls). At 10 Profissional clients, that's 5 hours/month of calls alone. Phase 2 automation (whatsmeow bot) becomes urgent sooner. Plan for this.
- **The cron job is a new operational dependency.** If it fails, charges don't get created and payments don't happen. Needs monitoring: if the cron runs and creates zero charges on a day when charges were expected, alert Elenice. PocketBase logs are sufficient for now, a proper alerting system can wait until there are 20+ clients.
