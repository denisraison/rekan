package migrations

import (
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/pocketbase/pocketbase/core"
)

// Update posts access rules now that businesses no longer have a user field.
// Any authenticated user can list/view/update/delete posts.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return err
		}

		authed := `@request.auth.id != ""`
		updateRule := `@request.auth.id != ""` +
			" && @request.body.business:isset = false" +
			" && @request.body.batch_id:isset = false" +
			" && @request.body.role:isset = false" +
			" && @request.body.hook:isset = false"

		collection.ListRule = &authed
		collection.ViewRule = &authed
		collection.UpdateRule = &updateRule
		collection.DeleteRule = &authed

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return nil
		}

		old := "business.user = @request.auth.id"
		updateRule := "business.user = @request.auth.id" +
			" && @request.body.business:isset = false" +
			" && @request.body.batch_id:isset = false" +
			" && @request.body.role:isset = false" +
			" && @request.body.hook:isset = false"

		collection.ListRule = &old
		collection.ViewRule = &old
		collection.UpdateRule = &updateRule
		collection.DeleteRule = &old

		return app.Save(collection)
	})
}
