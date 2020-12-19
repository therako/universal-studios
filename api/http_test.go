package api_test

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/rides"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testConfig = api.Config{HTTPPort: 8081}
	gormLogger = logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		},
	)
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{
		Logger: gormLogger,
	})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&rides.Ride{})
	gormDB.AutoMigrate(&customers.Customer{})
	gormDB.AutoMigrate(&events.Event{})
	return gormDB
}
