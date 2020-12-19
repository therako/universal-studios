package rides

import (
	"errors"
	"time"

	"gitlab.com/therako/universal-studios/data/models"
	"gorm.io/gorm"
)

// DB table names
const (
	TableName          = "rides"
	RideStateTableName = "ride_states"
)

// Errors
var (
	ErrRideHasNoQueue = errors.New("Ride has no one in queue to remove")
)

// Ride DB model for the studios
type Ride struct {
	models.Model
	Name     string        `gorm:"column:name" json:"name"`
	Desc     string        `gorm:"column:desc" json:"desc"`
	RideTime time.Duration `gorm:"column:ride_time" json:"ride_time"`
	Capacity uint          `gorm:"column:capacity" json:"capacity"`
}

// RideState a view of current state of a ride
type RideState struct {
	RideID           uint          `gorm:"primary_key;column:ride_id" json:"ride_id"`
	CustomersInQueue uint          `gorm:"column:customers_in_queue" json:"customers_in_queue"`
	EstimatedWaiting time.Duration `gorm:"column:estimated_waiting" json:"estimated_waiting"`
	Since            time.Time     `gorm:"column:since" json:"since"`
	CreatedAt        time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

// DAO is data access object for rides
type DAO struct {
	DB *gorm.DB
}

// List returns a list of studio rides
func (r DAO) List() (rides []*Ride, err error) {
	err = r.DB.Table(TableName).Scan(&rides).Error
	return
}

// Add adds a new ride to the studio
func (r DAO) Add(ride *Ride) (err error) {
	err = r.DB.Create(&ride).Error
	return
}

func estimateWaitingTime(ride *Ride, state *RideState) time.Duration {
	trips := state.CustomersInQueue / ride.Capacity

	if trips < 1 {
		// Means there's no queue
		return 0
	}
	return time.Duration(int64(ride.RideTime) * int64(trips))
}

// QueueACustomer update ride state by one customer at a time
func (r DAO) QueueACustomer(ride *Ride) (*RideState, error) {
	rideState := &RideState{RideID: ride.ID}
	err := r.DB.Table(RideStateTableName).Find(&rideState).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	rideState.CustomersInQueue++
	// Add the customer and re-estimate waiting time
	rideState.EstimatedWaiting = estimateWaitingTime(ride, rideState)
	rideState.Since = time.Now()

	err = r.DB.Save(&rideState).Error
	return rideState, err
}

// UnQueueACustomers updates ride state by removing A customer at a time
func (r DAO) UnQueueACustomers(ride *Ride) (*RideState, error) {
	rideState := &RideState{RideID: ride.ID}
	err := r.DB.Table(RideStateTableName).Find(&rideState).Error
	if err != nil {
		return nil, err
	}

	if rideState.CustomersInQueue == 0 {
		return nil, ErrRideHasNoQueue
	}

	rideState.CustomersInQueue--
	// Add the customer and re-estimate waiting time
	rideState.EstimatedWaiting = estimateWaitingTime(ride, rideState)
	rideState.Since = time.Now()

	err = r.DB.Save(&rideState).Error
	return rideState, err
}
