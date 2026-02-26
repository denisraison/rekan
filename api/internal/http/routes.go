package http

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/denisraison/rekan/api/internal/http/handlers"
)

func RegisterRoutes(rtr *router.Router[*core.RequestEvent], deps handlers.Deps) {
	auth := apis.RequireAuth()

	// Custom method on the business resource (Google API style: :verb suffix)
	rtr.POST("/api/businesses/{id}/posts:generate", handlers.GeneratePosts(deps)).Bind(auth)

	// Terms (public, no auth)
	rtr.GET("/api/terms", handlers.Terms())

	// Invite flow (public, no auth)
	rtr.GET("/api/invites/{token}", handlers.InviteGet(deps))
	rtr.POST("/api/invites/{token}/accept", handlers.InviteAccept(deps))

	// Invite management (auth required)
	rtr.POST("/api/businesses/{id}/invites:send", handlers.InviteSend(deps)).Bind(auth)
	rtr.POST("/api/businesses/{id}/authorization:cancel", handlers.AuthorizationCancel(deps)).Bind(auth)

	// Operator tool (single-post generation from WhatsApp message)
	rtr.POST("/api/businesses/{id}/posts:generateFromMessage", handlers.OperatorGenerate(deps)).Bind(auth)

	// WhatsApp
	rtr.GET("/api/whatsapp/status", handlers.WhatsAppStatus(deps)).Bind(auth)
	rtr.POST("/api/messages:send", handlers.SendMessage(deps)).Bind(auth)

	// Asaas webhook (server-to-server, no auth middleware)
	rtr.POST("/api/webhooks/asaas", handlers.AsaasWebhook(deps))
}
