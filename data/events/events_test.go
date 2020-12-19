package events_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/rides"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gotest.tools/v3/assert"
)

var (
	gormLogger = logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)
)

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{Logger: gormLogger})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&customers.Customer{})
	gormDB.AutoMigrate(&customers.CustomerState{})
	gormDB.AutoMigrate(&rides.Ride{})
	gormDB.AutoMigrate(&rides.RideState{})
	return gormDB
}
func TestCustomerQueued(t *testing.T) {
	t.Run("Expect update stated for adding a customer to a ride queue", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{}
		db.Create(&customer)
		ride := &rides.Ride{Name: "t1", Capacity: 20, RideTime: 10 * time.Minute}
		db.Create(&ride)
		rideState := &rides.RideState{RideID: ride.ID, CustomersInQueue: 60, EstimatedWaiting: 30 * time.Minute}
		db.Create(&rideState)

		event := events.CustomerQueued{
			BaseEvent: events.BaseEvent{At: time.Now(), Name: "CustomerQueued"},
			Customer:  customer,
			Ride:      ride,
		}
		err := event.Apply(rides.DAO{DB: db}, customers.DAO{DB: db})

		assert.NilError(t, err)
	})

	t.Run("Expect no change in ride state when customer state update has errors", func(t *testing.T) {
		db := testDB(t.Name())
		ride := &rides.Ride{Name: "t1", Capacity: 20, RideTime: 10 * time.Minute}
		db.Create(&ride)
		rideState := &rides.RideState{RideID: ride.ID, CustomersInQueue: 60, EstimatedWaiting: 30 * time.Minute}
		db.Create(&rideState)

		event := events.CustomerQueued{
			BaseEvent: events.BaseEvent{At: time.Now(), Name: "CustomerQueued"},
			Customer:  &customers.Customer{},
			Ride:      ride,
		}
		err := event.Apply(rides.DAO{DB: db}, customers.DAO{DB: db})

		assert.Error(t, err, "Customer not found")

		rideStateInDB := &rides.RideState{}
		db.Find(rideStateInDB, ride.ID)
		assert.Equal(t, rideState.CustomersInQueue, rideStateInDB.CustomersInQueue)
		assert.Equal(t, rideState.EstimatedWaiting, rideStateInDB.EstimatedWaiting)
	})
}
