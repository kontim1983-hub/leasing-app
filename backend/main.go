package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "modernc.org/sqlite"
)

var db *sql.DB
var uploadedFiles []string
var uploadedFilesV2 []string
var uploadedFilesV3 []string
var filesMutex sync.RWMutex
var shutdownFunc func()

func Shutdown() {
	log.Println("Shutdown signal received")
	if shutdownFunc != nil {
		shutdownFunc()
	}
	os.Exit(0)
}

func main() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	if dataEnv := os.Getenv("DATA_DIR"); dataEnv != "" {
		if filepath.IsAbs(dataEnv) {
			os.MkdirAll(dataEnv, 0755)
		} else {
			os.MkdirAll(filepath.Join(exeDir, dataEnv), 0755)
		}
	}

	dataDir := filepath.Join(exeDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	dbPath := filepath.Join(dataDir, "leasing.db")
	connStr := dbPath

	log.Println("Data directory:", dataDir)
	log.Println("Database:", dbPath)

	var err error
	db, err = sql.Open("sqlite", connStr)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	initDB()

	r := mux.NewRouter()

	r.HandleFunc("/api/shutdown", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})
		go func() {
			log.Println("Shutting down server...")
			if db != nil {
				db.Close()
			}
			os.Exit(0)
		}()
	}).Methods("POST")

	RegisterV1Routes(r)
	RegisterV2Routes(r)
	RegisterV3Routes(r)

	frontendDir := filepath.Join(exeDir, "frontend", "build")
	if _, err := os.Stat(frontendDir); err != nil {
		frontendDir = filepath.Join(exeDir, "frontend")
	}
	if _, err := os.Stat(frontendDir); err == nil {
		r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(frontendDir))))
		log.Println("Frontend static files:", frontendDir)
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func initDB() {
	query := `
    CREATE TABLE IF NOT EXISTS leasing_records (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       subject TEXT,
       location TEXT,
       subject_type TEXT,
       vehicle_type TEXT,
       vin TEXT UNIQUE NOT NULL,
       year TEXT,
       mileage TEXT,
       days_on_sale TEXT,
       approved_price TEXT,
       old_price TEXT,
       status TEXT,
       photos TEXT,
       is_new INTEGER DEFAULT 0,
       changed_columns TEXT,
       created_at TEXT DEFAULT CURRENT_TIMESTAMP,
       updated_at TEXT DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	query2 := `
    CREATE TABLE IF NOT EXISTS leasing_records_v2 (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       brand TEXT,
       model TEXT,
       vin TEXT UNIQUE NOT NULL,
       exposure_period TEXT,
       vehicle_type TEXT,
       vehicle_subtype TEXT,
       year TEXT,
       mileage TEXT,
       city TEXT,
       actual_price TEXT,
       old_price TEXT,
       status TEXT,
       photos TEXT,
       is_new INTEGER DEFAULT 0,
       changed_columns TEXT,
       created_at TEXT DEFAULT CURRENT_TIMESTAMP,
       updated_at TEXT DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err = db.Exec(query2)
	if err != nil {
		log.Fatal("Failed to create table v2:", err)
	}

	query3 := `
    CREATE TABLE IF NOT EXISTS leasing_records_v3 (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       brand TEXT,
       model TEXT,
       vin TEXT UNIQUE NOT NULL,
       exposure_period TEXT,
       vehicle_type TEXT,
       vehicle_subtype TEXT,
       year TEXT,
       mileage TEXT,
       city TEXT,
       actual_price TEXT,
       old_price TEXT,
       status TEXT,
       photos TEXT,
       is_new INTEGER DEFAULT 0,
       changed_columns TEXT,
       created_at TEXT DEFAULT CURRENT_TIMESTAMP,
       updated_at TEXT DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err = db.Exec(query3)
	if err != nil {
		log.Fatal("Failed to create table v3:", err)
	}
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}
