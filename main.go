package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/rides"
)

func main() {
	fmt.Println("Welcome to Universal Studios")

	ctx := context.Background()
	cfg, err := api.GetConfig(ctx)
	if err != nil {
		log.Fatalln(ctx, err, "config-init-error")
	}

	viper.SetDefault("POSTGRES_PORT", 5432)
	dbDNS := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?statement_timeout=%d&connect_timeout=%d&sslmode=%s",
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_HOST"),
		viper.GetUint("POSTGRES_PORT"),
		viper.GetString("POSTGRES_DB"),
		2000,
		1,
		"disable",
	)
	gormDB, err := gorm.Open(postgres.Open(dbDNS), &gorm.Config{})
	if err != nil {
		log.Fatalln(ctx, err, "connecting-to-db")
	}

	gormDB.AutoMigrate(&rides.Ride{})
	gormDB.AutoMigrate(&customers.Customer{})
	gormDB.AutoMigrate(&events.Event{})

	gin.SetMode(gin.ReleaseMode)
	router := api.New(ctx, cfg, gormDB)
	router.Run(fmt.Sprintf(":%d", cfg.HTTPPort))
}
