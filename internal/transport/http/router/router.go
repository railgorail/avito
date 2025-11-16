package router

import (
	"log/slog"
	"railgorail/avito/internal/config"
	"railgorail/avito/internal/transport/http/handlers"
	"railgorail/avito/internal/transport/http/handlers/pr"
	"railgorail/avito/internal/transport/http/handlers/stats"
	"railgorail/avito/internal/transport/http/handlers/team"
	"railgorail/avito/internal/transport/http/handlers/user"
	mw "railgorail/avito/internal/transport/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger, cfg *config.Config,
	teamHandler *team.TeamHandler,
	userHandler *user.UserHandler,
	prHandler *pr.PrHandler,
	statsHandler *stats.StatsHandler,
) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mw.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	log.Info("starting http server", slog.String("address", cfg.HTTPServer.Address))

	// Health check
	router.Get("/health", handlers.Healthcheck())

	// Team routes
	router.Route("/team", func(r chi.Router) {
		r.Post("/add", teamHandler.Add)
		r.Get("/get", teamHandler.Get)
	})

	// User routes
	router.Route("/users", func(r chi.Router) {
		r.Get("/getReview", userHandler.GetReview)
		r.Post("/setIsActive", userHandler.SetIsActive)
	})

	// Pull Request routes
	router.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", prHandler.Create)
		r.Post("/merge", prHandler.Merge)
		r.Post("/reassign", prHandler.Reassign)
	})

	// Stats routes
	router.Get("/stats", statsHandler.GetStatistics)

	return router
}
