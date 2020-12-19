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
	"gitlab.com/therako/universal-studios/data/models"
	"gitlab.com/therako/universal-studios/data/rides"
	customerEvents "gitlab.com/therako/universal-studios/events/customers"
	"gotest.tools/v3/assert"
)

func TestCustomerEnter(t *testing.T) {
	t.Run("expected to create a new customer on entry", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/enter", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"customer_id":1,"status":"added"}`, w.Body.String())

		customer := &customers.Customer{}
		db.Table(customers.TableName).First(&customer)
		assert.Assert(t, customer.Model.ID == 1)
		assert.Assert(t, customer.Model.CreatedAt.Before(time.Now()))
		assert.Assert(t, customer.ExitAt == nil)
	})
}

func TestGetCustomer(t *testing.T) {
	t.Run("expected to return all customer who are inside the studio as of now", func(t *testing.T) {
		db := testDB(t.Name())
		input := []*customers.Customer{
			{Model: models.Model{ID: 1}, ExitAt: models.TimeP(time.Now().Add(-1 * time.Second))},
			{Model: models.Model{ID: 2}},
			{Model: models.Model{ID: 3}},
		}
		db.Create(&input)
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/customer", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		returnedCustomers := []*customers.Customer{}
		json.Unmarshal([]byte(w.Body.String()), &returnedCustomers)
		assert.Equal(t, 2, len(returnedCustomers))
		assert.Assert(t, returnedCustomers[0].ExitAt == nil)
		assert.Assert(t, returnedCustomers[1].ExitAt == nil)
	})
}

func TestCustomerExit(t *testing.T) {
	t.Run("expected to mark the customer as exited with exit time", func(t *testing.T) {
		db := testDB(t.Name())
		db.Create(&customers.Customer{})
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("id", "1")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/exit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"customer_id":1,"status":"updated"}`, w.Body.String())

		customer := &customers.Customer{}
		db.Table(customers.TableName).First(&customer)
		assert.Assert(t, customer.ExitAt.Before(time.Now()))
	})

	t.Run("error when customer not found", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("id", "1")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/exit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
		assert.Equal(t, `{"err":"customer record not found"}`, w.Body.String())
	})
}

func TestCustomerQueued(t *testing.T) {
	t.Run("expected to error on missing params", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/queue", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		assert.Equal(
			t,
			`{"err":"Key: 'queueForm.ID' Error:Field validation for 'ID' failed on the 'required' tag\nKey: 'queueForm.RideID' Error:Field validation for 'RideID' failed on the 'required' tag"}`,
			w.Body.String(),
		)
	})

	t.Run("expected to log customer as entered a ride queue", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{}
		db.Create(customer)
		db.Create(&rides.Ride{Name: "ride1", Capacity: 10, RideTime: 10 * time.Minute})
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("id", "1")
		form.Add("ride_id", "1")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/queue", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"customer_id":1,"status":"queued"}`, w.Body.String())

		state, err := customerEvents.GetCurrentState(db, customer)
		assert.NilError(t, err)
		assert.Equal(t, true, state.Queueing)
		assert.Equal(t, uint(1), state.RideID)
		assert.Assert(t, state.From.Before(time.Now()))
		assert.Assert(t, state.To.After(time.Now()))
	})
}

func TestCustomerUnQueued(t *testing.T) {
	t.Run("expected to error on missing params", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/unqueue", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		assert.Equal(
			t,
			`{"err":"Key: 'unqueueForm.ID' Error:Field validation for 'ID' failed on the 'required' tag"}`,
			w.Body.String(),
		)
	})

	t.Run("error on unqueuing a not queued customer", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{Model: models.Model{ID: 122}}
		db.Create(customer)
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("id", "122")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/unqueue", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)
		assert.Equal(t, `{"err":"un-queue Customer is not in any queue"}`, w.Body.String())
	})

	t.Run("expected to log customer as exited a ride queue", func(t *testing.T) {
		db := testDB(t.Name())
		customer := &customers.Customer{Model: models.Model{ID: 123}}
		db.Create(customer)
		ride := &rides.Ride{Name: "ride1", Capacity: 10, RideTime: 10 * time.Minute}
		db.Create(ride)
		customerEvents.LogCustomerInQueue(db, customer, ride)
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("id", "123")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/unqueue", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"customer_id":123,"status":"un-queued"}`, w.Body.String())

		state, err := customerEvents.GetCurrentState(db, customer)
		assert.NilError(t, err)
		assert.Equal(t, false, state.Queueing)
		assert.Equal(t, uint(0), state.RideID)
	})
}
