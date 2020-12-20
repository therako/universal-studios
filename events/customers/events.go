package customers

import (
	"encoding/json"
	"time"

	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/rides"
)

const (
	// AggregateRoot is grouping key for all customer events
	AggregateRoot = "Customer"
)

// CustomerQueued is an event representing when a customer enters a queue for a ride
type CustomerQueued struct {
	Customer *customers.Customer `json:"customer"`
	Ride     *rides.Ride         `json:"ride"`
	From     time.Time           `json:"from"`
	To       time.Time           `json:"to"`
}

func (e *CustomerQueued) FromDBEvent(event *events.Event) (err error) {
	err = json.Unmarshal(event.Data, e)
	return
}

func (e CustomerQueued) ToDBEvent() (*events.Event, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return &events.Event{
		SourceID:      e.Customer.ID,
		AggregateRoot: AggregateRoot,
		Name:          "CustomerQueued",
		At:            e.From,
		EndsAt:        &e.To,
		Data:          data,
	}, nil
}

func (e CustomerQueued) Aggregate(state *CustomerState) {
	if e.To.Before(time.Now()) {
		state.Queueing = false
		state.RideID = 0
		state.From = time.Now()
		state.To = time.Time{}
		return
	}

	state.Queueing = true
	state.RideID = e.Ride.ID
	state.From = e.From
	state.To = e.To
}

// CustomerUnQueued is an event representing when a customer exits a queue for a ride
type CustomerUnQueued struct {
	Customer *customers.Customer `json:"customer"`
	At       time.Time
}

func (e *CustomerUnQueued) FromDBEvent(event *events.Event) (err error) {
	err = json.Unmarshal(event.Data, e)
	return
}

func (e CustomerUnQueued) ToDBEvent() (*events.Event, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return &events.Event{
		SourceID:      e.Customer.ID,
		AggregateRoot: AggregateRoot,
		Name:          "CustomerUnQueued",
		At:            e.At,
		Data:          data,
	}, nil
}

func (e CustomerUnQueued) Aggregate(state *CustomerState) {
	state.Queueing = false
	state.RideID = 0
	state.From = e.At
	state.To = time.Time{}
}
