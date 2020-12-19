package rides_test

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

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
			LogLevel:      logger.Silent,
			Colorful:      false,
		},
	)
)

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{Logger: gormLogger})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&rides.Ride{})
	gormDB.AutoMigrate(&rides.RideState{})
	return gormDB
}

func TestQueueACustomer(t *testing.T) {
	t.Run("Create ride state with a new user in queue for the first time", func(t *testing.T) {
		db := testDB(t.Name())
		ride := &rides.Ride{Name: "t1", Capacity: 20, RideTime: 10 * time.Minute}
		db.Create(&ride)
		dao := rides.DAO{DB: db}

		rideState, err := dao.QueueACustomer(ride)

		assert.NilError(t, err)
		assert.Equal(t, ride.ID, rideState.RideID)
		assert.Equal(t, uint(1), rideState.CustomersInQueue)
		assert.Equal(t, time.Duration(0), rideState.EstimatedWaiting)
		assert.Assert(t, rideState.CreatedAt.Before(time.Now()))
		assert.Assert(t, rideState.UpdatedAt.Before(time.Now()))
	})

	t.Run("Update ride state with a new user in queue", func(t *testing.T) {
		db := testDB(t.Name())
		ride := &rides.Ride{Name: "t1", Capacity: 20, RideTime: 10 * time.Minute}
		db.Create(&ride)
		currentRideState := &rides.RideState{RideID: ride.ID, CustomersInQueue: 53}
		db.Create(&currentRideState)
		dao := rides.DAO{DB: db}

		newRideState, err := dao.QueueACustomer(ride)

		assert.NilError(t, err)
		assert.Equal(t, ride.ID, newRideState.RideID)
		assert.Equal(t, uint(54), newRideState.CustomersInQueue)
		// 20 capacity with 54 people means new person coming in will need to wait for 2 batch before their chance
		assert.Equal(t, 20*time.Minute, newRideState.EstimatedWaiting)
		assert.DeepEqual(t, currentRideState.CreatedAt, newRideState.CreatedAt)
		assert.Assert(t, newRideState.UpdatedAt.After(currentRideState.UpdatedAt))
	})
}

func TestUnQueueACustomer(t *testing.T) {
	t.Run("Errors when ride state doesn't exist", func(t *testing.T) {
		db := testDB(t.Name())
		ride := &rides.Ride{Name: "t1", Capacity: 20, RideTime: 10 * time.Minute}
		db.Create(&ride)
		dao := rides.DAO{DB: db}

		_, err := dao.UnQueueACustomers(ride)

		assert.Assert(t, errors.Is(err, rides.ErrRideHasNoQueue))
	})

	t.Run("Remove a customer from queue and recalculate waiting time", func(t *testing.T) {
		db := testDB(t.Name())
		ride := &rides.Ride{Name: "t1", Capacity: 20, RideTime: 10 * time.Minute}
		db.Create(&ride)
		currentRideState := &rides.RideState{RideID: ride.ID, CustomersInQueue: 60, EstimatedWaiting: 30 * time.Minute}
		db.Create(&currentRideState)
		dao := rides.DAO{DB: db}

		newRideState, err := dao.UnQueueACustomers(ride)

		assert.NilError(t, err)
		assert.Equal(t, ride.ID, newRideState.RideID)
		assert.Equal(t, uint(59), newRideState.CustomersInQueue)
		// 20 capacity with 59 people means new person coming in will need to wait for 2 batch before their chance
		assert.Equal(t, 20*time.Minute, newRideState.EstimatedWaiting)
		assert.DeepEqual(t, currentRideState.CreatedAt, newRideState.CreatedAt)
		assert.Assert(t, newRideState.UpdatedAt.After(currentRideState.UpdatedAt))
	})
}
