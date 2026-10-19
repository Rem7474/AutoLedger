package main

import (
	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/auth"
	appMiddleware "github.com/teslacost/teslacost/internal/middleware"
)

// registerIntegrationRoutes registers the only routes an API token opens (Home Assistant, scripts, assistants). A
// token has a scope: pushing an event needs "ingest", the Home Assistant views are open to both scopes, and the
// read API under /api/integrations/v1 is for "read" tokens only, so a token that can push cannot also read the costs.
func (h *apiHandlers) registerIntegrationRoutes(r chi.Router) {
	r.With(appMiddleware.RequireTokenScope(auth.ScopeIngest)).Post("/api/integrations/homeassistant/event", h.ha.HandleEvent)
	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.RequireTokenScope(auth.ScopeIngest, auth.ScopeRead))
		r.Get("/api/integrations/homeassistant/vehicles", h.ha.ListVehicles)
		r.Get("/api/integrations/homeassistant/vehicles/{vehicleId}", h.ha.GetVehicle)
		r.Get("/api/integrations/homeassistant/vehicles/{vehicleId}/metrics", h.ha.GetVehicleMetrics)
	})
	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.RequireTokenScope(auth.ScopeRead))
		r.Get("/api/integrations/v1/vehicles", h.integrationRead.ListVehicles)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}", h.integrationRead.GetVehicle)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/costs", h.integrationRead.GetCosts)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/data-gaps", h.integrationRead.GetDataGaps)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/trips", h.integrationRead.GetTrips)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/monthly-costs", h.integrationRead.GetMonthlyCosts)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/reminders", h.integrationRead.GetReminders)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/ownership", h.integrationRead.GetOwnership)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/energy-efficiency", h.integrationRead.GetEnergyEfficiency)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/expenses", h.integrationRead.GetExpenses)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/tags", h.integrationRead.GetTags)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/carpool", h.integrationRead.GetCarpool)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/charging-places", h.integrationRead.GetChargingPlaces)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/trip-groups", h.integrationRead.GetTripGroups)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/tires", h.integrationRead.GetTires)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/odometer", h.integrationRead.GetOdometer)
		r.Get("/api/integrations/v1/vehicles/{vehicleId}/data-sources", h.integrationRead.GetDataSources)
		r.Get("/api/integrations/v1/fleet", h.integrationRead.GetFleetSummary)
	})
}
