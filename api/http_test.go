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

	"github.com/gin-gonic/gin"
	"gitlab.com/therako/universal-studios/api"
	"gitlab.com/therako/universal-studios/api/rides"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gotest.tools/v3/assert"
)

var (
	testConfig = api.Config{HTTPPort: 8081}
)

func init() {
	gin.SetMode("test")
}

func testDB() *gorm.DB {
	gormDB, _ := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	gormDB.Exec("PRAGMA foreign_keys = ON") // SQLite defaults to `foreign_keys = off'`
	gormDB.AutoMigrate(&rides.Ride{})
	return gormDB
}

func TestListRides(t *testing.T) {
	t.Run("Expect to return all rides stores in DB", func(t *testing.T) {
		db := testDB()
		rides := []*rides.Ride{
			{Name: "RollerCoster", Desc: "World's best roller cosater", RideTime: 4 * time.Minute},
			{Name: "BumperCar", Desc: "Bump all the way", RideTime: 7 * time.Minute},
		}
		db.Create(&rides)
		router := api.New(context.Background(), testConfig, db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/rides", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		ridesStr, _ := json.Marshal(rides)
		assert.Equal(t, string(ridesStr), w.Body.String())
	})
}

func TestAddRide(t *testing.T) {
	t.Run("Expect to add ride to DB", func(t *testing.T) {
		db := testDB()
		router := api.New(context.Background(), testConfig, db)
		form := url.Values{}
		form.Add("name", "DareDevil")
		form.Add("desc", "Booooo")
		form.Add("ride_time_secs", "300")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/rides/add", strings.NewReader(form.Encode()))
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
}
