package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// messages and posts were created without AutodateField for "created"/"updated",
// so PocketBase rejects sort requests on those fields.
func init() {
	m.Register(func(app core.App) error {
		for _, name := range []string{"messages", "posts"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}

			if collection.Fields.GetByName("created") == nil {
				collection.Fields.Add(&core.AutodateField{
					Name:     "created",
					OnCreate: true,
					OnUpdate: false,
					System:   true,
				})
			}

			if collection.Fields.GetByName("updated") == nil {
				collection.Fields.Add(&core.AutodateField{
					Name:     "updated",
					OnCreate: true,
					OnUpdate: true,
					System:   true,
				})
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{"messages", "posts"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return nil
			}

			changed := false
			if collection.Fields.GetByName("created") != nil {
				collection.Fields.RemoveByName("created")
				changed = true
			}
			if collection.Fields.GetByName("updated") != nil {
				collection.Fields.RemoveByName("updated")
				changed = true
			}

			if changed {
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
