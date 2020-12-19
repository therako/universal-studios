package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/data/customers"
)

type Customers struct {
	DAO customers.DAO
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
		if errors.Is(err, customers.ErrCustomerNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"err": "Customer not found"})
		} else {
			c.AbortWithStatus(http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated", "customer_id": customer.ID})
}
