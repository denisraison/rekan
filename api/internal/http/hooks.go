package http

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHooks(app *pocketbase.PocketBase) {
	app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if e.IsNewRecord {
			e.Record.Set("subscription_status", "trial")
			e.Record.Set("generations_used", 0)
			return e.App.Save(e.Record)
		}
		return nil
	})
}
