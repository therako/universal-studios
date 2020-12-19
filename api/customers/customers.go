package customers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/models"
	"gorm.io/gorm"
)

const (
	TableName = "customers"
)

// Customer DB model for the studios
type Customer struct {
	models.Model
	// For simplicity let's ignore all customer personal info and use just ID's
	ExitAt *time.Time `gorm:"column:exit_at" json:"exit_at"`
}

type Customers struct {
	DB *gorm.DB
}

// List returns a list of all customers inside the studio
func (r Customers) List(c *gin.Context) {
	customers := &[]Customer{}
	err := r.DB.Table(TableName).Where("exit_at IS NULL").Find(&customers).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
	}

	c.JSON(http.StatusOK, customers)
}

// Enter marks a new customer entrying the studio
func (r Customers) Enter(c *gin.Context) {
	// We can mark each entry of a customer with a random id
	err := r.DB.Create(&Customer{ExitAt: nil}).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "added"})
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

	customer := &Customer{}
	err = r.DB.First(&customer, "id = ?", input.ID).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"err": "Customer not found"})
		return
	}

	customer.ExitAt = models.Timep(time.Now())
	err = r.DB.Save(&customer).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
