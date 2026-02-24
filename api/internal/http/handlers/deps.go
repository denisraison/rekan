package handlers

import (
	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
)

type Deps struct {
	App                 core.App
	Asaas               *asaas.Client    // nil when ASAAS_API_KEY is not set
	WhatsApp            *whatsapp.Client // nil when WhatsApp is not connected
	WebhookToken        string
	AppURL              string
	Generate            eval.GenerateFunc
	GenerateFromMessage eval.GenerateFromMessageFunc
}
