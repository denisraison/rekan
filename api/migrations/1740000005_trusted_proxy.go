package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

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
