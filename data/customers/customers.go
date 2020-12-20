package customers

import (
	"time"

	"gitlab.com/therako/universal-studios/data/models"
	"gorm.io/gorm"
)

// DB table names
const (
	TableName = "customers"
)

// Customer DB model for the studios
type Customer struct {
	models.Model
	// For simplicity let's ignore all customer personal info and use just ID's
	ExitAt *time.Time `gorm:"column:exit_at" json:"exit_at"`
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
func (r DAO) Exit(id uint) (customer *Customer, err error) {
	customer = &Customer{Model: models.Model{ID: id}}
	err = r.DB.First(&customer).Error
	if err != nil {
		return
	}

	customer.ExitAt = models.TimeP(time.Now())
	err = r.DB.Save(&customer).Error
	if err != nil {
		return nil, err
	}

	return customer, nil
}

// Get returns the customer details
func (r DAO) Get(id uint) (customer *Customer, err error) {
	customer = &Customer{Model: models.Model{ID: id}}
	err = r.DB.First(&customer).Error
	return
}
