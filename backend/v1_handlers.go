package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
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

func RegisterV1Routes(r *mux.Router) {
	r.HandleFunc("/api/upload", uploadHandler).Methods("POST")
	r.HandleFunc("/api/records", getRecordsHandler).Methods("GET")
	//r.HandleFunc("/api/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		filesMutex.RLock()
		defer filesMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadedFiles)
	}).Methods("GET")
	r.HandleFunc("/api/clear-changed-columns", clearChangedColumnsHandler).Methods("POST")
	r.HandleFunc("/api/delete-all-records", deleteAllRecordsHandler).Methods("POST")
	r.HandleFunc("/api/export", exportExcelHandler).Methods("GET")
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

	result := make([]LeasingRecord, 0)

	for rowNum := 2; rowNum <= len(rows); rowNum++ {
		subject := getCellValueByColumn(f, sheetName, "B", rowNum)
		location := getCellValueByColumn(f, sheetName, "AD", rowNum)
		subjectType := getCellValueByColumn(f, sheetName, "E", rowNum)
		vehicleType := getCellValueByColumn(f, sheetName, "F", rowNum)
		vin := getCellValueByColumn(f, sheetName, "G", rowNum)
		year := getCellValueByColumn(f, sheetName, "K", rowNum)
		mileage := getCellValueByColumn(f, sheetName, "L", rowNum)
		daysOnSale := getCellValueByColumn(f, sheetName, "O", rowNum)
		approvedPrice := getCellValueByColumn(f, sheetName, "Q", rowNum)
		status := getCellValueByColumn(f, sheetName, "AN", rowNum)

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
				ChangedColumns: []string{},
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
       (subject, 
        location, 
        subject_type, 
        vehicle_type,
        vin, 
        year, 
        mileage, 
        days_on_sale,
        approved_price, 
        old_price, 
        status, 
        photos, 
        is_new, 
        changed_columns)
       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
       RETURNING id
    `,
		record.Subject,
		record.Location,
		record.SubjectType,
		record.VehicleType,
		record.VIN,
		record.Year, record.Mileage,
		record.DaysOnSale,
		record.ApprovedPrice,
		record.OldPrice,
		record.Status, pq.Array(record.Photos),
		record.IsNew,
		pq.Array(record.ChangedColumns),
	).Scan(&id)
	return id, err
}

func updateRecord(record LeasingRecord) error {
	_, err := db.Exec(`
       UPDATE leasing_records SET
          subject=$1, 
          location=$2, 
          subject_type=$3, 
          vehicle_type=$4,
          year=$5, 
          mileage=$6, 
          days_on_sale=$7,
          approved_price=$8, 
          old_price=$9, 
          status=$10,
          is_new=$11, 
          changed_columns=$12, 
          updated_at=CURRENT_TIMESTAMP
       WHERE vin=$13
    `,
		record.Subject,
		record.Location,
		record.SubjectType,
		record.VehicleType,
		record.Year,
		record.Mileage,
		record.DaysOnSale,
		record.ApprovedPrice,
		record.OldPrice,
		record.Status,
		record.IsNew,
		pq.Array(record.ChangedColumns),
		record.VIN,
	)
	return err
}

func getRecordsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
       SELECT id,
              subject, 
              location, 
              subject_type, 
              vehicle_type, 
              vin,
              year, 
			   mileage, 
			   days_on_sale, 
			   approved_price, 
			   old_price,
              status, 
           COALESCE(photos, '{}'), 
           is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records ORDER BY updated_at DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := make([]LeasingRecord, 0)

	for rows.Next() {
		var r LeasingRecord
		var photos []string
		var changedCols []string
		var subject,
			location,
			subjectType,
			vehicleType sql.NullString
		var year,
			mileage,
			daysOnSale,
			approvedPrice,
			oldPrice,
			status sql.NullString

		err := rows.Scan(
			&r.ID,
			&subject,
			&location,
			&subjectType,
			&vehicleType,
			&r.VIN,
			&year,
			&mileage,
			&daysOnSale,
			&approvedPrice,
			&oldPrice,
			&status,
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

//
//func healthHandler(w http.ResponseWriter, r *http.Request) {
//	w.WriteHeader(http.StatusOK)
//	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
//}

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
func exportExcelHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
       SELECT 
           subject, 
           location, 
           subject_type, 
           vehicle_type, 
           vin,
              year, 
           mileage, 
           days_on_sale, 
           approved_price, 
           old_price, 
           status
       FROM leasing_records ORDER BY updated_at DESC
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

	headers := []string{"Предмет лизинга", "Местонахождение", "Вид предмета лизинга", "Вид ТС", "VIN", "Год выпуска", "Пробег", "Дни в продаже", "Текущая цена", "Старая цена", "Разница", "Статус"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	rowNum := 2
	for rows.Next() {
		var subject, location, subjectType, vehicleType, vin sql.NullString
		var year, mileage, daysOnSale, approvedPrice, oldPrice, status sql.NullString

		err := rows.Scan(&subject, &location, &subjectType, &vehicleType, &vin,
			&year, &mileage, &daysOnSale, &approvedPrice, &oldPrice, &status)
		if err != nil {
			log.Println("Failed scan for export:", err)
			continue
		}

		var difference string
		if oldPrice.Valid && approvedPrice.Valid {
			oldPriceCleaned := strings.ReplaceAll(oldPrice.String, ",", "")
			approvedPriceCleaned := strings.ReplaceAll(approvedPrice.String, ",", "")

			oldVal, err1 := strconv.ParseFloat(oldPriceCleaned, 64)
			approvedVal, err2 := strconv.ParseFloat(approvedPriceCleaned, 64)

			if err1 == nil && err2 == nil {
				diff := oldVal - approvedVal
				difference = fmt.Sprintf("%.2f", diff)
			}
		}

		values := []string{
			nullStringToString(subject),
			nullStringToString(location),
			nullStringToString(subjectType),
			nullStringToString(vehicleType),
			nullStringToString(vin),
			nullStringToString(year),
			nullStringToString(mileage),
			nullStringToString(daysOnSale),
			nullStringToString(approvedPrice),
			nullStringToString(oldPrice),
			difference,
			nullStringToString(status),
		}

		for i, val := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowNum)
			f.SetCellValue(sheetName, cell, val)
		}
		rowNum++
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=leasing_records.xlsx")
	f.Write(w)
}
func deleteRecord(vin string) {
	db.Exec("DELETE FROM leasing_records WHERE vin=$1", vin)
}
