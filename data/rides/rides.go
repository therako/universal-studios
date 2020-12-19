package rides

import (
	"time"

	"gitlab.com/therako/universal-studios/data/models"
	"gorm.io/gorm"
)

// DB table names
const (
	TableName = "rides"
)

// Ride DB model for the studios
type Ride struct {
	models.Model
	Name     string        `gorm:"column:name" json:"name"`
	Desc     string        `gorm:"column:desc" json:"desc"`
	RideTime time.Duration `gorm:"column:ride_time" json:"ride_time"`
	Capacity uint          `gorm:"column:capacity" json:"capacity"`
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

// Get returns a single studio ride
func (r DAO) Get(id uint) (ride *Ride, err error) {
	ride = &Ride{Model: models.Model{ID: id}}
	err = r.DB.Table(TableName).First(ride).Error
	return
}

// Add adds a new ride to the studio
func (r DAO) Add(ride *Ride) (err error) {
	err = r.DB.Create(&ride).Error
	return
}
