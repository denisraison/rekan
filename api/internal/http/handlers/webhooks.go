package handlers

import (
	"encoding/json"
	"net/http"

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
				Subscription string `json:"subscription"`
			} `json:"payment"`
			Subscription *struct {
				ID string `json:"id"`
			} `json:"subscription"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&payload); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "invalid payload"})
		}

		var subscriptionID string
		if payload.Payment != nil {
			subscriptionID = payload.Payment.Subscription
		} else if payload.Subscription != nil {
			subscriptionID = payload.Subscription.ID
		}

		if subscriptionID == "" {
			return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
		}

		var newStatus string
		switch payload.Event {
		case "PAYMENT_CONFIRMED":
			newStatus = "active"
		case "PAYMENT_OVERDUE":
			newStatus = "past_due"
		case "SUBSCRIPTION_DELETED":
			newStatus = "cancelled"
		default:
			return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
		}

		users, err := e.App.FindAllRecords("users", dbx.HashExp{"subscription_id": subscriptionID})
		if err != nil || len(users) == 0 {
			return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
		}

		user := users[0]
		oldStatus := user.GetString("subscription_status")
		user.Set("subscription_status", newStatus)
		if err := e.App.Save(user); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		// First payment: upgrade subscription from R$19 to R$69.90
		if oldStatus == "trial" && newStatus == "active" && deps.Asaas != nil {
			if err := deps.Asaas.UpdateSubscription(e.Request.Context(), subscriptionID, monthlyPriceBRL); err != nil {
				e.App.Logger().Error("asaas update subscription price", "subscription", subscriptionID, "error", err)
			}
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "ok"})
	}
}
