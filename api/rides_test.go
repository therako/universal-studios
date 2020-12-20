package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/data/customers"
	"gitlab.com/therako/universal-studios/data/events"
	"gitlab.com/therako/universal-studios/data/models"
	"gitlab.com/therako/universal-studios/data/rides"
	ridesEvents "gitlab.com/therako/universal-studios/events/rides"
	"gotest.tools/v3/assert"
)

func TestRideEndpoints(t *testing.T) {
	t.Run("expected to return all rides stores in DB", func(t *testing.T) {
		db := testDB(t.Name())
		rides := []*rides.Ride{
			{Name: "RollerCoster", Desc: "World's best roller cosater", RideTime: 4 * time.Minute},
			{Name: "BumperCar", Desc: "Bump all the way", RideTime: 7 * time.Minute},
		}
		db.Create(&rides)
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/ride", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		ridesStr, _ := json.Marshal(rides)
		assert.Equal(t, string(ridesStr), w.Body.String())
	})

	t.Run("expected to return estimated wait time and queue counter where available", func(t *testing.T) {
		db := testDB(t.Name())
		allRides := []*rides.Ride{
			{Model: models.Model{ID: 1234}, Name: "RollerCoster", Desc: "World's best roller cosater", Capacity: 2, RideTime: 4 * time.Minute},
			{Model: models.Model{ID: 1235}, Name: "BumperCar", Desc: "Bump all the way", Capacity: 4, RideTime: 7 * time.Minute},
		}
		db.Create(&allRides)
		customer := &customers.Customer{}
		db.Create(&customer)
		eventDAO := events.DAO{DB: db}
		eventDAO.Add(&ridesEvents.RideCustomerQueued{Ride: allRides[0], Customer: customer, From: time.Now(), To: time.Now().Add(10 * time.Minute)})
		eventDAO.Add(&ridesEvents.RideCustomerQueued{Ride: allRides[0], Customer: customer, From: time.Now(), To: time.Now().Add(10 * time.Minute)})
		eventDAO.Add(&ridesEvents.RideCustomerQueued{Ride: allRides[0], Customer: customer, From: time.Now(), To: time.Now().Add(10 * time.Minute)})
		eventDAO.Add(&ridesEvents.RideCustomerQueued{Ride: allRides[0], Customer: customer, From: time.Now(), To: time.Now().Add(10 * time.Minute)})
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/ride", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var responseRides []*rides.Ride
		err := json.Unmarshal(w.Body.Bytes(), &responseRides)
		assert.NilError(t, err)
		// There are 4 in queue for a ride with 2 capacity, so wait time is more than 1 ride time
		assert.Assert(t, responseRides[0].EstimatedWaitingTime > 4*time.Minute)
		// but less than 2 rides as time moves forward
		assert.Assert(t, responseRides[0].EstimatedWaitingTime < 8*time.Minute)
	})
}
func TestRideAddEndpoints(t *testing.T) {
	t.Run("expected to add ride to DB when all values are present", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("name", "DareDevil")
		form.Add("desc", "Booooo")
		form.Add("ride_time_secs", "300")
		form.Add("capacity", "20")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/ride/add", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"status":"added"}`, w.Body.String())

		ride := &rides.Ride{}
		db.Table(rides.TableName).First(&ride)
		assert.Equal(t, "DareDevil", ride.Name)
		assert.Equal(t, "Booooo", ride.Desc)
		assert.Equal(t, 300*time.Second, ride.RideTime)
		assert.Equal(t, uint(20), ride.Capacity)
	})

	t.Run("error on missing input", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/ride/add", nil)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		assert.Equal(t, `{"err":"Invalid request input. Expected atleast name, capacity \u0026 ride_time_secs"}`, w.Body.String())
	})
}
