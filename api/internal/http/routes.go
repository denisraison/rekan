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

	// Subscription resource
	rtr.POST("/api/subscriptions", handlers.CreateSubscription(deps)).Bind(auth)
	rtr.GET("/api/subscriptions/current", handlers.GetSubscription(deps)).Bind(auth)

	// Asaas webhook (server-to-server, no auth middleware)
	rtr.POST("/api/webhooks/asaas", handlers.AsaasWebhook(deps))
}
