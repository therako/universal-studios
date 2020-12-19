package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
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

	c := Customers{DAO: customers.DAO{DB: gormDB}, RideDAO: rides.DAO{DB: gormDB}, eventDAO: events.DAO{DB: gormDB}}
	router.GET("/customer", c.List)
	router.POST("/customer/enter", c.Enter)
	router.POST("/customer/exit", c.Exit)
	router.POST("/customer/queue", c.Queue)
	router.POST("/customer/unqueue", c.UnQueue)

	return router
}

func handleError(c *gin.Context, err error, errPrefix string) {
	var status int
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status = http.StatusNotFound
	} else {
		status = http.StatusInternalServerError
	}
	c.AbortWithStatusJSON(status, gin.H{"err": fmt.Sprintf("%s %s", errPrefix, err.Error())})
}
