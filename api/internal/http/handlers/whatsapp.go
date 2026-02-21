package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// WhatsAppStatus returns the current WhatsApp connection status and QR code.
func WhatsAppStatus(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}
		return e.JSON(http.StatusOK, deps.WhatsApp.Status())
	}
}
