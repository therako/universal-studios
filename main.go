package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/rides"
)

func main() {
	fmt.Println("Welcome to Universal Studios")

	ctx := context.Background()
	cfg, err := api.GetConfig(ctx)
	if err != nil {
		log.Fatalln(ctx, err, "config-init-error")
	}

	gormDB, err := gorm.Open(sqlite.Open("ustudios.db"), &gorm.Config{})
	if err != nil {
		log.Fatalln(ctx, err, "connecting-to-db")
	}

	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&rides.Ride{})
	gormDB.AutoMigrate(&rides.RideState{})
	gormDB.AutoMigrate(&customers.Customer{})
	gormDB.AutoMigrate(&customers.CustomerState{})

	gin.SetMode(gin.ReleaseMode)
	router := api.New(ctx, cfg, gormDB)
	router.Run(fmt.Sprintf(":%d", cfg.HTTPPort))
}
