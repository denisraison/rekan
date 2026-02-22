package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	asaasclient "github.com/denisraison/rekan/api/internal/asaas"
	"github.com/pocketbase/pocketbase/core"
)

const (
	monthlyPriceBRL    = 69.90
	firstMonthPriceBRL = 19.00
)

func CreateSubscription(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.Asaas == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "pagamentos não configurados.",
			})
		}

		status := e.Auth.GetString("subscription_status")
		if status == "active" {
			return e.JSON(http.StatusConflict, map[string]string{
				"message": "assinatura já está ativa.",
			})
		}

		var body struct {
			BillingType string `json:"billing_type"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || body.BillingType == "" {
			body.BillingType = "PIX"
		}

		name := e.Auth.GetString("name")
		email := e.Auth.GetString("email")

		customer, err := deps.Asaas.CreateCustomer(e.Request.Context(), name, email)
		if err != nil {
			e.App.Logger().Error("asaas create customer", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "erro ao criar conta de pagamento. Tente novamente.",
			})
		}

		sub, err := deps.Asaas.CreateSubscription(e.Request.Context(), asaasclient.CreateSubscriptionReq{
			Customer:    customer.ID,
			BillingType: body.BillingType,
			Value:       firstMonthPriceBRL,
			NextDueDate: time.Now().Format("2006-01-02"),
			Cycle:       "MONTHLY",
			Description: "Rekan - Primeiro Mês",
		})
		if err != nil {
			e.App.Logger().Error("asaas create subscription", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "erro ao criar assinatura. Tente novamente.",
			})
		}

		e.Auth.Set("subscription_id", sub.ID)
		if err := e.App.Save(e.Auth); err != nil {
			e.App.Logger().Error("save subscription_id", "error", err)
		}

		return e.JSON(http.StatusOK, map[string]string{
			"payment_url": sub.PaymentLink,
		})
	}
}

func GetSubscription(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]any{
			"status":           e.Auth.GetString("subscription_status"),
			"subscription_id":  e.Auth.GetString("subscription_id"),
			"generations_used": e.Auth.GetInt("generations_used"),
		})
	}
}
