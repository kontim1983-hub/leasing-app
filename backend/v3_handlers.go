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

type LeasingRecordV3 struct {
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

func RegisterV3Routes(r *mux.Router) {
	r.HandleFunc("/api/v3/upload", uploadHandlerV3).Methods("POST")
	r.HandleFunc("/api/v3/records", getRecordsHandlerV3).Methods("GET")
	//r.HandleFunc("/api/v3/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v3/files", func(w http.ResponseWriter, r *http.Request) {
		filesMutex.RLock()
		defer filesMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadedFilesV3)
	}).Methods("GET")
	r.HandleFunc("/api/v3/clear-changed-columns", clearChangedColumnsHandlerV3).Methods("POST")
	r.HandleFunc("/api/v3/delete-all-records", deleteAllRecordsHandlerV3).Methods("POST")
	r.HandleFunc("/api/v3/export", exportExcelHandlerV3).Methods("GET")
}

func uploadHandlerV3(w http.ResponseWriter, r *http.Request) {
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

	records, err := processExcelV3(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process Excel: %v", err), http.StatusInternalServerError)
		return
	}

	filesMutex.Lock()
	exists := false
	for _, fn := range uploadedFilesV3 {
		if fn == filename {
			exists = true
			break
		}
	}
	if !exists {
		uploadedFilesV3 = append(uploadedFilesV3, filename)
	}
	filesCopy := make([]string, len(uploadedFilesV3))
	copy(filesCopy, uploadedFilesV3)
	filesMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records":   records,
		"file_name": filename,
		"files":     filesCopy,
	})
}

