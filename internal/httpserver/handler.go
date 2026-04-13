package httpserver

import (
	"context"
	"net/http"

	campaignhttp "project-srv/internal/campaign/delivery/http"
	campaignrepo "project-srv/internal/campaign/repository/postgre"
	campaignuc "project-srv/internal/campaign/usecase"
	crisishttp "project-srv/internal/crisis/delivery/http"
	crisisrepo "project-srv/internal/crisis/repository/postgre"
	crisisuc "project-srv/internal/crisis/usecase"
	"project-srv/internal/domain"
	"project-srv/internal/model"
	"project-srv/internal/project"
	projecthttp "project-srv/internal/project/delivery/http"
	projectkafkaproducer "project-srv/internal/project/delivery/kafka/producer"
	projectrepo "project-srv/internal/project/repository/postgre"
	projectuc "project-srv/internal/project/usecase"
	ingestsrv "project-srv/pkg/microservice/ingest"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/middleware"
	"github.com/smap-hcmut/shared-libs/go/paginator"

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
	domainRepo := domain.NewRepository(srv.mainRedisClient, srv.l)
	ingestSrv := ingestsrv.New(srv.l, srv.microservice.Ingest.BaseURL, srv.microservice.Ingest.TimeoutMS, srv.internalKey)
	lifecyclePublisher := projectkafkaproducer.New(srv.l, srv.kafkaProducer)
	projectUC := projectuc.New(srv.l, projectRepo, domainRepo, campaignUC, ingestSrv, lifecyclePublisher)
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

	internalAPI := apiV1.Group("/internal")
	projectHandler.RegisterInternalRoutes(internalAPI, mw)

	// Internal campaign detail — returns flat {id, name, project_ids} for service-to-service calls.
	// knowledge-srv needs project_ids for a campaign, which requires joining campaign + project data.
	internalCampaigns := internalAPI.Group("/campaigns")
	internalCampaigns.Use(mw.InternalAuth())
	internalCampaigns.GET("/:id", func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		// Get campaign details
		campOut, err := campaignUC.Detail(ctx, id)
		if err != nil {
			srv.l.Errorf(ctx, "internal.campaigns.Detail: campaign_id=%s err=%v", id, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
			return
		}

		// Get projects belonging to this campaign (up to 1000)
		projOut, err := projectUC.List(ctx, project.ListInput{
			CampaignID: id,
			Paginator:  paginator.PaginateQuery{Page: 1, Limit: 1000},
		})
		if err != nil {
			srv.l.Errorf(ctx, "internal.campaigns.ListProjects: campaign_id=%s err=%v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list projects"})
			return
		}

		projectIDs := make([]string, len(projOut.Projects))
		for i, p := range projOut.Projects {
			projectIDs[i] = p.ID
		}

		// Return flat response that knowledge-srv can unmarshal directly into Campaign{ID, Name, ProjectIDs}
		c.JSON(http.StatusOK, gin.H{
			"id":          campOut.Campaign.ID,
			"name":        campOut.Campaign.Name,
			"project_ids": projectIDs,
		})
	})

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
