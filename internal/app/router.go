package app

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"

	"github.com/DK-0x3/TickTackBackend/internal/config"
	"github.com/DK-0x3/TickTackBackend/internal/modules/health"
	"github.com/DK-0x3/TickTackBackend/internal/modules/tasks"
)

func newRouter(
	environment config.Environment,
	database *pgxpool.Pool,
	taskService tasks.TaskService,
	developmentUserID uuid.UUID,
) *gin.Engine {
	if environment == config.EnvironmentProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	health.RegisterRoutes(router, database)
	tasks.RegisterRoutes(router, taskService, developmentUserID)

	return router
}
