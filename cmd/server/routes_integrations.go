package main

import (
	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/auth"
	appMiddleware "github.com/teslacost/teslacost/internal/middleware"
)

// registerIntegrationRoutes registers the only routes an API token opens (Home Assistant, scripts). A token has a
// scope: pushing an event needs "ingest", reading is open to both scopes.
func (h *apiHandlers) registerIntegrationRoutes(r chi.Router) {
	r.With(appMiddleware.RequireTokenScope(auth.ScopeIngest)).Post("/api/integrations/homeassistant/event", h.ha.HandleEvent)
	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.RequireTokenScope(auth.ScopeIngest, auth.ScopeRead))
		r.Get("/api/integrations/homeassistant/vehicles", h.ha.ListVehicles)
		r.Get("/api/integrations/homeassistant/vehicles/{vehicleId}", h.ha.GetVehicle)
		r.Get("/api/integrations/homeassistant/vehicles/{vehicleId}/metrics", h.ha.GetVehicleMetrics)
	})
}
