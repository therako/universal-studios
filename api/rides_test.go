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
	"gitlab.com/therako/universal-studios/data/rides"
	"gotest.tools/v3/assert"
)

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
	})
}
