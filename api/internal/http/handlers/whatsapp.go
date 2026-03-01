package handlers

import (
	"encoding/json"
	"fmt"
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

// WhatsAppStatusStream streams WhatsApp status changes as SSE.
// Sends the current status immediately, then pushes on every state change.
func WhatsAppStatusStream(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}

		w := e.Response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"message": "streaming não suportado",
			})
		}

		send := func(v any) error {
			data, err := json.Marshal(v)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return err
		}

		// Send current status immediately.
		if err := send(deps.WhatsApp.Status()); err != nil {
			return nil
		}

		ch := deps.WhatsApp.Subscribe()
		defer deps.WhatsApp.Unsubscribe(ch)

		ctx := e.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return nil
			case s, ok := <-ch:
				if !ok {
					return nil
				}
				if err := send(s); err != nil {
					return nil
				}
			}
		}
	}
}
