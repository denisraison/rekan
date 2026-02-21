# PEP-005: Backend MVP

**Status:** In Progress (Wave 1 done, Wave 2 done, Wave 3 done)
**Date:** 2026-02-20

## Context

The eval pipeline (PEP-001, 002, 003) validates that content generation works. The frontend has a marketing page. But there's no product yet: no way for a real user to sign up, describe their business, generate content, or pay.

This PEP defines the minimum backend needed to put Rekan in front of paying users. One business per account, monthly subscription via PIX (Asaas), content generation with hook-based rotation.

## Decisions

- **Everything as code:** No manual configuration via PocketBase admin UI. Collections, API rules, OAuth providers, rate limits, hooks, and settings are all defined in Go migrations and code. A fresh `go run .` on an empty database must produce a fully configured, secure instance. The admin UI is a debugging tool only.
- **Auth:** Google OAuth via PocketBase (no email/password for MVP)
- **Data access:** Firebase-style. The SvelteKit frontend talks directly to PocketBase collections via the JS SDK. No custom REST endpoints for CRUD.
- **Custom endpoints:** Only for server-side logic the client cannot perform: content generation (LLM calls), billing (Asaas API keys), webhooks (server-to-server). Follow Google's resource-oriented API design: resources are nouns, custom methods use `:verb` suffix.
- **Onboarding:** Staged form matching eval testdata structure (basics, services, personality)
- **Post-gen:** Display, save, and edit generated content
- **Billing:** Monthly subscription via Asaas (PIX + boleto + card)
- **Rotation:** Backend tracks previous hooks per business (PEP-003 integration)
- **Multi-business:** One business per account (simplifies data model)

## Architecture

```
┌─────────────────────────────────────────────────┐
│  SvelteKit Frontend                             │
│                                                 │
│  pb.collection('businesses').create(...)  ──────┼──► PocketBase Collections
│  pb.collection('posts').getList(...)      ──────┼──►  (direct SDK access)
│  pb.authWithOAuth2({ provider: 'google' })──────┼──►
│                                                 │
│  POST /api/businesses/{id}/posts:generate ──────┼──► Custom Go Endpoints
│  POST /api/subscriptions                  ──────┼──►  (server-side only)
│                                                 │
└─────────────────────────────────────────────────┘
                                                  │
                          Asaas webhooks ─────────┼──► POST /api/webhooks/asaas
```

### What PocketBase handles (no custom code)

- **Auth**: Google OAuth2, token refresh, session management
- **CRUD**: businesses, posts collections. Create, read, update, delete via JS SDK.
- **Authorization**: Collection API rules. Users can only access their own data.
- **Filtering/sorting**: `pb.collection('posts').getList(1, 12, { sort: '-created' })`
- **Realtime**: Live subscription to collection changes (if needed later)

### What needs custom Go endpoints (3 total)

