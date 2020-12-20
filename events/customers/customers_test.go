package customers_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	customersData "gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/models"
	ridesData "gitlab.com/therako/universal-studios/data/rides"
	"gitlab.com/therako/universal-studios/events/customers"
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
	customers.Cache.Clear()
}

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{Logger: gormLogger})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&customersData.Customer{})
	gormDB.AutoMigrate(&ridesData.Ride{})
	gormDB.AutoMigrate(&events.Event{})
	return gormDB
}
func TestGetCurrentState(t *testing.T) {
	customer := &customersData.Customer{Model: models.Model{ID: 110}}
	ride1 := &ridesData.Ride{Name: "ride1"}
	ride2 := &ridesData.Ride{Name: "ride2"}
	customerStartTime := time.Now()

	db := testDB(t.Name())
	dao := events.DAO{DB: db}
	dao.Add(&customers.CustomerQueued{
		Customer: customer,
		Ride:     ride1,
		From:     customerStartTime,
		To:       customerStartTime.Add(10 * time.Millisecond),
	})
	dao.Add(&customers.CustomerUnQueued{
		Customer: customer,
	})
	dao.Add(&customers.CustomerQueued{
		Customer: customer,
		Ride:     ride2,
		From:     customerStartTime.Add(20 * time.Millisecond),
		To:       customerStartTime.Add(100 * time.Millisecond),
	})

	state, err := customers.GetCurrentState(db, customer)
	assert.NilError(t, err)
	assert.Equal(t, ride2.ID, state.RideID)
	assert.Equal(t, true, state.Queueing)
	assert.DeepEqual(t, customerStartTime.Add(20*time.Millisecond), state.From)
	assert.DeepEqual(t, customerStartTime.Add(100*time.Millisecond), state.To)

	stateFromCache, _ := customers.GetCurrentState(db, customer)
	assert.Equal(t, ride2.ID, stateFromCache.RideID)
	assert.Equal(t, true, stateFromCache.Queueing)
	assert.DeepEqual(t, customerStartTime.Add(20*time.Millisecond), stateFromCache.From)
	assert.DeepEqual(t, customerStartTime.Add(100*time.Millisecond), stateFromCache.To)
}

func TestE2ECustomerQueuingFlow(t *testing.T) {
	db := testDB(t.Name())
	ts := time.Now()
	rides.Clock = clockwork.NewFakeClockAt(ts)
	customer := &customersData.Customer{Model: models.Model{ID: 111}}
	db.Create(customer)
	ride1 := &ridesData.Ride{Model: models.Model{ID: 123}, Name: "ride1", Capacity: 4, RideTime: 10 * time.Minute}
	db.Create(ride1)
	ride2 := &ridesData.Ride{Model: models.Model{ID: 456}, Name: "ride2", Capacity: 4, RideTime: 10 * time.Minute}
	db.Create(ride2)

	err := customers.LogCustomerInQueue(db, customer, ride1)
	assert.NilError(t, err, "expected to queue customer with no error")

	err = customers.LogCustomerInQueue(db, customer, ride2)
	assert.Error(t, err, customers.ErrCustomerCantBeQueue.Error(), "expected to fail since customer is already in queue for ride1")

	err = customers.LogCustomerLeftAQueue(db, customer)
	assert.NilError(t, err, "expected to unqueue customer from ride1")

	err = customers.LogCustomerLeftAQueue(db, customer)
	assert.Error(t, err, customers.ErrCustomerCantBeUnQueue.Error(), "expected error since customer is already unqueued")

	err = customers.LogCustomerInQueue(db, customer, ride2)
	assert.NilError(t, err, "expected to queue customer with no error to ride2")

	state, err := customers.GetCurrentState(db, customer)
	assert.NilError(t, err)
	assert.Equal(t, ride2.ID, state.RideID)
	assert.Equal(t, true, state.Queueing)
	assert.Assert(t, state.From.Before(time.Now()))
	assert.Assert(t, state.To.After(time.Now()))

	// Fill the ride capacity and validate From & To time in customer state
	customers.LogCustomerInQueue(db, &customersData.Customer{Model: models.Model{ID: 115}}, ride2)
	customers.LogCustomerInQueue(db, &customersData.Customer{Model: models.Model{ID: 116}}, ride2)
	customers.LogCustomerLeftAQueue(db, &customersData.Customer{Model: models.Model{ID: 116}})
	customers.LogCustomerInQueue(db, &customersData.Customer{Model: models.Model{ID: 117}}, ride2)
	customers.LogCustomerInQueue(db, &customersData.Customer{Model: models.Model{ID: 118}}, ride2)

	state, _ = customers.GetCurrentState(db, &customersData.Customer{Model: models.Model{ID: 117}})
	// expected to be in the same batch, only time is ride time for this user
	assert.DeepEqual(t, ts.Add(10*time.Minute), state.To)

	state, _ = customers.GetCurrentState(db, &customersData.Customer{Model: models.Model{ID: 118}})
	// expected to be in the new batch
	assert.DeepEqual(t, ts.Add(20*time.Minute), state.To)
}
