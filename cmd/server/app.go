package main

import (
	"net/netip"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teslacost/teslacost/internal/config"
	"github.com/teslacost/teslacost/internal/crypto"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/handlers"
	"github.com/teslacost/teslacost/internal/services"
)

// newApplication builds the services and the router the server and the end-to-end tests run.
func newApplication(cfg *config.Config, trustedProxies []netip.Prefix, pool *pgxpool.Pool, repo *database.Repository, encryptor *crypto.Encryptor) (chi.Router, appServices, error) {
	notificationService := services.NewNotificationService(repo)
	syncService := services.NewSyncService(repo, encryptor)
	syncService.SetNotificationService(notificationService)
	tcoService := services.NewTCOService(pool, cfg.ReportingTimezone)
	svc := appServices{
		sync:         syncService,
		tireWear:     services.NewTireWearService(repo),
		tco:          tcoService,
		energyStats:  services.NewEnergyStatsService(pool, cfg.ReportingTimezone),
		comparison:   services.NewComparisonService(tcoService),
		carpool:      services.NewCarpoolService(pool, repo),
		notification: notificationService,
	}

	r := newRouter(cfg, trustedProxies)
	registerSystemRoutes(r, pool.Ping)

	api, err := newAPIHandlers(cfg, repo, encryptor, svc)
	if err != nil {
		return nil, appServices{}, err
	}
	registerAPIRoutes(r, cfg.JWTSecret, repo, repo, handlers.Idempotency(repo), api)
	registerSPA(r)
	return r, svc, nil
}
