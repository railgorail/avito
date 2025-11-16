package main

import (
	"log/slog"

	"railgorail/avito/internal/config"
	"railgorail/avito/internal/lib/logger"
	"railgorail/avito/internal/repo"
	"railgorail/avito/internal/server"
	"railgorail/avito/internal/service/pr"
	"railgorail/avito/internal/service/stats"
	"railgorail/avito/internal/service/team"
	"railgorail/avito/internal/service/user"
	"railgorail/avito/internal/storage"
	prhandler "railgorail/avito/internal/transport/http/handlers/pr"
	statshandler "railgorail/avito/internal/transport/http/handlers/stats"
	teamhandler "railgorail/avito/internal/transport/http/handlers/team"
	userhandler "railgorail/avito/internal/transport/http/handlers/user"
	"railgorail/avito/internal/transport/http/router"

	trm "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	// config and logger
	cfg := config.MustLoad()
	log := logger.New(cfg.Env)
	log.Info("starting service", slog.String("env", cfg.Env))

	// storage
	db, cleanup := storage.MustInit(cfg.Postgres.DatabaseURL, log)
	defer cleanup()

	// repo layer
	trManager := manager.Must(trm.NewDefaultFactory(db))

	teamRepo := repo.NewTeamRepo(db, trm.DefaultCtxGetter)
	userRepo := repo.NewUserRepo(db, trm.DefaultCtxGetter)
	prRepo := repo.NewPullRequestRepo(db, trm.DefaultCtxGetter, trManager)
	statsRepo := repo.NewStatisticsRepo(db)

	// service layer
	teamService := team.NewTeamService(trManager, teamRepo, userRepo, prRepo)
	userService := user.NewUserService(trManager, prRepo, userRepo, teamRepo)
	prService := pr.NewPullRequestService(trManager, prRepo, prRepo, userRepo)
	statsService := stats.NewStatsService(trManager, statsRepo)

	// transport layer
	teamHandler := teamhandler.NewTeamHandler(log, teamService)
	userHandler := userhandler.NewUserHandler(log, userService)
	prHandler := prhandler.NewPrHandler(log, prService)
	statsHandler := statshandler.NewStatsHandler(log, statsService)

	// http router
	router := router.New(log, cfg, teamHandler, userHandler, prHandler, statsHandler)

	// server
	srv := server.New(router, log, cfg)
	srv.Run(cfg.HTTPServer.ShutdownTimeout)

	log.Info("service stopped")
}
