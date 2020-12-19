package rides

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/models"
	"gorm.io/gorm"
)

const (
	TableName = "rides"
)

// Ride DB model for the studios
type Ride struct {
	models.Model
	Name     string        `gorm:"column:name" json:"name"`
	Desc     string        `gorm:"column:desc" json:"desc"`
	RideTime time.Duration `gorm:"column:ride_time" json:"ride_time"`
}

type Rides struct {
	DB *gorm.DB
}

// List returns a list of studio rides
func (r Rides) List(c *gin.Context) {
	rides := &[]Ride{}
	err := r.DB.Table(TableName).Scan(&rides).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
	}

	c.JSON(http.StatusOK, rides)
}

type AddForm struct {
	Name         string `form:"name" binding:"required"`
	Desc         string `form:"desc"`
	RideTimeSecs uint   `form:"ride_time_secs" binding:"required"`
}

// Add adds a new ride to the studio
func (r Rides) Add(c *gin.Context) {
	var input AddForm
	err := c.Bind(&input)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{"err": "Invalid request input. Expected atleast name & ride_time_secs"},
		)
		return
	}

	err = r.DB.Create(&Ride{
		Name:     input.Name,
		Desc:     input.Desc,
		RideTime: time.Duration(input.RideTimeSecs) * time.Second,
	}).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "added"})
}
