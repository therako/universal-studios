package events

import (
	"log"
	"time"

	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/rides"
)

// Event all events should adhere to this contract
type Event interface {
	// Events have to define what to do when they are applied using this method
	Apply(ridesDAO rides.DAO, customersDAO customers.DAO) (err error)
}

// BaseEvent hold common metadata required for all events
type BaseEvent struct {
	At   time.Time
	Name string
}

// CustomerQueued is an event representing when a customer enters a queue for a ride
type CustomerQueued struct {
	BaseEvent
	Customer *customers.Customer
	Ride     *rides.Ride
}

// Apply will update customer & ride view accordingly
func (c CustomerQueued) Apply(ridesDAO rides.DAO, customersDAO customers.DAO) (err error) {
	rideState, err := ridesDAO.QueueACustomer(c.Ride)
	if err != nil {
		return err
	}

	_, err = customersDAO.QueueFor(c.Customer, c.Ride.ID, rideState.EstimatedWaiting)
	if err != nil {
		if _, er := ridesDAO.UnQueueACustomers(c.Ride); er != nil {
			log.Println(er)
		}
		return err
	}

	return
}

// CustomerUnQueued is an event representing when a customer exits a queue for a ride
type CustomerUnQueued struct {
	BaseEvent
	customer *customers.Customer
	ride     *rides.Ride
}