Following [Google's API design guide](https://cloud.google.com/apis/design): resources are nouns, standard methods map to HTTP verbs, custom methods use `:verb` suffix on the resource.

1. **`POST /api/businesses/{id}/posts:generate`** - Custom method on the business resource. Creates posts via LLM generation pipeline.
2. **`POST /api/subscriptions`** - Standard Create on the subscription resource. Returns Asaas payment link.
3. **`POST /api/webhooks/asaas`** - Webhook callback. Not user-facing, no resource semantics needed.

## Everything as code

The PocketBase admin UI must never be the source of truth. Every configuration lives in Go code, version-controlled and reproducible. Deleting `pb_data/` and running `go run .` must produce a fully working, secure instance.

### Project structure

Following the same layout as guardiansgamers/api:

```
api/
  main.go                              # PocketBase init + migratecmd + hook/route registration
  migrations/
    1740000000_users_collection.go     # extend users: subscription fields + OAuth2 + API rules
    1740000001_businesses_collection.go # businesses collection + fields + API rules
    1740000002_posts_collection.go     # posts collection + fields + API rules
    1740000003_app_settings.go         # app name, URL, sender
    1740000004_rate_limits.go          # per-endpoint rate limits
    1740000005_trusted_proxy.go        # proxy headers for correct client IP
  internal/
    http/
      hooks.go                         # RegisterHooks: auth hooks (trial defaults)
      routes.go                        # RegisterRoutes: custom endpoints
      handlers/
        generate.go                    # POST /api/businesses/{id}/posts:generate
        subscribe.go                   # POST /api/subscriptions
        webhooks.go                    # POST /api/webhooks/asaas
        deps.go                        # Deps struct for dependency injection
```

### What goes where

| Configuration | Mechanism | Why not admin UI |
|---------------|-----------|------------------|
| Collections + fields | Go migration (`core.NewBaseCollection`, `Fields.Add`) | Version-controlled, reviewable, reproducible |
| API rules (list/view/create/update/delete) | Go migration (`collection.ListRule = &rule`) | Security rules must be code-reviewed, not clicked |
| Field-level protections (`:isset`) | Part of API rule strings in migration | Same as above |
| Unique indexes | Go migration (`collection.AddIndex`) | Schema integrity |
| Google OAuth provider | Go migration (`collection.OAuth2.Providers`, `collection.OAuth2.Enabled`) | Credentials from env vars, not pasted into UI |
| OAuth credentials | Environment variables (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`) | Secrets never in code or UI |
| Rate limits | Go migration (`app.Settings().RateLimits`) | Consistent across deployments |
| Trusted proxy headers | Go migration (`app.Settings().TrustedProxy`) | Rate limiting needs correct client IP |
| App settings (name, URL) | Go migration (`app.Settings().Meta`) | Reproducible |
| Trial defaults on new user | Go hook (`OnRecordAuthWithOAuth2Request`) | Logic, not manual setup |
| Custom endpoints | `RegisterRoutes` in `OnServe` hook | Application code |
| Dev mode overrides | `main.go` (`disableRateLimits`, email logging) | Environment-aware |

### Migration: users collection (OAuth + fields + rules)

```go
// migrations/1740000000_users_collection.go
package migrations

import (
    "os"
    "github.com/pocketbase/pocketbase/core"
    m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    m.Register(func(app core.App) error {
        collection, err := app.FindCollectionByNameOrId("users")
        if err != nil {
            return nil
        }

        // Google OAuth only, no password auth
        collection.PasswordAuth.Enabled = false
        collection.OAuth2.Enabled = true
        collection.OAuth2.Providers = []core.OAuth2ProviderConfig{
            {
                Name:         "google",
                ClientId:     os.Getenv("GOOGLE_CLIENT_ID"),
                ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
            },
        }

        // Server-managed fields (client cannot write these)
        collection.Fields.Add(
            &core.SelectField{
                Name:      "subscription_status",
                Values:    []string{"trial", "active", "past_due", "cancelled"},
                MaxSelect: 1,
            },
            &core.TextField{Name: "subscription_id"},
            &core.NumberField{Name: "generations_used"},
        )

        // API rules: user can view own record, cannot write server fields
        viewRule := "id = @request.auth.id"
        updateRule := "id = @request.auth.id" +
            " && @request.body.subscription_status:isset = false" +
            " && @request.body.subscription_id:isset = false" +
            " && @request.body.generations_used:isset = false"

        collection.ListRule = nil // no client listing of other users
        collection.ViewRule = &viewRule
        collection.UpdateRule = &updateRule
        collection.DeleteRule = nil // users cannot self-delete

        return app.Save(collection)
    }, func(app core.App) error {
        return nil
    })
}
```

### Migration: rate limits

Same pattern as guardiansgamers, with rules tuned for Rekan's endpoints:

```go
// migrations/1740000004_rate_limits.go
func init() {
    m.Register(func(app core.App) error {
        settings := app.Settings()
        settings.RateLimits.Enabled = true
        settings.RateLimits.Rules = []core.RateLimitRule{
            // Generation: expensive (LLM call). 10 per 5 minutes.
            {Label: "/api/businesses/", Duration: 300, MaxRequests: 10},
            // Subscribe: 3 per 15 minutes (prevent abuse)
            {Label: "/api/subscriptions", Duration: 900, MaxRequests: 3},
            // Auth endpoints
            {Label: "*:auth", Duration: 3, MaxRequests: 5},
            // Collection creates
            {Label: "*:create", Duration: 5, MaxRequests: 20},
            // Global fallback
            {Label: "/api/", Duration: 10, MaxRequests: 300},
        }
        return app.Save(settings)
    }, func(app core.App) error {
        settings := app.Settings()
        settings.RateLimits.Enabled = false
        return app.Save(settings)
    })
}
```

### Migration: trusted proxy

Without this, rate limits apply to the proxy IP instead of the actual client. Must match the deployment platform.

Deployed on Hetzner VPS behind Nginx/Caddy, which sets `X-Real-IP`.

```go
// migrations/1740000005_trusted_proxy.go
func init() {
    m.Register(func(app core.App) error {
        settings := app.Settings()
        settings.TrustedProxy.Headers = []string{"X-Real-IP"}
        settings.TrustedProxy.UseLeftmostIP = true
        return app.Save(settings)
    }, func(app core.App) error {
        settings := app.Settings()
        settings.TrustedProxy.Headers = []string{}
        settings.TrustedProxy.UseLeftmostIP = false
        return app.Save(settings)
    })
}
```

### main.go setup

Following the guardiansgamers pattern: `run(getenv)`, migrations import, hooks/routes registration, dev mode overrides.

```go
func run(getenv func(string) string) error {
    app := pocketbase.New()
    isDev := getenv("DEV_MODE") == "true"

    migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
        Dir:         "migrations",
        Automigrate: true,
    })

    // Register hooks (before app.Start)
    apphttp.RegisterHooks(app)

    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        if isDev {
            disableRateLimits(app)
        }
        apphttp.RegisterRoutes(se.Router, handlers.Deps{
            App: app,
            // ... other deps (eval client, Asaas client)
        })
        return se.Next()
    })

    return app.Start()
}
```

### Hook: trial defaults on new user

```go
// internal/http/hooks.go
func RegisterHooks(app *pocketbase.PocketBase) {
    app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
        if err := e.Next(); err != nil {
            return err
        }
        if e.IsNewRecord {
            e.Record.Set("subscription_status", "trial")
            e.Record.Set("generations_used", 0)
            e.App.Save(e.Record)
        }
        return nil
    })
}
```

### Route registration

```go
// internal/http/routes.go
func RegisterRoutes(rtr *router.Router[*core.RequestEvent], deps handlers.Deps) {
    auth := apis.RequireAuth()

    // Google-style: custom method on resource uses :verb suffix
    rtr.POST("/api/businesses/{id}/posts:generate", handlers.GeneratePosts(deps)).Bind(auth)

    // Standard Create on subscription resource
    rtr.POST("/api/subscriptions", handlers.CreateSubscription(deps)).Bind(auth)
    rtr.GET("/api/subscriptions/current", handlers.GetSubscription(deps)).Bind(auth)

    // Webhook callback (server-to-server, no auth middleware)
    rtr.POST("/api/webhooks/asaas", handlers.AsaasWebhook(deps))
}
```

### Environment variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `DEV_MODE` | Disable rate limits, log emails | No |
| `GOOGLE_CLIENT_ID` | Google OAuth2 client ID | Yes |
| `GOOGLE_CLIENT_SECRET` | Google OAuth2 client secret | Yes |
| `OPENROUTER_API_KEY` | LLM generation (via eval pipeline) | Yes |
| `ASAAS_API_KEY` | Asaas billing API | Yes (prod) |
| `ASAAS_WEBHOOK_TOKEN` | Validate Asaas webhook signatures | Yes (prod) |

Loaded via `.env` (gitignored) through direnv / flake.nix dev shell.

### Verification checklist

Before any deployment, verify the "everything as code" guarantee:

1. Delete `api/pb_data/` entirely
2. Run `cd api && go run .`
3. Confirm: collections exist with correct fields
4. Confirm: API rules reject unauthorized access (test each threat model row)
5. Confirm: Google OAuth login works
6. Confirm: rate limits are active
7. Confirm: new user gets `subscription_status: "trial"`
8. Confirm: admin UI shows all settings as expected (read-only verification)

## Wave 1: Collections and Auth

Set up PocketBase collections and Google OAuth. After this wave a user can sign in and land on an empty dashboard.

### Collections

```
users (PocketBase built-in, extended)
  - subscription_status: select (trial|active|past_due|cancelled)
  - subscription_id: text (Asaas subscription ID)
  - generations_used: number (tracks trial usage)

