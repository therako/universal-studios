package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/rides"
	customersEvents "gitlab.com/therako/universal-studios/events/customers"
)

type Customers struct {
	DAO      customers.DAO
	RideDAO  rides.DAO
	eventDAO events.DAO
}

// List returns a list of all customers inside the studio
func (r Customers) List(c *gin.Context) {
	customers, err := r.DAO.List()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, customers)
}

// Enter marks a new customer entrying the studio
func (r Customers) Enter(c *gin.Context) {
	// We can mark each entry of a customer with a random id
	customer, err := r.DAO.Enter()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "added", "customer_id": customer.ID})
}

type exitForm struct {
	ID uint `form:"id" binding:"required"`
}

// Exit marks a the customer leving the studio
func (r Customers) Exit(c *gin.Context) {
	var input exitForm
	err := c.Bind(&input)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}

	customer, err := r.DAO.Exit(input.ID)
	if err != nil {
		handleError(c, err, "customer")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated", "customer_id": customer.ID})
}

type queueForm struct {
	ID     uint `form:"id" binding:"required"`
	RideID uint `form:"ride_id" binding:"required"`
}

// Queue marks a the customer entering a queue for a ride
func (r Customers) Queue(c *gin.Context) {
	var input queueForm
	err := c.Bind(&input)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}

	customer, err := r.DAO.Get(input.ID)
	if err != nil {
		handleError(c, err, "customer")
		return
	}

	ride, err := r.RideDAO.Get(input.RideID)
	if err != nil {
		handleError(c, err, "ride")
		return
	}

	err = customersEvents.LogCustomerInQueue(r.DAO.DB, customer, ride)
	if err != nil {
		handleError(c, err, "queue")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "queued", "customer_id": customer.ID})
}

type unqueueForm struct {
	ID uint `form:"id" binding:"required"`
}

// UnQueue marks a the customer leaving a queue for a ride before ride + wait timer runs out
func (r Customers) UnQueue(c *gin.Context) {
	var input unqueueForm
	err := c.Bind(&input)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}

	customer, err := r.DAO.Get(input.ID)
	if err != nil {
		handleError(c, err, "customer")
		return
	}

	err = customersEvents.LogCustomerLeftAQueue(r.DAO.DB, customer)
	if err != nil {
		handleError(c, err, "un-queue")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "un-queued", "customer_id": customer.ID})
}
