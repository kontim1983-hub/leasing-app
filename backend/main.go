package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var db *sql.DB
var uploadedFiles []string
var uploadedFilesV2 []string
var filesMutex sync.RWMutex

func main() {
	var err error
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "leasing")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... attempt %d/10", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	initDB()

	r := mux.NewRouter()

	// V1 routes (первая вкладка)
	RegisterV1Routes(r)

	// V2 routes (вторая вкладка)
	RegisterV2Routes(r)

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
       id SERIAL PRIMARY KEY,
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
       photos TEXT[],
       is_new BOOLEAN DEFAULT false,
       changed_columns TEXT[],
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	query2 := `
    CREATE TABLE IF NOT EXISTS leasing_records_v2 (
       id SERIAL PRIMARY KEY,
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
       photos TEXT[],
       is_new BOOLEAN DEFAULT false,
       changed_columns TEXT[],
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err = db.Exec(query2)
	if err != nil {
		log.Fatal("Failed to create table v2:", err)
	}
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}