businesses
  - user: relation(users), unique, required
  - name: text, required
  - type: text, required
  - city: text, required
  - state: text, required
  - description: text
  - services: json (array of {name, price_brl}), required
  - target_audience: text
  - brand_vibe: text
  - quirks: text
  - onboarding_step: number (1=basics, 2=services, 3=personality, 4=complete)

posts
  - business: relation(businesses), required
  - caption: text, required
  - hashtags: json (array of strings)
  - production_note: text
  - role: text (from PEP-003 role pool)
  - hook: text (first sentence, extracted for rotation)
  - batch_id: text (groups posts from same generation)
  - edited: bool
```

### API rules and field-level security

With Firebase-style direct SDK access, API rules ARE the security layer. Every rule must be explicit. PocketBase uses `null` to mean "superusers only" (locked), empty string `""` for "anyone", and a filter expression for conditional access.

Field-level protection uses PocketBase's `:isset` modifier: `@request.body.field:isset = false` prevents the client from submitting that field at all. The `:changed` modifier (`@request.body.field:changed = false`) prevents modifying a field's value while still allowing it to be sent.

All rules are defined in Go migrations (not the admin UI) so they're version-controlled and reproducible.

#### users (auth collection, extended)

```
listRule:   null                        -- no client listing of other users
viewRule:   id = @request.auth.id       -- can only view own record
createRule: null                        -- PocketBase handles via OAuth
updateRule: id = @request.auth.id       -- can update own profile
              && @request.body.subscription_status:isset = false
              && @request.body.subscription_id:isset = false
              && @request.body.generations_used:isset = false
