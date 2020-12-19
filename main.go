package main

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/api/rides"
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

	router := api.New(ctx, cfg, gormDB)
	router.Run(fmt.Sprintf(":%d", cfg.HTTPPort))
}
