package httpserver

import (
	"context"

	campaignhttp "project-srv/internal/campaign/delivery/http"
	campaignrepo "project-srv/internal/campaign/repository/postgre"
	campaignuc "project-srv/internal/campaign/usecase"
	crisishttp "project-srv/internal/crisis/delivery/http"
	crisisrepo "project-srv/internal/crisis/repository/postgre"
	crisisuc "project-srv/internal/crisis/usecase"
	"project-srv/internal/model"
	projecthttp "project-srv/internal/project/delivery/http"
	projectrepo "project-srv/internal/project/repository/postgre"
	projectuc "project-srv/internal/project/usecase"

	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/middleware"

	// Import this to execute the init function in docs.go which setups the Swagger docs.
	// Uncomment after running: make swagger
	// _ "project-srv/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	mw := middleware.New(middleware.Config{
		JWTManager:       auth.NewManager(srv.jwtSecretKey),
		CookieName:       srv.cookieConfig.Name,
		ProductionDomain: srv.cookieConfig.Domain,
		InternalKey:      srv.internalKey,
		IsProduction:     srv.environment == string(model.EnvironmentProduction),
	})

	srv.registerMiddlewares()
	srv.registerSystemRoutes()

	// Campaign module
	campaignRepo := campaignrepo.New(srv.postgresDB, srv.l)
	campaignUC := campaignuc.New(srv.l, campaignRepo)
	campaignHandler := campaignhttp.New(srv.l, campaignUC, srv.discord)

	// Project module
	projectRepo := projectrepo.New(srv.postgresDB, srv.l)
	projectUC := projectuc.New(srv.l, projectRepo, campaignUC)
	projectHandler := projecthttp.New(srv.l, projectUC, srv.discord)

	// Crisis Config module
	crisisRepo := crisisrepo.New(srv.postgresDB, srv.l)
	crisisUC := crisisuc.New(srv.l, crisisRepo, projectUC)
	crisisHandler := crisishttp.New(srv.l, crisisUC, srv.discord)

	// Map routes
	apiV1 := srv.gin.Group(model.APIV1Prefix)
	campaignHandler.RegisterRoutes(apiV1, mw)
	projectHandler.RegisterRoutes(apiV1, mw)
	crisisHandler.RegisterRoutes(apiV1, mw)

	return nil
}

func (srv HTTPServer) registerMiddlewares() {
	srv.gin.Use(middleware.Recovery(srv.l, srv.discord))

	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	srv.gin.Use(middleware.Tracing())

	// Log CORS mode for visibility
	ctx := context.Background()
	if srv.environment == string(model.EnvironmentProduction) {
		srv.l.Infof(ctx, "CORS mode: production (strict origins only)")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s (permissive - allows localhost and private subnets)", srv.environment)
	}

	// Add locale middleware to extract and set locale from request header
	srv.gin.Use(middleware.Locale())
}

func (srv HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)

	// Swagger UI and docs
	srv.gin.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("doc.json"), // Use relative path
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
}