deleteRule: null                        -- users cannot self-delete
```

**Why:** `subscription_status`, `subscription_id`, and `generations_used` are server-managed fields. A malicious client could set `subscription_status = "active"` or `generations_used = 0` to bypass billing. The `:isset = false` rules make these fields completely invisible to client writes. Only server-side Go code (generate endpoint, webhook handler) can modify them.

#### businesses

```
listRule:   user = @request.auth.id     -- only own business
viewRule:   user = @request.auth.id     -- only own business
createRule: @request.auth.id != ""
              && @request.body.user = @request.auth.id
updateRule: user = @request.auth.id
              && @request.body.user:isset = false
deleteRule: user = @request.auth.id
```

**Why:**
- **createRule** ensures the client can only create a business linked to their own user ID. Without `@request.body.user = @request.auth.id`, a user could create a business under someone else's ID.
- **updateRule** with `@request.body.user:isset = false` prevents reassigning a business to a different user. Without this, a user could change the `user` field to hijack another account's business.
- One business per user is enforced via a unique index on the `user` field.

#### posts

```
listRule:   business.user = @request.auth.id
viewRule:   business.user = @request.auth.id
createRule: null                        -- server-only (generate endpoint)
updateRule: business.user = @request.auth.id
              && @request.body.business:isset = false
              && @request.body.batch_id:isset = false
              && @request.body.role:isset = false
              && @request.body.hook:isset = false
