package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// ExtractProfile handles POST /api/businesses/profile:extract.
// Accepts a multipart form with an audio file and business_type,
// transcribes the audio, extracts structured profile fields, and returns them.
// No database writes — the caller reviews and saves via the normal create/update flow.
func ExtractProfile(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.ExtractFromAudio == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "extração de perfil não configurada",
			})
		}

		if err := e.Request.ParseMultipartForm(10 << 20); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"message": "corpo da requisição inválido",
			})
		}

		businessType := e.Request.FormValue("business_type")

		file, header, err := e.Request.FormFile("audio")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"message": "arquivo de áudio obrigatório",
			})
		}
		defer func() { _ = file.Close() }()

		audioBytes, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("read audio: %w", err)
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "audio/webm"
		}

		profile, err := deps.ExtractFromAudio(e.Request.Context(), audioBytes, mimeType, businessType)
		if err != nil {
			e.App.Logger().Error("extract profile failed", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "não foi possível analisar o áudio",
			})
		}

		type serviceResponse struct {
			Name     string   `json:"name"`
			PriceBRL *float64 `json:"price_brl"`
		}

		services := make([]serviceResponse, len(profile.Services))
		for i, s := range profile.Services {
			services[i] = serviceResponse{Name: s.Name, PriceBRL: s.PriceBRL}
		}

		return e.JSON(http.StatusOK, map[string]any{
			"services":        services,
			"target_audience": profile.TargetAudience,
			"brand_vibe":      profile.BrandVibe,
			"quirks":          profile.Quirks,
		})
	}
}
