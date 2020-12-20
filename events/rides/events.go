package rides

import (
	"encoding/json"
	"time"

	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	ridesData "gitlab.com/therako/universal-studios/data/rides"
)

const (
	// AggregateRoot is grouping key for all ride based events
	AggregateRoot = "Ride"
)

// RideCustomerQueued is an event representing rides when customers join the queue
type RideCustomerQueued struct {
	Ride     *ridesData.Ride     `json:"ride"`
	Customer *customers.Customer `json:"customer"`
	From     time.Time           `json:"From"`
	To       time.Time           `json:"To"`
}

func (e *RideCustomerQueued) FromDBEvent(event *events.Event) (err error) {
	err = json.Unmarshal(event.Data, e)
	return
}

func (e RideCustomerQueued) ToDBEvent() (*events.Event, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return &events.Event{
		SourceID:      e.Ride.ID,
		AggregateRoot: AggregateRoot,
		Name:          "RideCustomerQueued",
		At:            e.From,
		EndsAt:        &e.To,
		Data:          data,
	}, nil
}

func (e RideCustomerQueued) Aggregate(state *RideState) {
	if e.To.Before(time.Now()) {
		// Skip ended events
		return
	}

	state.QueueCount++
	state.calculateNewWait(e.Ride, false)
}

// RideCustomerUnQueued is an event representing rides when customers leaves the queue
type RideCustomerUnQueued struct {
	Ride     *ridesData.Ride     `json:"ride"`
	Customer *customers.Customer `json:"customer"`
	At       time.Time           `json:"At"`
}

func (e *RideCustomerUnQueued) FromDBEvent(event *events.Event) (err error) {
	err = json.Unmarshal(event.Data, e)
	return
}

func (e RideCustomerUnQueued) ToDBEvent() (*events.Event, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return &events.Event{
		SourceID:      e.Ride.ID,
		AggregateRoot: AggregateRoot,
		Name:          "RideCustomerUnQueued",
		At:            e.At,
		Data:          data,
	}, nil
}

func (e RideCustomerUnQueued) Aggregate(state *RideState) {
	if state.QueueCount == 0 {
		return
	}

	state.QueueCount--
	state.calculateNewWait(e.Ride, true)
}
