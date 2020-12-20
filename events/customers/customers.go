package customers

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/dgraph-io/ristretto"
	customersData "gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	ridesData "gitlab.com/therako/universal-studios/data/rides"
	"gitlab.com/therako/universal-studios/events/rides"
	"gorm.io/gorm"
)

// Cache A global cache for customer state
var Cache *ristretto.Cache

// Errors
var (
	ErrCustomerCantBeQueue   = errors.New("Customer is already in a queue or riding")
	ErrCustomerCantBeUnQueue = errors.New("Customer is not in any queue")
)

func init() {
	var err error
	Cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
}

// CustomerState represents a customers current state
type CustomerState struct {
	Queueing  bool
	RideID    uint
	From      time.Time
	To        time.Time
	UpdatedAt time.Time
}

// GetCurrentState from cache or calculate using events from DB
func GetCurrentState(db *gorm.DB, customer *customersData.Customer) (state *CustomerState, err error) {
	value, found := Cache.Get(strconv.Itoa(int(customer.ID)))
	if !found {
		log.Printf("Cache miss for customer %d\n", customer.ID)
		state, err = aggregateState(db, customer)
		return
	}

	var ok bool
	if state, ok = value.(*CustomerState); !ok {
		log.Printf("Cache value for customer %d is invalid\n", customer.ID)
		state, err = aggregateState(db, customer)
	}

	return
}

// LogCustomerInQueue validates and adds customer to queue of the ride
func LogCustomerInQueue(db *gorm.DB, customer *customersData.Customer, ride *ridesData.Ride) (err error) {
	var state *CustomerState
	state, err = GetCurrentState(db, customer)
	if err != nil {
		return
	}

	if state.Queueing && state.To.After(time.Now()) {
		return ErrCustomerCantBeQueue
	}

	err = rides.LogCustomerJoinedRideQueue(db, ride, customer)
	if err != nil {
		return
	}

	rideState, err := rides.GetCurrentState(db, ride)
	if err != nil {
		return
	}

	now := time.Now()
	e := &CustomerQueued{
		Customer: customer,
		Ride:     ride,
		From:     now,
		// To = whole journey (waiting time + ride time)
		To: rideState.EstimatedWaitTill.Add(ride.RideTime),
	}
	doa := events.DAO{DB: db}
	err = doa.Add(e)
	// State changed - invalidate cache
	Cache.Del(strconv.Itoa(int(customer.ID)))
	return
}

// LogCustomerLeftAQueue validates and removes customer from queue of the ride
func LogCustomerLeftAQueue(db *gorm.DB, customer *customersData.Customer) (err error) {
	var state *CustomerState
	state, err = GetCurrentState(db, customer)
	if err != nil {
		return
	}

	if !state.Queueing {
		return ErrCustomerCantBeUnQueue
	}

	rideDAO := ridesData.DAO{DB: db}
	ride, err := rideDAO.Get(state.RideID)
	if err != nil {
		return
	}

	err = rides.LogCustomerLeftRideQueue(db, ride, customer)
	if err != nil {
		return
	}

	now := time.Now()
	e := &CustomerUnQueued{
		Customer: customer,
		At:       now,
	}
	doa := events.DAO{DB: db}
	err = doa.Add(e)
	// State changed - invalidate cache
	Cache.Del(strconv.Itoa(int(customer.ID)))
	return
}

func aggregateState(db *gorm.DB, customer *customersData.Customer) (*CustomerState, error) {
	newState := &CustomerState{}

	dao := events.DAO{DB: db}
	events, err := dao.EventFor(customer.ID, AggregateRoot)
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		switch event.Name {
		case "CustomerQueued":
			e := &CustomerQueued{}
			err = e.FromDBEvent(event)
			if err != nil {
				return nil, err
			}
			e.Aggregate(newState)
		case "CustomerUnQueued":
			e := &CustomerUnQueued{}
			err = e.FromDBEvent(event)
			if err != nil {
				return nil, err
			}
			e.Aggregate(newState)
		default:
			log.Printf("Unknown event received with name: %s\n", event.Name)
		}
	}

	newState.UpdatedAt = time.Now()
	if newState.Queueing == true {
		Cache.SetWithTTL(strconv.Itoa(int(customer.ID)), newState, 0, newState.To.Sub(time.Now()))
	} else {
		Cache.Set(strconv.Itoa(int(customer.ID)), newState, 0)
	}

	return newState, nil
}
