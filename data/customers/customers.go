package customers

import (
	"errors"
	"fmt"
	"time"

	"gitlab.com/therako/universal-studios/data/models"
	"gorm.io/gorm"
)

// DB table names
const (
	TableName              = "customers"
	CustomerStateTableName = "customer_states"
)

// Errors
var (
	ErrCustomerNotFound = errors.New("Customer not found")
	ErrAlreadyQueued    = errors.New("Customer already in queue")
)

// Customer DB model for the studios
type Customer struct {
	models.Model
	// For simplicity let's ignore all customer personal info and use just ID's
	ExitAt *time.Time `gorm:"column:exit_at" json:"exit_at"`
}

// CustomerState a view of customer current queue state
type CustomerState struct {
	CustomerID       uint          `gorm:"primary_key;column:customer_id" json:"customer_id"`
	RideID           *uint         `gorm:"column:ride_id" json:"ride_id"`
	EstimatedWaiting time.Duration `gorm:"column:estimated_waiting" json:"estimated_waiting"`
	CreatedAt        time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

// DAO is data access object for customer
type DAO struct {
	DB *gorm.DB
}

// List returns a list of all customers inside the studio
func (r DAO) List() (customers []*Customer, err error) {
	err = r.DB.Table(TableName).Where("exit_at IS NULL").Find(&customers).Error
	return
}

// Enter marks a new customer entrying the studio
func (r DAO) Enter() (*Customer, error) {
	newCustomer := &Customer{ExitAt: nil}
	err := r.DB.Create(newCustomer).Error
	return newCustomer, err
}

// Exit marks a the customer leving the studio
func (r DAO) Exit(id uint) (*Customer, error) {
	customer := &Customer{}
	err := r.DB.First(&customer, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("%w, id = %d", ErrCustomerNotFound, id)
	}

	customer.ExitAt = models.TimeP(time.Now())
	err = r.DB.Save(&customer).Error
	if err != nil {
		return nil, err
	}

	return customer, nil
}

// QueueFor update customer queuing state
func (r DAO) QueueFor(customer *Customer, rideID uint, estimatedWaiting time.Duration) (*CustomerState, error) {
	if customer.ID == 0 {
		return nil, ErrCustomerNotFound
	}

	customerState := &CustomerState{CustomerID: customer.ID}
	err := r.DB.Table(CustomerStateTableName).Find(&customerState).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if customerState.RideID != nil {
		return customerState, ErrAlreadyQueued
	}

	customerState.RideID = &rideID
	customerState.EstimatedWaiting = estimatedWaiting
	err = r.DB.Save(&customerState).Error
	return customerState, err
}
