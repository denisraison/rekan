package handlers

import (
	"net/http"

	"github.com/denisraison/rekan/api/internal/terms"
	"github.com/pocketbase/pocketbase/core"
)

func Terms() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, terms.Clauses())
	}
}
