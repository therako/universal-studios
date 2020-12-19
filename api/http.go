package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/api/rides"
	"gorm.io/gorm"
)

// New Returns a HTTP router with all studios routes
func New(ctx context.Context, config Config, gormDB *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	r := rides.Rides{DB: gormDB}
	router.GET("/rides", r.List)
	router.POST("/rides/add", r.Add)

	return router
}