deleteRule: business.user = @request.auth.id
```

**Why:**
- **createRule = null** is critical. Posts are only created by the generate endpoint, which sets `hook`, `batch_id`, and `role` correctly. If clients could create posts, they could inject fake hooks that pollute the rotation system, or create posts without going through the generation pipeline.
- **updateRule** locks down server-managed fields. The client can edit `caption`, `hashtags`, `production_note`, and `edited` (the user-facing content). But `business`, `batch_id`, `role`, and `hook` are immutable from the client. Moving a post to a different business or tampering with rotation metadata would break data integrity.
- **list/view rules** use `business.user` (relation traversal) to ensure users only see posts belonging to their own business.

### Threat model summary

| Attack | Mitigation |
|--------|------------|
| User sets `subscription_status = "active"` | `:isset = false` on users updateRule |
| User resets `generations_used = 0` | `:isset = false` on users updateRule |
| User creates business under another user's ID | `@request.body.user = @request.auth.id` on createRule |
| User reassigns business to another user | `@request.body.user:isset = false` on updateRule |
| User creates posts directly (bypassing generation) | `createRule = null` on posts |
| User moves post to another business | `@request.body.business:isset = false` on updateRule |
| User tampers with hook/role/batch_id | `:isset = false` on those fields in updateRule |
| User lists/views other users' posts | `business.user = @request.auth.id` on list/viewRule |
| User deletes another user's data | Owner check on all deleteRules |

### Auth

All configured via Go code (see "Everything as code" section above):

- Google OAuth2 provider configured in migration with credentials from env vars
- Password auth disabled (Google only for MVP)
- Trial defaults set via `OnRecordAuthWithOAuth2Request` hook on first login
- Frontend: `pb.collection('users').authWithOAuth2({ provider: 'google' })`

### Acceptance criteria

- [x] Collections exist with all API rules and field-level protections (`migrations/174000000[0-2]_*.go`)
- [x] Google OAuth2 configured in migration with credentials from env vars (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`)
- [x] Password auth disabled
- [x] Trial defaults set on first login (`internal/http/hooks.go`)
- [x] Rate limits and trusted proxy configured as code (`migrations/174000000[4-5]_*.go`)
- [x] Route skeleton wired: generate, subscriptions, webhook stubs (`internal/http/routes.go`)
- [x] `go run .` on empty `pb_data/` produces a fully configured instance
- [ ] Google sign-in works end-to-end from SvelteKit (requires Wave 2 frontend)
- [ ] Manual threat model verification: attempt each attack row and confirm rejection

## Wave 2: Onboarding and Business Profile

Staged form in SvelteKit that creates/updates the business via PocketBase SDK.

### Step 1: Basics

Fields: name, type (dropdown with common Brazilian business types), city, state.

Business types from eval testdata: Salão de Beleza, Restaurante, Personal Trainer, Nail Designer, Confeitaria, Barbearia, Loja de Roupas, Pet Shop, Banda Musical, Estúdio de Tatuagem, Hamburgueria, Loja de Açaí. Plus "Outro" with free text.

Frontend calls: `pb.collection('businesses').create({ user: authUser.id, name, type, city, state, onboarding_step: 1 })`

### Step 2: Services

Dynamic list of services with name + price (BRL). Minimum 1 service.

Frontend calls: `pb.collection('businesses').update(id, { services: [...], onboarding_step: 2 })`

### Step 3: Personality (optional)

Target audience, brand vibe, quirks. Sensible defaults if skipped ("Público geral", "Profissional e acolhedor", empty quirks).

Frontend calls: `pb.collection('businesses').update(id, { target_audience, brand_vibe, quirks, onboarding_step: 3 })`

### Acceptance criteria

- [x] Three-step form flow works using direct PocketBase SDK calls (`routes/(app)/onboarding/+page.svelte`)
- [x] No custom backend endpoints needed for onboarding
- [x] Business saved with correct onboarding_step tracking
- [x] Defaults applied for step 3 if skipped ("Público geral", "Profissional e acolhedor")

## Wave 3: Content Generation Endpoint

The one custom endpoint that makes the product work. Everything else is PocketBase SDK.

### Integration approach

Import the eval pipeline as a Go package. The API calls `Generate()` with the business profile converted to `BusinessProfile` struct. This avoids duplicating BAML prompts or running a separate service.

### Endpoint

```
POST /api/businesses/{id}/posts:generate
  Auth: PocketBase auth token (validated server-side)
  Response: {
    batch_id: string,
    posts: [
      { id, caption, hashtags, production_note, role, hook }
    ]
  }
```

The business ID is in the URL path, not the request body. No body needed for this endpoint.

### Flow

1. Validate auth token, extract user ID
2. Load business `{id}` from PocketBase, verify user owns it
3. Check `subscription_status` (must be `trial` with remaining generations, or `active`)
4. Convert business record to eval's `BusinessProfile` struct
5. Pick 3 random roles (via `role.PickRoles`)
6. Load previous hooks: query `posts` collection for this business's hook field values
7. Call `Generate(profile, roles, previousHooks)`
8. Extract hooks from generated posts
9. Save posts to `posts` collection with batch_id, roles, hooks (server-side insert bypasses API rules)
10. If trial, increment `generations_used`
11. Return generated posts

