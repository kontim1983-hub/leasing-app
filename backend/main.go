package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
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

type LeasingRecordV2 struct {
	ID             int      `json:"id"`
	Brand          string   `json:"brand"`
	Model          string   `json:"model"`
	VIN            string   `json:"vin"`
	ExposurePeriod string   `json:"exposure_period"`
	VehicleType    string   `json:"vehicle_type"`
	VehicleSubtype string   `json:"vehicle_subtype"`
	Year           string   `json:"year"`
	Mileage        string   `json:"mileage"`
	City           string   `json:"city"`
	ActualPrice    string   `json:"actual_price"`
	OldPrice       string   `json:"old_price,omitempty"`
	Status         string   `json:"status"`
	Photos         []string `json:"photos"`
	IsNew          bool     `json:"is_new"`
	ChangedColumns []string `json:"changed_columns,omitempty"`
}

var db *sql.DB
var uploadedFiles []string
var uploadedFilesV2 []string
var filesMutex sync.RWMutex // Защита от race condition

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

	r.HandleFunc("/api/upload", uploadHandler).Methods("POST")
	r.HandleFunc("/api/records", getRecordsHandler).Methods("GET")
	r.HandleFunc("/api/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		filesMutex.RLock()
		defer filesMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadedFiles)
	}).Methods("GET")
	r.HandleFunc("/api/v2/files", func(w http.ResponseWriter, r *http.Request) {
		filesMutex.RLock()
		defer filesMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadedFilesV2)
	}).Methods("GET")

	r.HandleFunc("/api/clear-changed-columns", clearChangedColumnsHandler).Methods("POST")
	r.HandleFunc("/api/delete-all-records", deleteAllRecordsHandler).Methods("POST")

	r.HandleFunc("/api/v2/upload", uploadHandlerV2).Methods("POST")
	r.HandleFunc("/api/v2/records", getRecordsHandlerV2).Methods("GET")
	r.HandleFunc("/api/v2/clear-changed-columns", clearChangedColumnsHandlerV2).Methods("POST")
	r.HandleFunc("/api/v2/delete-all-records", deleteAllRecordsHandlerV2).Methods("POST")
	r.HandleFunc("/api/v2/export", exportExcelHandlerV2).Methods("GET")

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

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

func deleteAllRecordsHandler(w http.ResponseWriter, r *http.Request) {
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

	filesMutex.Lock()
	uploadedFiles = []string{}
	filesMutex.Unlock()

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Все записи успешно удалены из базы",
		"rows_deleted": rowsAffected,
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := header.Filename

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

	filesMutex.Lock()
	exists := false
	for _, fn := range uploadedFiles {
		if fn == filename {
			exists = true
			break
		}
	}
	if !exists {
		uploadedFiles = append(uploadedFiles, filename)
	}
	filesCopy := make([]string, len(uploadedFiles))
	copy(filesCopy, uploadedFiles)
	filesMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records":   records,
		"file_name": filename,
		"files":     filesCopy,
	})
}

