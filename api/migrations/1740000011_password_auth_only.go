package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		// Switch to password auth, disable OAuth2
		collection.PasswordAuth.Enabled = true
		collection.OAuth2.Enabled = false
		collection.OAuth2.Providers = nil

		// Block self-registration: only PocketBase admin can create users
		collection.CreateRule = nil

		return app.Save(collection)
	}, func(app core.App) error {
		// Revert: re-enable OAuth2, disable password auth
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		collection.PasswordAuth.Enabled = false
		collection.OAuth2.Enabled = true
		empty := ""
		collection.CreateRule = &empty

		return app.Save(collection)
	})
}
