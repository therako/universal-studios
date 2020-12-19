package rides

import (
	"log"
	"strconv"
	"time"

	"github.com/dgraph-io/ristretto"
	"gorm.io/gorm"

	clock "github.com/jonboulle/clockwork"

	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	ridesData "gitlab.com/therako/universal-studios/data/rides"
)

// Cache A global cache for customer state
var Cache *ristretto.Cache

// Clock - for test overrides only
var Clock clock.Clock

func init() {
	Clock = clock.NewRealClock()
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

// RideState represents a ride's current state
type RideState struct {
	UpdatedAt time.Time

	QueueCount        uint
	EstimatedWaitTill time.Time
}

func (s *RideState) calculateNewWait(ride *ridesData.Ride, isReduced bool) {
	if s.EstimatedWaitTill.IsZero() {
		s.EstimatedWaitTill = Clock.Now()
	}

	batches := (s.QueueCount / ride.Capacity)
	remainingSeatsInCurrentBatch := (s.QueueCount % ride.Capacity)
	if !isReduced && remainingSeatsInCurrentBatch == 0 && batches >= 1 {
		// Filled capacity by another batch in the queue, add ride time for the estimated_wait
		s.EstimatedWaitTill = s.EstimatedWaitTill.Add(ride.RideTime)
	}

	if isReduced && remainingSeatsInCurrentBatch == ride.Capacity-1 {
		// A seat got vacated in the previous batch due to some one leaving the queue
		s.EstimatedWaitTill = s.EstimatedWaitTill.Add(-ride.RideTime)
	}

	if s.EstimatedWaitTill.Before(Clock.Now()) {
		// Wait time can't be in the past, so adjsut wait to now
		s.EstimatedWaitTill = Clock.Now()
	}
	return
}

// GetCurrentState from cache or calculate using events from DB
func GetCurrentState(db *gorm.DB, ride *ridesData.Ride) (state *RideState, err error) {
	value, found := Cache.Get(strconv.Itoa(int(ride.ID)))
	if !found {
		log.Printf("Cache miss for ride %d\n", ride.ID)
		state, err = aggregateState(db, ride)
		return
	}

	var ok bool
	if state, ok = value.(*RideState); !ok {
		log.Printf("Cache value for ride %d is invalid\n", ride.ID)
		state, err = aggregateState(db, ride)
	}

	return
}

// LogCustomerJoinedRideQueue validates and adds customer in queue of the ride
func LogCustomerJoinedRideQueue(db *gorm.DB, ride *ridesData.Ride, customer *customers.Customer) (err error) {
	now := time.Now()
	e := &RideCustomerQueued{
		Ride:     ride,
		Customer: customer,
		From:     now,
		// Fix -- Add ride waiting time estimates here
		To: now.Add(ride.RideTime),
	}
	doa := events.DAO{DB: db}
	err = doa.Add(e)
	// State changed - invalidate cache
	Cache.Del(strconv.Itoa(int(ride.ID)))
	return
}

// LogCustomerLeftRideQueue validates and removes customer from queue of the ride
func LogCustomerLeftRideQueue(db *gorm.DB, ride *ridesData.Ride, customer *customers.Customer) (err error) {
	now := time.Now()
	e := &RideCustomerUnQueued{
		Ride:     ride,
		Customer: customer,
		At:       now,
	}
	doa := events.DAO{DB: db}
	err = doa.Add(e)
	// State changed - invalidate cache
	Cache.Del(strconv.Itoa(int(ride.ID)))
	return
}

func aggregateState(db *gorm.DB, ride *ridesData.Ride) (state *RideState, err error) {
	newState := &RideState{}

	dao := events.DAO{DB: db}
	events, err := dao.EventFor(ride.ID, AggregateRoot)
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		switch event.Name {
		case "RideCustomerQueued":
			e := &RideCustomerQueued{}
			err = e.FromDBEvent(event)
			if err != nil {
				return nil, err
			}
			e.Aggregate(newState)
		case "RideCustomerUnQueued":
			e := &RideCustomerUnQueued{}
			err = e.FromDBEvent(event)
			if err != nil {
				return nil, err
			}
			e.Aggregate(newState)
		}
	}

	newState.UpdatedAt = time.Now()
	Cache.Set(strconv.Itoa(int(ride.ID)), newState, 0)
	return newState, nil
}
