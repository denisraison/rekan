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
			&core.DateField{Name: "terms_accepted_at"},
			&core.TextField{Name: "terms_accepted_text"},
			&core.TextField{Name: "authorization_id"},
			&core.TextField{Name: "customer_id"},
			&core.SelectField{
				Name:      "tier",
				Values:    []string{"basico", "parceiro", "profissional"},
				MaxSelect: 1,
			},
			&core.SelectField{
				Name:      "commitment",
				Values:    []string{"mensal", "trimestral"},
				MaxSelect: 1,
			},
			&core.DateField{Name: "next_charge_date"},
			&core.BoolField{Name: "charge_pending"},
			&core.TextField{Name: "qr_payload"},
		)

		collection.AddIndex("idx_businesses_invite_token", true, "invite_token", "invite_token != ''")
		collection.AddIndex("idx_businesses_authorization_id", true, "authorization_id", "authorization_id != ''")

		updateRule := `user = @request.auth.id
&& @request.body.user:isset = false
&& @request.body.invite_token:isset = false
&& @request.body.invite_status:isset = false
&& @request.body.invite_sent_at:isset = false
&& @request.body.terms_accepted_at:isset = false
&& @request.body.terms_accepted_text:isset = false
&& @request.body.authorization_id:isset = false
&& @request.body.customer_id:isset = false
&& @request.body.tier:isset = false
&& @request.body.commitment:isset = false
&& @request.body.next_charge_date:isset = false
&& @request.body.charge_pending:isset = false
&& @request.body.qr_payload:isset = false`
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
		collection.Fields.RemoveByName("terms_accepted_at")
		collection.Fields.RemoveByName("terms_accepted_text")
		collection.Fields.RemoveByName("authorization_id")
		collection.Fields.RemoveByName("customer_id")
		collection.Fields.RemoveByName("tier")
		collection.Fields.RemoveByName("commitment")
		collection.Fields.RemoveByName("next_charge_date")
		collection.Fields.RemoveByName("charge_pending")
		collection.Fields.RemoveByName("qr_payload")

		collection.RemoveIndex("idx_businesses_invite_token")
		collection.RemoveIndex("idx_businesses_authorization_id")

		updateRule := "user = @request.auth.id && @request.body.user:isset = false"
		collection.UpdateRule = &updateRule

		return app.Save(collection)
	})
}