### Acceptance criteria

- [x] Authenticated user can generate content for their business
- [x] Generated posts appear in `posts` collection (queryable via SDK)
- [x] Previous hooks loaded and passed to generation (rotation works)
- [x] Trial usage tracked, generation blocked when trial exhausted and no subscription (limit: 3)
- [x] Proper error response if LLM is down (502 Bad Gateway)

## Wave 4: Post Management

No custom backend work. PocketBase SDK + SvelteKit UI.

### Frontend operations (all via PocketBase JS SDK)

```js
// List posts by batch, newest first
pb.collection('posts').getList(page, 12, {
  filter: `business = "${businessId}"`,
  sort: '-created',
})

// Edit a post
pb.collection('posts').update(postId, {
  caption: editedCaption,
  hashtags: editedHashtags,
  edited: true,
})

// Delete a post
pb.collection('posts').delete(postId)
```

### Behavior

- Editing a post sets `edited: true`
- Posts grouped by `batch_id` in the UI (frontend logic)
- API rules ensure user can only access their own posts

### Acceptance criteria

- List, edit, delete work via PocketBase SDK
- API rules enforced (no access to other users' posts)
- Edited flag tracked correctly

## Wave 5: Billing (Asaas Integration)

Two custom endpoints: one to create a subscription, one to receive webhooks.

### Endpoints

```
POST /api/subscriptions
  Auth: PocketBase auth token
  Body: { billing_type: "PIX" | "BOLETO" | "CREDIT_CARD" }
  Response: { payment_url: string }

GET /api/subscriptions/current
  Auth: PocketBase auth token
  Response: { status, billing_type, next_due_date }

POST /api/webhooks/asaas
  Auth: Asaas webhook signature
  Body: Asaas event payload
```

### Flow

1. **Trial:** New users get N free generations (tracked via `generations_used`). No payment required.
2. **Paywall:** Generation endpoint checks `subscription_status` and `generations_used`. If trial exhausted and not `active`, return 402.
3. **Subscribe:** Frontend calls `POST /api/subscriptions`. Backend creates Asaas customer + subscription, returns payment link/QR code.
4. **Webhook:** Asaas sends payment events to `POST /api/webhooks/asaas`. Backend updates `subscription_status` on user record.
5. **Renewal:** Asaas handles recurring billing automatically.

### Webhook events

- `PAYMENT_CONFIRMED` -> set status `active`
- `PAYMENT_OVERDUE` -> set status `past_due`
- `SUBSCRIPTION_DELETED` -> set status `cancelled`

### Security

- Webhook validates Asaas signature
- Asaas API keys stored server-side only
- Subscription status checked server-side on every generation request
- `subscription_status` field is read-only from client (API rule: no client updates to this field)

### Acceptance criteria

- User can subscribe via PIX, boleto, or card
- Webhook correctly updates subscription status
- Generation blocked for expired trials without active subscription
- Subscription status visible to frontend via PocketBase SDK (read-only)

## Summary: Custom endpoints vs PocketBase SDK

| Operation | Method | Approach |
|-----------|--------|----------|
| Auth (Google OAuth) | SDK | `pb.collection('users').authWithOAuth2(...)` |
| Business CRUD | SDK | `pb.collection('businesses').create/update/delete(...)` |
| Post read/update/delete | SDK | `pb.collection('posts').getList/update/delete(...)` |
| Generate posts | `POST /api/businesses/{id}/posts:generate` | Custom (LLM calls) |
| Create subscription | `POST /api/subscriptions` | Custom (Asaas API) |
| Get subscription | `GET /api/subscriptions/current` | Custom (Asaas state) |
| Payment webhook | `POST /api/webhooks/asaas` | Custom (server-to-server) |

**3 custom endpoints + 1 read endpoint. Everything else is PocketBase SDK.**

## Out of scope (post-MVP)

- Multiple businesses per account
- Image generation or story templates
- Direct Instagram posting / scheduling
- Email notifications
- Admin dashboard
- Usage analytics
- Referral system
- Team/agency accounts
- Content calendar view
