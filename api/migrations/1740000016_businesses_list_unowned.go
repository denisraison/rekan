package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Drop the user field from businesses. Access is operator-only (no client portal),
// so user-scoped rules add no value. Any authenticated user can list/view/create/update/delete.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("user")

		authed := "@request.auth.id != \"\""
		collection.ListRule = &authed
		collection.ViewRule = &authed
		collection.CreateRule = &authed
		collection.UpdateRule = &authed
		collection.DeleteRule = &authed

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return nil
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		collection.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1})

		rule := "user = @request.auth.id"
		collection.ListRule = &rule
		collection.ViewRule = &rule
		createRule := `@request.auth.id != "" && @request.body.user = @request.auth.id`
		collection.CreateRule = &createRule
		updateRule := "user = @request.auth.id && @request.body.user:isset = false"
		collection.UpdateRule = &updateRule
		collection.DeleteRule = &rule

		return app.Save(collection)
	})
}
