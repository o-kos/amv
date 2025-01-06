package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestLoginHandler(t *testing.T) {
	// Prepare a test server
	reqBody := `{"username":"test","password":"password","isRememberMe":false}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(reqBody)))
	w := httptest.NewRecorder()

	loginHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	// Validate the token cookie
	cookie := resp.Cookies()
	if len(cookie) == 0 || cookie[0].Name != "s" {
		t.Error("expected a token cookie named 's'")
	}
}

func TestVehicleListsHandler(t *testing.T) {
	// Mock storage
	storage.Lists = map[int64]VehicleList{
		1: {ID: 1, DisplayName: "Test List", Name: "testList"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehiclelists", nil)
	w := httptest.NewRecorder()

	vehicleListsHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	entries := result["entries"].([]interface{})
	if len(entries) != 1 {
		t.Errorf("expected 1 list, got %d", len(entries))
	}
}

func TestRecordMiddleware(t *testing.T) {
	// Mock request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehiclelist/record?id=1", nil)
	w := httptest.NewRecorder()

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		id := contextID(r.Context())
		if id != 1 {
			t.Errorf("expected id 1, got %d", id)
		}
	})

	recordMiddleware(handler).ServeHTTP(w, req)

	if !called {
		t.Error("middleware did not call next handler")
	}
}

func TestHandleGetRecord(t *testing.T) {
	// Mock storage
	id := int64(1)
	record := Record{ID: 100, Plate: "ABC123", VehicleType: "Car"}
	storage.Records = map[int64][]Record{
		id: {record},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehiclelist/record?id=1", nil)
	req = req.WithContext(contextWithID(req.Context(), id))
	w := httptest.NewRecorder()

	handleGetRecord(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	var result map[string][]Record
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if len(result["entries"]) != 1 || result["entries"][0].ID != record.ID {
		t.Errorf("unexpected record returned: %v", result["entries"])
	}
}

func TestHandlePostRecord(t *testing.T) {
	// Mock storage
	id := int64(1)
	storage.Records = map[int64][]Record{
		id: {},
	}

	reqBody := `{"id":101,"plate":"XYZ789","vehicleType":"Truck"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehiclelist/record?id=1", bytes.NewReader([]byte(reqBody)))
	req = req.WithContext(contextWithID(req.Context(), id))
	w := httptest.NewRecorder()

	handlePostRecord(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status Created, got %v", resp.StatusCode)
	}

	// Verify record added
	if len(storage.Records[id]) != 1 || storage.Records[id][0].Plate != "XYZ789" {
		t.Errorf("record not added correctly: %v", storage.Records[id])
	}
}

func TestHandleDeleteRecord(t *testing.T) {
	// Mock storage
	id := int64(1)
	record := Record{ID: 100, Plate: "ABC123", VehicleType: "Car"}
	storage.Records = map[int64][]Record{
		id: {record},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vehiclelist/record?id=1&recordId=100", nil)
	req = req.WithContext(contextWithID(req.Context(), id))
	w := httptest.NewRecorder()

	handleDeleteRecord(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	// Verify record deleted
	if len(storage.Records[id]) != 0 {
		t.Errorf("record not deleted correctly: %v", storage.Records[id])
	}
}
