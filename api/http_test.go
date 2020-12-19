package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/api/customers"
	"gitlab.com/therako/universal-studios/api/rides"
	"gitlab.com/therako/universal-studios/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gotest.tools/v3/assert"
)

var (
	testConfig = api.Config{HTTPPort: 8081}
	gormLogger = logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testDB(name string) *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory", name)), &gorm.Config{
		Logger: gormLogger,
	})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&rides.Ride{})
	gormDB.AutoMigrate(&customers.Customer{})
	return gormDB
}

func TestRideEndpoints(t *testing.T) {
	t.Run("/ride expected to return all rides stores in DB", func(t *testing.T) {
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

	t.Run("/ride/add", func(t *testing.T) {
		t.Run("expected to add ride to DB when all values are present", func(t *testing.T) {
			db := testDB(t.Name())
			router := api.New(context.Background(), testConfig, db)
			form := url.Values{}
			form.Add("name", "DareDevil")
			form.Add("desc", "Booooo")
			form.Add("ride_time_secs", "300")

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
		})

		t.Run("error on missing input", func(t *testing.T) {
			db := testDB(t.Name())
			router := api.New(context.Background(), testConfig, db)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/ride/add", nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			router.ServeHTTP(w, req)

			assert.Equal(t, 400, w.Code)
			assert.Equal(t, `{"err":"Invalid request input. Expected atleast name \u0026 ride_time_secs"}`, w.Body.String())
		})
	})
}

func TestCustomerEndpoints(t *testing.T) {
	t.Run("/customer/enter expected to create a new customer on entry", func(t *testing.T) {
		db := testDB(t.Name())
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/customer/enter", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"status":"added"}`, w.Body.String())

		customer := &customers.Customer{}
		db.Table(customers.TableName).First(&customer)
		assert.Assert(t, customer.Model.ID == 1)
		assert.Assert(t, customer.Model.CreatedAt.Before(time.Now()))
		assert.Assert(t, customer.ExitAt == nil)
	})

	t.Run("/customer expected to return all customer who are inside the studio as of now", func(t *testing.T) {
		db := testDB(t.Name())
		input := []*customers.Customer{
			{Model: models.Model{ID: 1}, ExitAt: models.Timep(time.Now().Add(-1 * time.Second))},
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

	t.Run("/customer/exit", func(t *testing.T) {
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
			assert.Equal(t, `{"status":"updated"}`, w.Body.String())

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
			assert.Equal(t, `{"err":"Customer not found"}`, w.Body.String())
		})
	})
}
