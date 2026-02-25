package billing

import (
	"context"
	"time"

	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/pricing"
	"github.com/pocketbase/pocketbase/core"
)

// CreatePendingCharges finds active businesses whose next_charge_date is within
// 7 days and creates a Pix Automatico charge for each one via Asaas.
func CreatePendingCharges(app core.App, asaasClient *asaas.Client) {
	if asaasClient == nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, 7).Format("2006-01-02 00:00:00.000Z")

	businesses, err := app.FindRecordsByFilter(
		domain.CollBusinesses,
		"invite_status = 'active' && charge_pending = false && next_charge_date <= {:cutoff} && next_charge_date != ''",
		"",
		0, // no limit
		0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		app.Logger().Error("billing: query businesses", "error", err)
		return
	}

	ctx := context.Background()

	for _, biz := range businesses {
		tier := pricing.Tier(biz.GetString("tier"))
		commitment := pricing.Commitment(biz.GetString("commitment"))
		price, ok := pricing.Price(tier, commitment)
		if !ok {
			app.Logger().Error("billing: invalid tier/commitment", "business", biz.Id, "tier", string(tier), "commitment", string(commitment))
			continue
		}

		dueDate := biz.GetDateTime("next_charge_date").Time().Format("2006-01-02")

		// Set charge_pending BEFORE calling Asaas. If the process crashes
		// after Asaas succeeds but before this save, the next cron run
		// would create a duplicate charge. This order means a crash between
		// save and API call leaves charge_pending=true with no charge,
		// which the PAYMENT_CONFIRMED webhook won't clear (no payment to
		// confirm). That's a stuck flag, but it's recoverable manually
		// and far less harmful than a double charge.
		biz.Set("charge_pending", true)
		if err := app.Save(biz); err != nil {
			app.Logger().Error("billing: save charge_pending", "error", err, "business", biz.Id)
			continue
		}

		// Include due date in ExternalReference so each billing cycle has
		// a unique reference. Helps with reconciliation and prevents Asaas
		// from silently accepting duplicates for the same period.
		_, err := asaasClient.CreateCharge(ctx, asaas.CreateChargeReq{
			Customer:                    biz.GetString("customer_id"),
			BillingType:                 domain.BillingTypePIX,
			Value:                       price,
			DueDate:                     dueDate,
			Description:                 "Rekan - " + string(tier),
			ExternalReference:           biz.Id + "_" + dueDate,
			PixAutomaticAuthorizationId: biz.GetString("authorization_id"),
		})
		if err != nil {
			app.Logger().Error("billing: create charge", "error", err, "business", biz.Id)
			// Rollback: clear charge_pending so the next cron run retries
			biz.Set("charge_pending", false)
			if saveErr := app.Save(biz); saveErr != nil {
				app.Logger().Error("billing: rollback charge_pending", "error", saveErr, "business", biz.Id)
			}
			continue
		}
	}
}
