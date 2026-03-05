package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// DescribeMedia accepts an image or video upload and returns a text description via Gemini.
func DescribeMedia(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.Transcribe == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "Gemini não configurado",
			})
		}

		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Arquivo é obrigatório"})
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Erro ao ler arquivo"})
		}

		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		var description string
		if strings.HasPrefix(contentType, "video/") {
			description, err = deps.Transcribe.DescribeVideo(e.Request.Context(), data, contentType, "")
		} else {
			description, err = deps.Transcribe.DescribeImage(e.Request.Context(), data, contentType, "")
		}
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao descrever mídia"})
		}

		return e.JSON(http.StatusOK, map[string]string{"description": description})
	}
}
