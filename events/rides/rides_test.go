package rides_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/models"
	ridesData "gitlab.com/therako/universal-studios/data/rides"
	"gitlab.com/therako/universal-studios/events/rides"
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

func init() {
	rides.Cache.Clear()
}

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{Logger: gormLogger})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&ridesData.Ride{})
	gormDB.AutoMigrate(&events.Event{})
	return gormDB
}
func TestE2ERideWaitEstimates(t *testing.T) {
	db := testDB(t.Name())
	ts := time.Now()
	rides.Clock = clockwork.NewFakeClockAt(ts)
	ride := &ridesData.Ride{Model: models.Model{ID: 123}, Name: "ride1", Capacity: 4, RideTime: 10 * time.Minute}
	customer := &customers.Customer{Model: models.Model{ID: 111}}

	rides.LogCustomerJoinedRideQueue(db, ride, customer)
	rides.LogCustomerJoinedRideQueue(db, ride, customer)
	rides.LogCustomerJoinedRideQueue(db, ride, customer)

	state, err := rides.GetCurrentState(db, ride)
	assert.NilError(t, err)
	assert.Equal(t, uint(3), state.QueueCount)
	// expected wait still to be now since first batch is not full
	assert.DeepEqual(t, ts, state.EstimatedWaitTill)

	rides.LogCustomerJoinedRideQueue(db, ride, customer)
	state, _ = rides.GetCurrentState(db, ride)
	// expected wait to be increased to future after a batch was filled
	assert.DeepEqual(t, ts.Add(10*time.Minute), state.EstimatedWaitTill)

	rides.LogCustomerJoinedRideQueue(db, ride, customer)
	rides.LogCustomerLeftRideQueue(db, ride, customer)
	state, _ = rides.GetCurrentState(db, ride)
	// expected there to be no change in waits when adding customer and removing negates each other
	assert.DeepEqual(t, ts.Add(10*time.Minute), state.EstimatedWaitTill)

	rides.LogCustomerLeftRideQueue(db, ride, customer)
	state, _ = rides.GetCurrentState(db, ride)
	// removing another use reduces the batch size and wait as well
	assert.DeepEqual(t, ts, state.EstimatedWaitTill)

	// Offload all customers in queue
	rides.LogCustomerLeftRideQueue(db, ride, customer)
	rides.LogCustomerLeftRideQueue(db, ride, customer)
	rides.LogCustomerLeftRideQueue(db, ride, customer)
	rides.LogCustomerLeftRideQueue(db, ride, customer)
	state, err = rides.GetCurrentState(db, ride)
	assert.NilError(t, err)
	assert.DeepEqual(t, ts, state.EstimatedWaitTill)
	assert.DeepEqual(t, uint(0), state.QueueCount)
}

func TestRideWaitEstimatesAfterBatchOfCustomerExits(t *testing.T) {
	db := testDB(t.Name())
	ts := time.Now()
	rides.Clock = clockwork.NewFakeClockAt(ts)
	ride := &ridesData.Ride{Model: models.Model{ID: 123}, Name: "ride1", Capacity: 2, RideTime: 1 * time.Minute}
	customer := &customers.Customer{Model: models.Model{ID: 111}}

	for i := 0; i < 10; i++ {
		rides.LogCustomerJoinedRideQueue(db, ride, customer)
	}

	wait := ts.Add(time.Duration(ride.RideTime.Seconds()*10/2) * time.Second)
	state, _ := rides.GetCurrentState(db, ride)
	assert.Equal(t, wait, state.EstimatedWaitTill)

	// After a batch is over for the ride
	ts = ts.Add(ride.RideTime)
	wait = ts.Add(time.Duration(ride.RideTime.Seconds()*8/2) * time.Second)
	state, _ = rides.GetCurrentState(db, ride)
	assert.Equal(t, wait, state.EstimatedWaitTill)

	// After two batch is over for the ride
	ts = ts.Add(ride.RideTime * 2)
	wait = ts.Add(time.Duration(ride.RideTime.Seconds()*4/2) * time.Second)
	state, _ = rides.GetCurrentState(db, ride)
	assert.Equal(t, wait, state.EstimatedWaitTill)
}
