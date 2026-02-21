package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

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
