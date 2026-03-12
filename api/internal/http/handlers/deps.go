package handlers

import (
	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/pocketbase/pocketbase/core"
)

type Deps struct {
	App                 core.App
	Asaas               *asaas.Client        // nil when ASAAS_API_KEY is not set
	WhatsApp            *whatsapp.Client     // nil when WhatsApp is not connected
	Transcribe          *transcribe.Client   // nil when GEMINI_API_KEY is not set
	WebhookToken        string
	AppURL              string
	Generate            content.GenerateFunc
	GenerateFromMessage content.GenerateFromMessageFunc
	ExtractFromAudio    content.ExtractFromAudioFunc // nil when GEMINI_API_KEY is not set
}
