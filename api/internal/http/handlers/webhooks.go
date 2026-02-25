package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/pricing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func AsaasWebhook(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WebhookToken != "" {
			token := e.Request.Header.Get("asaas-access-token")
			if token != deps.WebhookToken {
				return e.JSON(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
			}
		}

		var payload struct {
			Event   string `json:"event"`
			Payment *struct {
				ID                                string `json:"id"`
				Status                            string `json:"status"`
				PixAutomaticAuthorizationId        string `json:"pixAutomaticAuthorizationId"`
			} `json:"payment"`
			PixAutomaticAuthorization *struct {
				ID string `json:"id"`
			} `json:"pixAutomaticAuthorization"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&payload); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "invalid payload"})
		}

		// Extract authorization ID from the appropriate field
		var authorizationID string
		if payload.Payment != nil && payload.Payment.PixAutomaticAuthorizationId != "" {
			authorizationID = payload.Payment.PixAutomaticAuthorizationId
		} else if payload.PixAutomaticAuthorization != nil {
			authorizationID = payload.PixAutomaticAuthorization.ID
		}

		if authorizationID == "" {
			return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
		}

		businesses, err := e.App.FindAllRecords(domain.CollBusinesses, dbx.HashExp{"authorization_id": authorizationID})
		if err != nil || len(businesses) == 0 {
			return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
		}

		business := businesses[0]

		status := business.GetString("invite_status")

		switch payload.Event {
		case domain.EventPixAuthActivated:
			// Idempotent: skip if already active (duplicate webhook)
			if status == domain.InviteStatusActive {
				return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
			}
			business.Set("invite_status", domain.InviteStatusActive)
			commitment := pricing.Commitment(business.GetString("commitment"))
			months := pricing.Months[commitment]
			nextDate := time.Now().AddDate(0, months, 0).Format("2006-01-02 00:00:00.000Z")
			business.Set("next_charge_date", nextDate)

		case domain.EventPixAuthRefused,
			domain.EventPixAuthCancelled:
			business.Set("invite_status", domain.InviteStatusCancelled)

		case domain.EventPixAuthExpired:
			business.Set("invite_status", domain.InviteStatusPaymentFailed)

		case domain.EventPaymentConfirmed:
			// Idempotent: only advance next_charge_date if a charge was pending.
			// Duplicate webhooks see charge_pending=false and become no-ops.
			if status != domain.InviteStatusActive || !business.GetBool("charge_pending") {
				return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
			}
			business.Set("charge_pending", false)
			commitment := pricing.Commitment(business.GetString("commitment"))
			months := pricing.Months[commitment]
			nextDate := time.Now().AddDate(0, months, 0).Format("2006-01-02 00:00:00.000Z")
			business.Set("next_charge_date", nextDate)

		case domain.EventPixPaymentRefused,
			domain.EventPixPaymentCancelled:
			// Idempotent: only process if a charge was actually pending
			if !business.GetBool("charge_pending") {
				return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
			}
			business.Set("charge_pending", false)
			business.Set("invite_status", domain.InviteStatusPaymentFailed)

		default:
			return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
		}

		if err := e.App.Save(business); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
	}
}
