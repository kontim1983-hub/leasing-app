package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/xuri/excelize/v2"
)

type LeasingRecord struct {
	ID             int      `json:"id"`
	Subject        string   `json:"subject"`
	Location       string   `json:"location"`
	SubjectType    string   `json:"subject_type"`
	VehicleType    string   `json:"vehicle_type"`
	VIN            string   `json:"vin"`
	Year           string   `json:"year"`
	Mileage        string   `json:"mileage"`
	DaysOnSale     string   `json:"days_on_sale"`
	ApprovedPrice  string   `json:"approved_price"`
	OldPrice       string   `json:"old_price,omitempty"`
	Status         string   `json:"status"`
	Photos         []string `json:"photos"`
	IsNew          bool     `json:"is_new"`
	ChangedColumns []string `json:"changed_columns,omitempty"`
}

var db *sql.DB

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

	// Существующие маршруты
	r.HandleFunc("/api/upload", uploadHandler).Methods("POST")
	r.HandleFunc("/api/records", getRecordsHandler).Methods("GET")
	r.HandleFunc("/api/health", healthHandler).Methods("GET")

	// === НОВЫЕ АДМИН-МАРШРУТЫ ===
	r.HandleFunc("/api/clear-changed-columns", clearChangedColumnsHandler).Methods("POST")
	r.HandleFunc("/api/delete-all-records", deleteAllRecordsHandler).Methods("POST")

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
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Новый обработчик: очистка changed_columns во всех записях
func clearChangedColumnsHandler(w http.ResponseWriter, r *http.Request) {
	result, err := db.Exec(`UPDATE leasing_records SET changed_columns = '{}', updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		http.Error(w, "Failed to clear changed_columns", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Все значения в колонке changed_columns успешно очищены",
		"rows_affected": rowsAffected,
	})
}

// Новый обработчик: удаление всех записей из таблицы
func deleteAllRecordsHandler(w http.ResponseWriter, r *http.Request) {
	// Защита от случайного вызова — требуем подтверждение в теле запроса
	var payload struct {
		Confirm string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.Confirm != "delete" {
		http.Error(w, "To delete all records, send JSON: {\"confirm\": \"delete\"}", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`DELETE FROM leasing_records`)
	if err != nil {
		http.Error(w, "Failed to delete all records", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Все записи успешно удалены из базы",
		"rows_deleted": rowsAffected,
	})
}

// Остальные функции без изменений (оставлены как были)
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Failed to read Excel file", http.StatusBadRequest)
		return
	}
	defer f.Close()

	records, err := processExcel(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process Excel: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func processExcel(f *excelize.File) ([]LeasingRecord, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	var result []LeasingRecord

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		subject := getColumnValue(row, 1)
		location := getColumnValue(row, 29)
		subjectType := getColumnValue(row, 4)
		vehicleType := getColumnValue(row, 5)
		vin := getColumnValue(row, 6)
		year := getColumnValue(row, 10)
		mileage := getColumnValue(row, 11)
		daysOnSale := getColumnValue(row, 14)
		approvedPrice := getColumnValue(row, 16)
		status := getColumnValue(row, 39)

		if vin == "" {
			continue
		}

		if status != "В продаже" {
			deleteRecord(vin)
			continue
		}

		existing, exists := getRecordByVIN(vin)

		if !exists {
			record := LeasingRecord{
				Subject:       subject,
				Location:      location,
				SubjectType:   subjectType,
				VehicleType:   vehicleType,
				VIN:           vin,
				Year:          year,
				Mileage:       mileage,
				DaysOnSale:    daysOnSale,
				ApprovedPrice: approvedPrice,
				Status:        status,
				Photos:        searchPhotos(vin),
				IsNew:         true,
			}

			id, err := insertRecord(record)
			if err != nil {
				log.Printf("Failed to insert record: %v", err)
				continue
			}
			record.ID = id
			result = append(result, record)
		} else {
			changed := compareRecords(existing, LeasingRecord{
				Subject:       subject,
				Location:      location,
				SubjectType:   subjectType,
				VehicleType:   vehicleType,
				VIN:           vin,
				Year:          year,
				Mileage:       mileage,
				DaysOnSale:    daysOnSale,
				ApprovedPrice: approvedPrice,
				Status:        status,
			})

			if len(changed) == 0 {
				continue
			}

			var oldPrice string
			for _, col := range changed {
				if col == "approved_price" {
					oldPrice = existing.ApprovedPrice
					break
				}
			}

			record := LeasingRecord{
				ID:             existing.ID,
				Subject:        subject,
				Location:       location,
				SubjectType:    subjectType,
				VehicleType:    vehicleType,
				VIN:            vin,
				Year:           year,
				Mileage:        mileage,
				DaysOnSale:     daysOnSale,
				ApprovedPrice:  approvedPrice,
				OldPrice:       oldPrice,
				Status:         status,
				Photos:         existing.Photos,
				IsNew:          false,
				ChangedColumns: changed,
			}

			err := updateRecord(record)
			if err != nil {
				log.Printf("Failed to update record: %v", err)
				continue
			}

			result = append(result, record)
		}
	}

	return result, nil
}

func getColumnValue(row []string, index int) string {
	if index < len(row) {
		return row[index]
	}
	return ""
}

func compareRecords(old, new LeasingRecord) []string {
	var changed []string
	if old.Subject != new.Subject {
		changed = append(changed, "subject")
	}
	if old.SubjectType != new.SubjectType {
		changed = append(changed, "subject_type")
	}
	if old.VehicleType != new.VehicleType {
		changed = append(changed, "vehicle_type")
	}
	if old.Mileage != new.Mileage {
		changed = append(changed, "mileage")
	}
	if old.ApprovedPrice != new.ApprovedPrice {
		changed = append(changed, "approved_price")
	}
	if old.Status != new.Status {
		changed = append(changed, "status")
	}
	return changed
}

func getRecordByVIN(vin string) (LeasingRecord, bool) {
	var rec LeasingRecord

	var subject, location, subjectType, vehicleType sql.NullString
	var year, mileage, daysOnSale, approvedPrice, oldPrice, status sql.NullString

	err := db.QueryRow(`
       SELECT id, subject, location, subject_type, vehicle_type, vin,
              year, mileage, days_on_sale, approved_price, old_price,
              status, COALESCE(photos, '{}'), is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records WHERE vin=$1
    `, vin).Scan(
		&rec.ID,
		&subject, &location, &subjectType, &vehicleType, &rec.VIN,
		&year, &mileage, &daysOnSale, &approvedPrice,
		&oldPrice, &status,
		pq.Array(&rec.Photos), &rec.IsNew, pq.Array(&rec.ChangedColumns),
	)
	if err != nil {
		return rec, false
	}

	// Преобразуем NullString → string (NULL становится "")
	rec.Subject = nullStringToString(subject)
	rec.Location = nullStringToString(location)
	rec.SubjectType = nullStringToString(subjectType)
	rec.VehicleType = nullStringToString(vehicleType)
	rec.Year = nullStringToString(year)
	rec.Mileage = nullStringToString(mileage)
	rec.DaysOnSale = nullStringToString(daysOnSale)
	rec.ApprovedPrice = nullStringToString(approvedPrice)
	rec.OldPrice = nullStringToString(oldPrice)
	rec.Status = nullStringToString(status)

	return rec, true
}

func insertRecord(record LeasingRecord) (int, error) {
	var id int
	err := db.QueryRow(`
       INSERT INTO leasing_records
       (subject, location, subject_type, vehicle_type, vin, year, mileage, days_on_sale,
        approved_price, old_price, status, photos, is_new, changed_columns)
       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
       RETURNING id
    `,
		record.Subject, record.Location, record.SubjectType, record.VehicleType, record.VIN,
		record.Year, record.Mileage, record.DaysOnSale, record.ApprovedPrice,
		record.OldPrice, record.Status, pq.Array(record.Photos), record.IsNew,
		pq.Array(record.ChangedColumns),
	).Scan(&id)
	return id, err
}

func updateRecord(record LeasingRecord) error {
	_, err := db.Exec(`
       UPDATE leasing_records SET
          subject=$1, location=$2, subject_type=$3, vehicle_type=$4,
          year=$5, mileage=$6, days_on_sale=$7,
          approved_price=$8, old_price=$9, status=$10,
          is_new=$11, changed_columns=$12, updated_at=CURRENT_TIMESTAMP
       WHERE vin=$13
    `,
		record.Subject, record.Location, record.SubjectType, record.VehicleType, record.Year,
		record.Mileage, record.DaysOnSale, record.ApprovedPrice, record.OldPrice,
		record.Status, record.IsNew, pq.Array(record.ChangedColumns), record.VIN,
	)
	return err
}

func deleteRecord(vin string) {
	db.Exec("DELETE FROM leasing_records WHERE vin=$1", vin)
}

func searchPhotos(vin string) []string {
	return []string{}
}

func getRecordsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
       SELECT id, subject, location, subject_type, vehicle_type, vin,
              year, mileage, days_on_sale, approved_price, old_price,
              status, COALESCE(photos, '{}'), is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records ORDER BY updated_at DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var records []LeasingRecord

	for rows.Next() {
		var r LeasingRecord
		var photos []string
		var changedCols []string

		var subject, location, subjectType, vehicleType sql.NullString
		var year, mileage, daysOnSale, approvedPrice, oldPrice, status sql.NullString

		err := rows.Scan(
			&r.ID,
			&subject, &location, &subjectType, &vehicleType, &r.VIN,
			&year, &mileage, &daysOnSale, &approvedPrice,
			&oldPrice, &status,
			pq.Array(&photos), &r.IsNew, pq.Array(&changedCols),
		)
		if err != nil {
			log.Println("Failed scan:", err)
			continue
		}

		if len(changedCols) == 0 {
			continue
		}

		// Преобразуем NULL → ""
		r.Subject = nullStringToString(subject)
		r.Location = nullStringToString(location)
		r.SubjectType = nullStringToString(subjectType)
		r.VehicleType = nullStringToString(vehicleType)
		r.Year = nullStringToString(year)
		r.Mileage = nullStringToString(mileage)
		r.DaysOnSale = nullStringToString(daysOnSale)
		r.ApprovedPrice = nullStringToString(approvedPrice)
		r.OldPrice = nullStringToString(oldPrice)
		r.Status = nullStringToString(status)

		r.Photos = photos
		r.ChangedColumns = changedCols
		records = append(records, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}
