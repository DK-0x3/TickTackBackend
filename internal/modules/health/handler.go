package health

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type response struct {
	Status string `json:"status"`
}

type Database interface {
	Ping(context.Context) error
}

type Handler struct {
	database Database
}

func RegisterRoutes(router gin.IRoutes, database Database) {
	handler := Handler{database: database}

	router.GET("/health", check)
	router.GET("/ready", handler.ready)
}

func check(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, response{Status: "ok"})
}

func (handler Handler) ready(ctx *gin.Context) {
	if err := handler.database.Ping(ctx.Request.Context()); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, response{Status: "unavailable"})
		return
	}

	ctx.JSON(http.StatusOK, response{Status: "ready"})
}
