package events

import (
	"time"

	"gitlab.com/therako/universal-studios/data/models"
	"gorm.io/gorm"
)

// DB table names
const (
	TableName = "events"
)

// EventInterface all events should adhere to this contract
type EventInterface interface {
	ToDBEvent() (event *Event, err error)
	FromDBEvent(event *Event) (err error)
}

// Event defines all customer & ride queue activities
type Event struct {
	models.Model
	SourceID      uint       `gorm:"column:source_id" json:"source_id"`
	At            time.Time  `gorm:"column:at" json:"at"`
	EndsAt        *time.Time `gorm:"column:ends_at" json:"ends_at"`
	AggregateRoot string     `gorm:"column:aggregate_root" json:"aggregate_root"`
	Name          string     `gorm:"column:name" json:"name"`
	Data          []byte     `gorm:"column:data" json:"data"`
}

// DAO is data access object for rides
type DAO struct {
	DB *gorm.DB
}

// Add adds the new event to DB
func (r DAO) Add(event EventInterface) (err error) {
	var e *Event
	e, err = event.ToDBEvent()
	if err != nil {
		return
	}

	err = r.DB.Create(e).Error
	return
}

// EventFor returns all events for a source ID for an aggregate sorted by event time (At)
func (r DAO) EventFor(id uint, aggregate string) ([]*Event, error) {
	events := []*Event{}
	err := r.DB.Table(TableName).Where("source_id = ? AND aggregate_root = ?", id, aggregate).Order("at asc").Find(&events).Error
	return events, err
}
