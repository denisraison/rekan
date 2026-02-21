package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func CreateSubscription(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
	}
}

func GetSubscription(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
	}
}
