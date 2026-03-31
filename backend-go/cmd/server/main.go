package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	appbootstrap "my-robot-backend/internal/app"
	"my-robot-backend/internal/domain/digest"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/config"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/middleware"
	"my-robot-backend/internal/platform/tracing"
)

func main() {
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := digest.Migrate(); err != nil {
		log.Fatalf("Failed to run digest migrations: %v", err)
	}

	if err := airouter.EnsureLegacySummaryConfigMigrated(); err != nil {
		log.Printf("Warning: Failed to migrate legacy AI summary config: %v", err)
	}

	traceCfg := tracing.DefaultConfig()
	tp, err := tracing.InitTracerProvider(database.DB, traceCfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize tracing: %v", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Warning: Failed to shutdown tracer: %v", err)
			}
		}()
	}

	if config.AppConfig != nil && config.AppConfig.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.Use(otelgin.Middleware("rss-reader-backend"))
	if config.AppConfig != nil {
		r.Use(middleware.CORS(config.AppConfig))
	}
	r.Use(gin.Recovery())

	appbootstrap.SetupRoutes(r)
	runtime := appbootstrap.StartRuntime()
	appbootstrap.SetupGracefulShutdown(runtime)

	addr := fmt.Sprintf(":%s", config.AppConfig.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("Environment: %s", config.AppConfig.Server.Mode)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}