func processExcelV3(f *excelize.File) ([]LeasingRecordV3, error) {
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

	result := make([]LeasingRecordV3, 0)

	for rowNum := 2; rowNum <= len(rows); rowNum++ {
		brand := getCellValueByColumn(f, sheetName, "K", rowNum)
		model := getCellValueByColumn(f, sheetName, "L", rowNum)
		vin := getCellValueByColumn(f, sheetName, "F", rowNum)
		exposurePeriod := getCellValueByColumn(f, sheetName, "AW", rowNum)
		vehicleType := getCellValueByColumn(f, sheetName, "G", rowNum)
		vehicleSubtype := getCellValueByColumn(f, sheetName, "H", rowNum)
		year := getCellValueByColumn(f, sheetName, "R", rowNum)
		mileage := getCellValueByColumn(f, sheetName, "BA", rowNum)
		city := getCellValueByColumn(f, sheetName, "P", rowNum)
		actualPrice := getCellValueByColumn(f, sheetName, "N", rowNum)
		status := getCellValueByColumn(f, sheetName, "C", rowNum)

		if vin == "" {
			continue
		}

		if status != "В свободной продаже" {
			deleteRecordV3(vin)
			continue
		}

		existing, exists := getRecordByVINV3(vin)

		if !exists {
			photos := searchPhotos(vin)
			if photos == nil {
				photos = []string{}
			}

			record := LeasingRecordV3{
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
				ChangedColumns: []string{},
			}

			id, err := insertRecordV3(record)
			if err != nil {
				log.Printf("Failed to insert record: %v", err)
				continue
			}
			record.ID = id
			result = append(result, record)
		} else {
			changed := compareRecordsV3(existing, LeasingRecordV3{
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

			record := LeasingRecordV3{
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

			err := updateRecordV3(record)
			if err != nil {
				log.Printf("Failed to update record v3: %v", err)
				continue
			}

			result = append(result, record)
		}
	}

	return result, nil
}

func compareRecordsV3(old, new LeasingRecordV3) []string {
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

func getRecordByVINV3(vin string) (LeasingRecordV3, bool) {
	var rec LeasingRecordV3
	var brand, model, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
	var year, mileage, city, actualPrice, oldPrice, status sql.NullString

	err := db.QueryRow(`
		SELECT id,
		       brand,
		       model,
		       vin,
		       exposure_period,
		       vehicle_type,
		       vehicle_subtype,
		       year,
		       mileage,
		       city,
		       actual_price,
		       old_price,
		       status,
		       COALESCE(photos, '{}'),
		       is_new,
		       COALESCE(changed_columns, '{}')
		FROM leasing_records_v3
		WHERE vin = $1
	`, vin).Scan(
		&rec.ID,
		&brand,
		&model,
		&rec.VIN,
		&exposurePeriod,
		&vehicleType,
		&vehicleSubtype,
		&year,
		&mileage,
		&city,
		&actualPrice,
		&oldPrice,
		&status,
		pq.Array(&rec.Photos),
		&rec.IsNew,
		pq.Array(&rec.ChangedColumns),
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

func insertRecordV3(record LeasingRecordV3) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO leasing_records_v3 (
			brand,
			model,
			vin,
			exposure_period,
			vehicle_type,
			vehicle_subtype,
			year,
			mileage,
			city,
			actual_price,
			old_price,
			status,
			photos,
			is_new,
			changed_columns
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id
	`,
		record.Brand,
		record.Model,
		record.VIN,
		record.ExposurePeriod,
		record.VehicleType,
		record.VehicleSubtype,
		record.Year,
		record.Mileage,
		record.City,
		record.ActualPrice,
		record.OldPrice,
		record.Status,
		pq.Array(record.Photos),
		record.IsNew,
		pq.Array(record.ChangedColumns),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func updateRecordV3(record LeasingRecordV3) error {
	_, err := db.Exec(`
        UPDATE leasing_records_v3 SET
            brand = $1,
            model = $2,
            exposure_period = $3,
            vehicle_type = $4,
            vehicle_subtype = $5,
            year = $6,
            mileage = $7,
            city = $8,
            actual_price = $9,
            old_price = $10,
            photos = $11,
            status = $12,
            is_new = $13,
            changed_columns = $14,
            updated_at = CURRENT_TIMESTAMP
        WHERE vin = $15
    `,
		record.Brand,
		record.Model,
		record.ExposurePeriod,
		record.VehicleType,
		record.VehicleSubtype,
		record.Year,
		record.Mileage,
		record.City,
		record.ActualPrice,
		record.OldPrice,
		pq.Array(record.Photos),         // $11
		record.Status,                   // $12
		record.IsNew,                    // $13 ← ИСПРАВЛЕНО
		pq.Array(record.ChangedColumns), // $14 ← ИСПРАВЛЕНО
		record.VIN,                      // $15 ← ИСПРАВЛЕНО
	)
	return err
}

func getRecordsHandlerV3(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id,
		       brand,
		       model,
		       vin,
		       exposure_period,
		       vehicle_type,
		       vehicle_subtype,
		       year,
		       mileage,
		       city,
		       actual_price,
		       old_price,
		       status,
		       COALESCE(photos, '{}'),
		       is_new,
		       COALESCE(changed_columns, '{}')
		FROM leasing_records_v3
		ORDER BY updated_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := make([]LeasingRecordV3, 0)

	for rows.Next() {
		var r LeasingRecordV3
		var photos, changedCols []string
		var brand, model, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
		var year, mileage, city, actualPrice, oldPrice, status sql.NullString

		err := rows.Scan(
			&r.ID,
			&brand,
			&model,
			&r.VIN,
			&exposurePeriod,
			&vehicleType,
			&vehicleSubtype,
			&year,
			&mileage,
			&city,
			&actualPrice,
			&oldPrice,
			&status,
			pq.Array(&photos),
			&r.IsNew,
			pq.Array(&changedCols),
		)
		if err != nil {
			log.Println("Failed scan:", err)
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
		r.Photos = photos
		r.ChangedColumns = changedCols

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

//
//func healthHandler(w http.ResponseWriter, r *http.Request) {
//	w.WriteHeader(http.StatusOK)
//	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
//}

func clearChangedColumnsHandlerV3(w http.ResponseWriter, r *http.Request) {
	result, err := db.Exec(`UPDATE leasing_records_v3 SET changed_columns = '{}', updated_at = CURRENT_TIMESTAMP`)
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

func deleteAllRecordsHandlerV3(w http.ResponseWriter, r *http.Request) {
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

	result, err := db.Exec(`DELETE FROM leasing_records_v3`)
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
func exportExcelHandlerV3(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT brand,
		       model,
		       vin,
		       exposure_period,
		       vehicle_type,
		       vehicle_subtype,
		       year,
		       mileage,
		       city,
		       actual_price,
		       old_price,
		       status
		FROM leasing_records_v3
		ORDER BY updated_at DESC
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

	headers := []string{"Марка", "Модель", "VIN", "Срок экспозиции (дн.)", "Вид ТС", "Подвид ТС", "Год выпуска", "Пробег", "Город", "Текущая цена", "Старая цена", "Разница", "Статус"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	rowNum := 2
	for rows.Next() {
		var brand, model, vin, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
		var year, mileage, city, actualPrice, oldPrice, status sql.NullString

		err := rows.Scan(&brand, &model, &vin, &exposurePeriod, &vehicleType, &vehicleSubtype,
			&year, &mileage, &city, &actualPrice, &oldPrice, &status)
		if err != nil {
			log.Println("Failed scan for export:", err)
			continue
		}

		var difference string
		if oldPrice.Valid && actualPrice.Valid {
			oldPriceCleaned := strings.ReplaceAll(oldPrice.String, ",", "")
			approvedPriceCleaned := strings.ReplaceAll(actualPrice.String, ",", "")

			oldVal, err1 := strconv.ParseFloat(oldPriceCleaned, 64)
			approvedVal, err2 := strconv.ParseFloat(approvedPriceCleaned, 64)

			if err1 == nil && err2 == nil {
				diff := oldVal - approvedVal
				difference = fmt.Sprintf("%.2f", diff)
			}
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
	w.Header().Set("Content-Disposition", "attachment; filename=leasing_records_v3.xlsx")
	f.Write(w)
}
func deleteRecordV3(vin string) {
	db.Exec("DELETE FROM leasing_records_v3 WHERE vin=$1", vin)
}