func processExcel(f *excelize.File) ([]LeasingRecord, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file must have at least header and one data row")
	}

	// ИСПРАВЛЕНО: инициализируем пустым массивом
	result := make([]LeasingRecord, 0)

	// ИСПРАВЛЕНО: используем буквенные обозначения столбцов Excel
	// Начинаем с rowNum=2 (пропускаем заголовок), rowNum соответствует строке в Excel (1-based)
	for rowNum := 2; rowNum <= len(rows); rowNum++ {
		// Маппинг согласно вашей структуре (нужно уточнить реальные столбцы)
		// Пример для корректного извлечения:
		subject := getCellValueByColumn(f, sheetName, "B", rowNum)       // Столбец 2
		location := getCellValueByColumn(f, sheetName, "AD", rowNum)     // Столбец 30
		subjectType := getCellValueByColumn(f, sheetName, "E", rowNum)   // Столбец 5
		vehicleType := getCellValueByColumn(f, sheetName, "F", rowNum)   // Столбец 6
		vin := getCellValueByColumn(f, sheetName, "G", rowNum)           // Столбец 7
		year := getCellValueByColumn(f, sheetName, "K", rowNum)          // Столбец 11
		mileage := getCellValueByColumn(f, sheetName, "L", rowNum)       // Столбец 12
		daysOnSale := getCellValueByColumn(f, sheetName, "O", rowNum)    // Столбец 15
		approvedPrice := getCellValueByColumn(f, sheetName, "Q", rowNum) // Столбец 17
		status := getCellValueByColumn(f, sheetName, "AN", rowNum)       // Столбец 40

		if vin == "" {
			continue
		}

		if status != "В продаже" {
			deleteRecord(vin)
			continue
		}

		existing, exists := getRecordByVIN(vin)

		if !exists {
			photos := searchPhotos(vin)
			if photos == nil {
				photos = []string{}
			}

			record := LeasingRecord{
				Subject:        subject,
				Location:       location,
				SubjectType:    subjectType,
				VehicleType:    vehicleType,
				VIN:            vin,
				Year:           year,
				Mileage:        mileage,
				DaysOnSale:     daysOnSale,
				ApprovedPrice:  approvedPrice,
				Status:         status,
				Photos:         photos,
				IsNew:          true,
				ChangedColumns: []string{}, // ИСПРАВЛЕНО: инициализируем пустым массивом
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

			photos := existing.Photos
			if photos == nil {
				photos = []string{}
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
				Photos:         photos,
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

func getCellValueByColumn(f *excelize.File, sheetName string, colLetter string, rowNum int) string {
	cellName := fmt.Sprintf("%s%d", colLetter, rowNum)
	value, err := f.GetCellValue(sheetName, cellName)
	if err != nil {
		return ""
	}
	return value
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
	// ИСПРАВЛЕНО: возвращаем пустой массив вместо nil
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

	// ИСПРАВЛЕНО: инициализируем пустым массивом вместо nil
	records := make([]LeasingRecord, 0)

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

		// ИСПРАВЛЕНО: инициализируем пустыми массивами если nil
		if r.Photos == nil {
			r.Photos = []string{}
		}
		if r.ChangedColumns == nil {
			r.ChangedColumns = []string{}
		}

		if r.IsNew || len(r.ChangedColumns) > 0 {
			records = append(records, r)
		}
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

// ========== V2 HANDLERS ==========

func uploadHandlerV2(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := header.Filename

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Failed to read Excel file", http.StatusBadRequest)
		return
	}
	defer f.Close()

	records, err := processExcelV2(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process Excel: %v", err), http.StatusInternalServerError)
		return
	}

	filesMutex.Lock()
	exists := false
	for _, fn := range uploadedFilesV2 {
		if fn == filename {
			exists = true
			break
		}
	}
	if !exists {
		uploadedFilesV2 = append(uploadedFilesV2, filename)
	}
	filesCopy := make([]string, len(uploadedFilesV2))
	copy(filesCopy, uploadedFilesV2)
	filesMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records":   records,
		"file_name": filename,
		"files":     filesCopy,
	})
}

func processExcelV2(f *excelize.File) ([]LeasingRecordV2, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file must have at least header and one data row")
	}

	// ИСПРАВЛЕНО: инициализируем пустым массивом
	result := make([]LeasingRecordV2, 0)

	// ИСПРАВЛЕНО: убрано +1, теперь корректное количество итераций
	for rowNum := 2; rowNum <= len(rows); rowNum++ {
		brand := getCellValueByColumn(f, sheetName, "K", rowNum)
		model := getCellValueByColumn(f, sheetName, "L", rowNum)
		vin := getCellValueByColumn(f, sheetName, "F", rowNum)
		exposurePeriod := getCellValueByColumn(f, sheetName, "E", rowNum)
		vehicleType := getCellValueByColumn(f, sheetName, "H", rowNum)
		vehicleSubtype := getCellValueByColumn(f, sheetName, "I", rowNum)
		year := getCellValueByColumn(f, sheetName, "R", rowNum)
		mileage := getCellValueByColumn(f, sheetName, "BA", rowNum)
		city := getCellValueByColumn(f, sheetName, "P", rowNum)
		actualPrice := getCellValueByColumn(f, sheetName, "N", rowNum)
		status := getCellValueByColumn(f, sheetName, "C", rowNum)

		if status == "" {
			status = "В продаже"
		}

		if vin == "" {
			continue
		}

		if status != "" && status != "В продаже" {
			deleteRecordV2(vin)
			continue
		}

		existing, exists := getRecordByVINV2(vin)

		if !exists {
			photos := searchPhotos(vin)
			if photos == nil {
				photos = []string{}
			}

			record := LeasingRecordV2{
				Brand:          brand,
				Model:          model,
				VIN:            vin,
				ExposurePeriod: exposurePeriod,
				VehicleType:    vehicleType,
				VehicleSubtype: vehicleSubtype,
				Year:           year,
				Mileage:        mileage,
				City:           city,
				ActualPrice:    actualPrice,
				Status:         status,
				Photos:         photos,
				IsNew:          true,
				ChangedColumns: []string{}, // ИСПРАВЛЕНО: инициализируем пустым массивом
			}

			id, err := insertRecordV2(record)
			if err != nil {
				log.Printf("Failed to insert record v2: %v", err)
				continue
			}
			record.ID = id
			result = append(result, record)
		} else {
			changed := compareRecordsV2(existing, LeasingRecordV2{
				Brand:          brand,
				Model:          model,
				VIN:            vin,
				ExposurePeriod: exposurePeriod,
				VehicleType:    vehicleType,
				VehicleSubtype: vehicleSubtype,
				Year:           year,
				Mileage:        mileage,
				City:           city,
				ActualPrice:    actualPrice,
				Status:         status,
			})

			if len(changed) == 0 {
				continue
			}

			var oldPrice string
			for _, col := range changed {
				if col == "actual_price" {
					oldPrice = existing.ActualPrice
					break
				}
			}

			photos := existing.Photos
			if photos == nil {
				photos = []string{}
			}

			record := LeasingRecordV2{
				ID:             existing.ID,
				Brand:          brand,
				Model:          model,
				VIN:            vin,
				ExposurePeriod: exposurePeriod,
				VehicleType:    vehicleType,
				VehicleSubtype: vehicleSubtype,
				Year:           year,
				Mileage:        mileage,
				City:           city,
				ActualPrice:    actualPrice,
				OldPrice:       oldPrice,
				Status:         status,
				Photos:         photos,
				IsNew:          false,
				ChangedColumns: changed,
			}

			err := updateRecordV2(record)
			if err != nil {
				log.Printf("Failed to update record v2: %v", err)
				continue
			}

			result = append(result, record)
		}
	}

	return result, nil
}

func compareRecordsV2(old, new LeasingRecordV2) []string {
	var changed []string
	if old.Brand != new.Brand {
		changed = append(changed, "brand")
	}
	if old.Model != new.Model {
		changed = append(changed, "model")
	}
	if old.ExposurePeriod != new.ExposurePeriod {
		changed = append(changed, "exposure_period")
	}
	if old.VehicleType != new.VehicleType {
		changed = append(changed, "vehicle_type")
	}
	if old.VehicleSubtype != new.VehicleSubtype {
		changed = append(changed, "vehicle_subtype")
	}
	if old.Year != new.Year {
		changed = append(changed, "year")
	}
	if old.Mileage != new.Mileage {
		changed = append(changed, "mileage")
	}
	if old.City != new.City {
		changed = append(changed, "city")
	}
	if old.ActualPrice != new.ActualPrice {
		changed = append(changed, "actual_price")
	}
	if old.Status != new.Status {
		changed = append(changed, "status")
	}
	return changed
}

func getRecordByVINV2(vin string) (LeasingRecordV2, bool) {
	var rec LeasingRecordV2
	var brand, model, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
	var year, mileage, city, actualPrice, oldPrice, status sql.NullString

	err := db.QueryRow(`
       SELECT id, brand, model, vin, exposure_period, vehicle_type, vehicle_subtype,
              year, mileage, city, actual_price, old_price,
              status, COALESCE(photos, '{}'), is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records_v2 WHERE vin=$1
    `, vin).Scan(
		&rec.ID,
		&brand, &model, &rec.VIN, &exposurePeriod, &vehicleType, &vehicleSubtype,
		&year, &mileage, &city, &actualPrice,
		&oldPrice, &status,
		pq.Array(&rec.Photos), &rec.IsNew, pq.Array(&rec.ChangedColumns),
	)
	if err != nil {
		return rec, false
	}

	rec.Brand = nullStringToString(brand)
	rec.Model = nullStringToString(model)
	rec.ExposurePeriod = nullStringToString(exposurePeriod)
	rec.VehicleType = nullStringToString(vehicleType)
	rec.VehicleSubtype = nullStringToString(vehicleSubtype)
	rec.Year = nullStringToString(year)
	rec.Mileage = nullStringToString(mileage)
	rec.City = nullStringToString(city)
	rec.ActualPrice = nullStringToString(actualPrice)
	rec.OldPrice = nullStringToString(oldPrice)
	rec.Status = nullStringToString(status)

	return rec, true
}

func insertRecordV2(record LeasingRecordV2) (int, error) {
	var id int
	err := db.QueryRow(`
       INSERT INTO leasing_records_v2
       (brand, model, vin, exposure_period, vehicle_type, vehicle_subtype, year, mileage, city,
        actual_price, old_price, status, photos, is_new, changed_columns)
       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
       RETURNING id
    `,
		record.Brand, record.Model, record.VIN, record.ExposurePeriod, record.VehicleType,
		record.VehicleSubtype, record.Year, record.Mileage, record.City, record.ActualPrice,
		record.OldPrice, record.Status, pq.Array(record.Photos), record.IsNew,
		pq.Array(record.ChangedColumns),
	).Scan(&id)
	return id, err
}

func updateRecordV2(record LeasingRecordV2) error {
	_, err := db.Exec(`
       UPDATE leasing_records_v2 SET
          brand=$1, model=$2, exposure_period=$3, vehicle_type=$4, vehicle_subtype=$5,
          year=$6, mileage=$7, city=$8,
          actual_price=$9, old_price=$10, status=$11,
          is_new=$12, changed_columns=$13, updated_at=CURRENT_TIMESTAMP
       WHERE vin=$14
    `,
		record.Brand, record.Model, record.ExposurePeriod, record.VehicleType, record.VehicleSubtype,
		record.Year, record.Mileage, record.City, record.ActualPrice, record.OldPrice,
		record.Status, record.IsNew, pq.Array(record.ChangedColumns), record.VIN,
	)
	return err
}

func deleteRecordV2(vin string) {
	db.Exec("DELETE FROM leasing_records_v2 WHERE vin=$1", vin)
}

func getRecordsHandlerV2(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
       SELECT id, brand, model, vin, exposure_period, vehicle_type, vehicle_subtype,
              year, mileage, city, actual_price, old_price,
              status, COALESCE(photos, '{}'), is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records_v2 ORDER BY updated_at DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// ИСПРАВЛЕНО: инициализируем пустым массивом вместо nil
	records := make([]LeasingRecordV2, 0)

	for rows.Next() {
		var r LeasingRecordV2
		var photos []string
		var changedCols []string
		var brand, model, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
		var year, mileage, city, actualPrice, oldPrice, status sql.NullString

		err := rows.Scan(
			&r.ID,
			&brand, &model, &r.VIN, &exposurePeriod, &vehicleType, &vehicleSubtype,
			&year, &mileage, &city, &actualPrice,
			&oldPrice, &status,
			pq.Array(&photos), &r.IsNew, pq.Array(&changedCols),
		)
		if err != nil {
			log.Println("Failed scan v2:", err)
			continue
		}

		r.Brand = nullStringToString(brand)
		r.Model = nullStringToString(model)
		r.ExposurePeriod = nullStringToString(exposurePeriod)
		r.VehicleType = nullStringToString(vehicleType)
		r.VehicleSubtype = nullStringToString(vehicleSubtype)
		r.Year = nullStringToString(year)
		r.Mileage = nullStringToString(mileage)
		r.City = nullStringToString(city)
		r.ActualPrice = nullStringToString(actualPrice)
		r.OldPrice = nullStringToString(oldPrice)
		r.Status = nullStringToString(status)
		if r.Status == "" {
			r.Status = "В продаже"
		}

		r.Photos = photos
		r.ChangedColumns = changedCols

		// ИСПРАВЛЕНО: инициализируем пустыми массивами если nil
		if r.Photos == nil {
			r.Photos = []string{}
		}
		if r.ChangedColumns == nil {
			r.ChangedColumns = []string{}
		}

		records = append(records, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func clearChangedColumnsHandlerV2(w http.ResponseWriter, r *http.Request) {
	result, err := db.Exec(`UPDATE leasing_records_v2 SET changed_columns = '{}', updated_at = CURRENT_TIMESTAMP`)
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

func deleteAllRecordsHandlerV2(w http.ResponseWriter, r *http.Request) {
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

	result, err := db.Exec(`DELETE FROM leasing_records_v2`)
	if err != nil {
		http.Error(w, "Failed to delete all records", http.StatusInternalServerError)
		return
	}

	filesMutex.Lock()
	uploadedFilesV2 = []string{}
	filesMutex.Unlock()

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Все записи успешно удалены из базы",
		"rows_deleted": rowsAffected,
	})
}

func exportExcelHandlerV2(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
       SELECT brand, model, vin, exposure_period, vehicle_type, vehicle_subtype,
              year, mileage, city, actual_price
       FROM leasing_records_v2 ORDER BY updated_at DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	headers := []string{"Марка", "Модель", "VIN", "Срок экспозиции (дн.)", "Вид ТС", "Подвид ТС", "Год выпуска", "Пробег", "Город", "Актуальная стоимость"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	rowNum := 2
	for rows.Next() {
		var brand, model, vin, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
		var year, mileage, city, actualPrice sql.NullString

		err := rows.Scan(&brand, &model, &vin, &exposurePeriod, &vehicleType, &vehicleSubtype,
			&year, &mileage, &city, &actualPrice)
		if err != nil {
			log.Println("Failed scan for export:", err)
			continue
		}

		values := []string{
			nullStringToString(brand),
			nullStringToString(model),
			nullStringToString(vin),
			nullStringToString(exposurePeriod),
			nullStringToString(vehicleType),
			nullStringToString(vehicleSubtype),
			nullStringToString(year),
			nullStringToString(mileage),
			nullStringToString(city),
			nullStringToString(actualPrice),
		}

		for i, val := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowNum)
			f.SetCellValue(sheetName, cell, val)
		}
		rowNum++
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=leasing_records_v2.xlsx")
	f.Write(w)
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}
