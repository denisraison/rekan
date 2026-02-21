package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func AsaasWebhook(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
	}
}
