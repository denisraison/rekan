package handlers

import (
	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/pocketbase/pocketbase/core"
)

type Deps struct {
	App          core.App
	Asaas        *asaas.Client // nil when ASAAS_API_KEY is not set
	WebhookToken string
}
