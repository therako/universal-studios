package customers_test

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/models"
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
	gormDB.AutoMigrate(&customers.Customer{})
	gormDB.AutoMigrate(&customers.CustomerState{})
	return gormDB
}

func TestQueueFor(t *testing.T) {
	t.Run("expected to create new state for customer's first queue", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{}
		db.Create(&customer)
		dao := customers.DAO{DB: db}

		customerState, err := dao.QueueFor(customer, 123, 100*time.Second)

		assert.NilError(t, err)
		assert.Equal(t, customer.ID, customerState.CustomerID)
		assert.Equal(t, uint(123), *customerState.RideID)
		assert.Equal(t, 100*time.Second, customerState.EstimatedWaiting)
		assert.Assert(t, customerState.CreatedAt.Before(time.Now()))
		assert.Assert(t, customerState.UpdatedAt.Before(time.Now()))
	})

	t.Run("expected to update customer state with new queue info", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{}
		db.Create(&customer)
		currentCustomerState := &customers.CustomerState{CustomerID: customer.ID}
		db.Create(&currentCustomerState)
		dao := customers.DAO{DB: db}

		newCustomerState, err := dao.QueueFor(customer, 123, 100*time.Second)

		assert.NilError(t, err)
		assert.Equal(t, customer.ID, newCustomerState.CustomerID)
		assert.Equal(t, uint(123), *newCustomerState.RideID)
		assert.Equal(t, 100*time.Second, newCustomerState.EstimatedWaiting)
		assert.DeepEqual(t, newCustomerState.CreatedAt, currentCustomerState.CreatedAt)
		assert.Assert(t, newCustomerState.UpdatedAt.After(currentCustomerState.UpdatedAt))
	})

	t.Run("expected to error when customer is already queued somewhere", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{}
		db.Create(&customer)
		currentCustomerState := &customers.CustomerState{CustomerID: customer.ID, RideID: models.UintP(345)}
		db.Create(&currentCustomerState)
		dao := customers.DAO{DB: db}

		_, err := dao.QueueFor(customer, 123, 10*time.Second)

		assert.Assert(t, errors.Is(err, customers.ErrAlreadyQueued))
	})
}
