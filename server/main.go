package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration structure.
type Config struct {
	BaseURL        string        `yaml:"base_url"`
	TokenExpiry    time.Duration `yaml:"token_expiry"`
}

// MemoryStorage is an in-memory store for lists and records.
type MemoryStorage struct {
	sync.Mutex
	Lists   map[int64]VehicleList
	Records map[int64][]Record
	Tokens  map[string]struct {
		Expiry time.Time
		ID     int64
	}
}

// VehicleList represents a vehicle list.
type VehicleList struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Order       int    `json:"order"`
	Status      int    `json:"status"`
}

// Record represents a record in a vehicle list.
type Record struct {
	ID          int64  `json:"id"`
	Plate       string `json:"plate"`
	VehicleType string `json:"vehicleType"`
}

var (
	storage MemoryStorage
	baseURL  string
	tokenExpiry time.Duration
)

func main() {
	// Parse flags and environment variables.
	defaultURL := "http://localhost:1608"
	defaultExpiry := 5 * time.Minute
	configFile := flag.String("config", "kpam.yaml", "Path to configuration file")
	flag.Parse()

	if envURL := os.Getenv("KPAM_URL"); envURL != "" {
		baseURL = envURL
	} else if config, err := readConfig(*configFile); err == nil {
		baseURL = config.BaseURL
		tokenExpiry = config.TokenExpiry
	} else {
		baseURL = defaultURL
		tokenExpiry = defaultExpiry
	}

	// Initialize storage.
	storage = MemoryStorage{
		Lists:   make(map[int64]VehicleList),
		Records: make(map[int64][]Record),
		Tokens:  make(map[string]struct {
			Expiry time.Time
			ID     int64
		}),
	}

	http.HandleFunc("/login", loginHandler)
	http.Handle("/api/v1/vehiclelists", tokenMiddleware(http.HandlerFunc(vehicleListsHandler)))
	http.Handle("/api/v1/vehiclelist/record", tokenMiddleware(recordMiddleware(http.HandlerFunc(recordHandler))))

	log.Printf("Starting server at %s\n", baseURL)
	log.Fatal(http.ListenAndServe(baseURL[len("http://"):], nil))
}

func readConfig(path string) (*Config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}
	if config.TokenExpiry == 0 {
		config.TokenExpiry = 5 * time.Minute
	}
	return &config, nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		IsRememberMe  bool   `json:"isRememberMe"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	id := time.Now().UnixNano() // Example ID generation for user session
	token := generateToken(id)
	expiry := time.Now().Add(tokenExpiry)
	storage.Lock()
	storage.Tokens[token] = struct {
		Expiry time.Time
		ID     int64
	}{
		Expiry: expiry,
		ID:     id,
	}
	storage.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:    "s",
		Value:   token,
		Expires: expiry,
	})

	response := map[string]interface{}{
		"redirectUrl": "/",
		"isAuthorized": true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func vehicleListsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Handle GET method for vehicle lists.
		offset, count := 0, 20 // Default values

		lists := []VehicleList{}
		storage.Lock()
		for _, list := range storage.Lists {
			lists = append(lists, list)
		}
		storage.Unlock()

		response := map[string]interface{}{
			"entries":   lists,
			"_metadata": map[string]int{"offset": offset, "limit": count, "totalCount": len(lists)},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func recordHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetRecord(w, r)
	case http.MethodPost:
		handlePostRecord(w, r)
	case http.MethodDelete:
		handleDeleteRecord(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func recordMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		// Pass ID as context value
		r = r.WithContext(contextWithID(r.Context(), id))
		next.ServeHTTP(w, r)
	})
}

func handleGetRecord(w http.ResponseWriter, r *http.Request) {
	id := contextID(r.Context())
	storage.Lock()
	records, exists := storage.Records[id]
	storage.Unlock()

	if !exists {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"entries": records,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePostRecord(w http.ResponseWriter, r *http.Request) {
	id := contextID(r.Context())
	var record Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	storage.Lock()
	storage.Records[id] = append(storage.Records[id], record)
	storage.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	id := contextID(r.Context())
	recordIDStr := r.URL.Query().Get("recordId")
	if recordIDStr == "" {
		http.Error(w, "Missing recordId parameter", http.StatusBadRequest)
		return
	}
	recordID, err := strconv.ParseInt(recordIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid recordId parameter", http.StatusBadRequest)
		return
	}

	storage.Lock()
	records, exists := storage.Records[id]
	if !exists {
		storage.Unlock()
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}

	for i, rec := range records {
		if rec.ID == recordID {
			storage.Records[id] = append(records[:i], records[i+1:]...)
			storage.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	storage.Unlock()
	http.Error(w, "Record not found", http.StatusNotFound)
}

// Context helpers for passing ID

func contextWithID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, "id", id)
}

func contextID(ctx context.Context) int64 {
	if id, ok := ctx.Value("id").(int64); ok {
		return id
	}
	return 0
}

func tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("s")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		storage.Lock()
		data, exists := storage.Tokens[cookie.Value]
		storage.Unlock()

		if !exists || time.Now().After(data.Expiry) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r.Header.Set("User-ID", strconv.FormatInt(data.ID, 10))
		next.ServeHTTP(w, r)
	})
}

func generateToken(id int64) string {
	return fmt.Sprintf("%d-%d", id, time.Now().UnixNano())
}
