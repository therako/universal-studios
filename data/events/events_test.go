package events_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"gitlab.com/therako/universal-studios/data/events"
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
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)
)

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{Logger: gormLogger})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&events.Event{})
	return gormDB
}

type testEvent struct {
	SourceID      uint
	AggregateRoot string
	At            time.Time
	testData      []byte
}

func (e *testEvent) FromDBEvent(event *events.Event) (err error) { panic("Unimplemented") }

func (e *testEvent) ToDBEvent() (*events.Event, error) {
	return &events.Event{
		SourceID:      e.SourceID,
		AggregateRoot: e.AggregateRoot,
		At:            e.At,
		Data:          e.testData,
	}, nil
}

func TestAddEvent(t *testing.T) {
	db := testDB(t.Name())
	dao := events.DAO{DB: db}
	event := &testEvent{
		SourceID:      123,
		AggregateRoot: "test",
		At:            time.Now(),
		testData:      []byte("event data bytes"),
	}

	err := dao.Add(event)

	assert.NilError(t, err)
	eventInDB := &events.Event{}
	db.Table(events.TableName).First(eventInDB)
	assert.Equal(t, event.SourceID, eventInDB.SourceID)
	assert.Equal(t, event.AggregateRoot, eventInDB.AggregateRoot)
	assert.DeepEqual(t, event.At, eventInDB.At)
}

func TestEventsForSourceID(t *testing.T) {
	t.Run("expected to return source events for the aggregate in increasing time order excluding ended events", func(t *testing.T) {
		db := testDB(t.Name())
		dao := events.DAO{DB: db}
		testingSourceID := uint(123)
		testingAggregate := "Customer"
		eventTime := time.Now()
		db.Create([]*events.Event{
			{
				SourceID:      testingSourceID,
				AggregateRoot: testingAggregate,
				Name:          "CustomerQueued",
				At:            eventTime.Add(1 * time.Second),
				EndsAt:        models.TimeP(eventTime.Add(10 * time.Second)),
				Data:          []byte("right source delayed event start"),
			},
			{
				SourceID:      987,
				AggregateRoot: testingAggregate,
				Name:          "CustomerQueued",
				At:            eventTime,
				EndsAt:        models.TimeP(eventTime.Add(10 * time.Second)),
				Data:          []byte("wrong source event"),
			},
			{
				SourceID:      testingSourceID,
				AggregateRoot: "Ride",
				Name:          "RideCustomerQueued",
				At:            eventTime,
				EndsAt:        models.TimeP(eventTime.Add(1 * time.Second)),
				Data:          []byte("wrong aggregate event"),
			},
			{
				SourceID:      testingSourceID,
				AggregateRoot: testingAggregate,
				Name:          "CustomerQueued",
				At:            eventTime,
				EndsAt:        models.TimeP(eventTime.Add(10 * time.Second)),
				Data:          []byte("right source happening now"),
			},
		})

		events, err := dao.EventFor(123, "Customer")

		assert.NilError(t, err)
		assert.Equal(t, 2, len(events))
		assert.Equal(t, "Customer", events[0].AggregateRoot)
		assert.DeepEqual(t, eventTime, events[0].At)
		assert.Equal(t, "Customer", events[1].AggregateRoot)
		assert.DeepEqual(t, eventTime.Add(1*time.Second), events[1].At)
	})
}
