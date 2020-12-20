package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/data/rides"
	ridesEvents "gitlab.com/therako/universal-studios/events/rides"
)

type Rides struct {
	DAO rides.DAO
}

// List returns a list of studio rides
func (r Rides) List(c *gin.Context) {
	rides, err := r.DAO.List()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	for idx, ride := range rides {
		rideState, err := ridesEvents.GetCurrentState(r.DAO.DB, ride)
		if err == nil {
			waitTime := time.Duration(0)
			if !rideState.EstimatedWaitTill.IsZero() {
				waitTime = rideState.EstimatedWaitTill.Sub(time.Now())
			}
			if waitTime < 0 {
				waitTime = 0
			}
			rides[idx].EstimatedWaitingTime = waitTime
			rides[idx].InQueue = rideState.QueueCount
		}
	}

	c.JSON(http.StatusOK, rides)
}

type AddForm struct {
	Name         string `form:"name" binding:"required"`
	Desc         string `form:"desc"`
	RideTimeSecs uint   `form:"ride_time_secs" binding:"required"`
	Capacity     uint   `form:"capacity" binding:"required"`
}

// Add adds a new ride to the studio
func (r Rides) Add(c *gin.Context) {
	var input AddForm
	err := c.Bind(&input)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{"err": "Invalid request input. Expected atleast name, capacity & ride_time_secs"},
		)
		return
	}

	err = r.DAO.Add(&rides.Ride{
		Name:     input.Name,
		Desc:     input.Desc,
		Capacity: input.Capacity,
		RideTime: time.Duration(input.RideTimeSecs) * time.Second,
	})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "added"})
}
