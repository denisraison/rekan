package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "client_name"},
			&core.TextField{Name: "client_email"},
			&core.TextField{Name: "invite_token"},
			&core.SelectField{
				Name:      "invite_status",
				Values:    []string{"draft", "invited", "accepted", "active", "payment_failed", "cancelled"},
				MaxSelect: 1,
			},
			&core.DateField{Name: "invite_sent_at"},
			&core.TextField{Name: "subscription_id"},
			&core.DateField{Name: "terms_accepted_at"},
		)

		collection.AddIndex("idx_businesses_invite_token", true, "invite_token", "invite_token != ''")

		updateRule := `user = @request.auth.id
&& @request.body.user:isset = false
&& @request.body.invite_token:isset = false
&& @request.body.invite_status:isset = false
&& @request.body.invite_sent_at:isset = false
&& @request.body.terms_accepted_at:isset = false
&& @request.body.subscription_id:isset = false`
		collection.UpdateRule = &updateRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("client_name")
		collection.Fields.RemoveByName("client_email")
		collection.Fields.RemoveByName("invite_token")
		collection.Fields.RemoveByName("invite_status")
		collection.Fields.RemoveByName("invite_sent_at")
		collection.Fields.RemoveByName("subscription_id")
		collection.Fields.RemoveByName("terms_accepted_at")

		collection.RemoveIndex("idx_businesses_invite_token")

		updateRule := "user = @request.auth.id && @request.body.user:isset = false"
		collection.UpdateRule = &updateRule

		return app.Save(collection)
	})
}
