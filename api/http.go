package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/rides"
	"gorm.io/gorm"
)

// New Returns a HTTP router with all studios routes
func New(ctx context.Context, config Config, gormDB *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	r := Rides{DAO: rides.DAO{DB: gormDB}}
	router.GET("/ride", r.List)
	router.POST("/ride/add", r.Add)

	c := Customers{DAO: customers.DAO{DB: gormDB}}
	router.GET("/customer", c.List)
	router.POST("/customer/enter", c.Enter)
	router.POST("/customer/exit", c.Exit)

	return router
}
