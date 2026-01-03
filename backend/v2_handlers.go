package main

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/xuri/excelize/v2"
)

const screenshotCacheDir = "./screenshots_cache"

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
	Photos         []string `json:"photos"`
	IsNew          bool     `json:"is_new"`
	ChangedColumns []string `json:"changed_columns,omitempty"`
}

func RegisterV2Routes(r *mux.Router) {
	r.HandleFunc("/api/v2/upload", uploadHandlerV2).Methods("POST")
	r.HandleFunc("/api/v2/records", getRecordsHandlerV2).Methods("GET")
	r.HandleFunc("/api/v2/files", func(w http.ResponseWriter, r *http.Request) {
		filesMutex.RLock()
		defer filesMutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadedFilesV2)
	}).Methods("GET")
	r.HandleFunc("/api/v2/clear-changed-columns", clearChangedColumnsHandlerV2).Methods("POST")
	r.HandleFunc("/api/v2/delete-all-records", deleteAllRecordsHandlerV2).Methods("POST")
	r.HandleFunc("/api/v2/export", exportExcelHandlerV2).Methods("GET")
	// Новый endpoint для скриншотов
	r.HandleFunc("/api/v2/screenshot", screenshotHandlerV2).Methods("GET")

	// Создаём директорию для кэша при старте
	os.MkdirAll(screenshotCacheDir, 0755)
}

func screenshotHandlerV2(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if url == "" {
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	hash := md5.Sum([]byte(url))
	filename := hex.EncodeToString(hash[:]) + ".jpg"
	cachePath := filepath.Join(screenshotCacheDir, filename)

	// Проверяем кэш
	if fileInfo, err := os.Stat(cachePath); err == nil {
		if time.Since(fileInfo.ModTime()) < 7*24*time.Hour {
			log.Printf("✓ Serving cached screenshot for: %s", url)
			serveCachedScreenshot(w, cachePath)
			return
		}
	}

	// Генерируем новый скриншот
	log.Printf("⏳ Generating screenshot for: %s", url)
	screenshot, err := captureScreenshot(url)
	if err != nil {
		log.Printf("❌ Screenshot error for %s: %v", url, err)
		// Отдаём плейсхолдер вместо ошибки
		serveErrorPlaceholder(w)
		return
	}

	// Сохраняем в кэш
	if err := ioutil.WriteFile(cachePath, screenshot, 0644); err != nil {
		log.Printf("⚠️  Failed to cache screenshot: %v", err)
	} else {
		log.Printf("✓ Screenshot cached: %s", filename)
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(screenshot)
}

// captureScreenshot с настройками для Docker
func captureScreenshot(url string) ([]byte, error) {
	// Опции для работы в Docker (без sandbox)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),            // ВАЖНО для Docker
		chromedp.Flag("disable-dev-shm-usage", true), // ВАЖНО для Docker
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.WindowSize(1280, 720),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Таймаут на всю операцию
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.CaptureScreenshot(&buf),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp error: %w", err)
	}

	return buf, nil
}

func serveCachedScreenshot(w http.ResponseWriter, path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, "Failed to read cached screenshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(data)
}

// Отдаём простой плейсхолдер при ошибке
func serveErrorPlaceholder(w http.ResponseWriter) {
	// 1x1 прозрачный PNG
	placeholder := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(placeholder)
}
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

	result := make([]LeasingRecordV2, 0)

	for rowNum := 2; rowNum <= len(rows); rowNum++ {
		brand := getCellValueByColumn(f, sheetName, "I", rowNum)
		model := getCellValueByColumn(f, sheetName, "J", rowNum)
		vin := getCellValueByColumn(f, sheetName, "D", rowNum)
		exposurePeriod := getCellValueByColumn(f, sheetName, "C", rowNum)
		vehicleType := getCellValueByColumn(f, sheetName, "F", rowNum)
		vehicleSubtype := getCellValueByColumn(f, sheetName, "G", rowNum)
		year := getCellValueByColumn(f, sheetName, "N", rowNum)
		mileage := getCellValueByColumn(f, sheetName, "AK", rowNum)
		city := getCellValueByColumn(f, sheetName, "L", rowNum)
		actualPrice := getCellValueByColumn(f, sheetName, "K", rowNum)

		// Собираем ссылки из трёх столбцов Excel
		photos := collectPhotosFromExcel(f, sheetName, rowNum)

		if vin == "" {
			continue
		}
		existing, exists := getRecordByVINV2(vin)

		if !exists {
			// Для новых записей используем ссылки из Excel
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
				Photos:         photos,
				IsNew:          true,
				ChangedColumns: []string{},
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

			// Для обновляемых записей также берём ссылки из Excel
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

// collectPhotosFromExcel собирает ссылки из указанных столбцов Excel
func collectPhotosFromExcel(f *excelize.File, sheetName string, rowNum int) []string {
	// Укажите здесь буквы столбцов, из которых нужно собирать ссылки
	// Например: "AL", "AM", "AN" - это столбцы 38, 39, 40
	photoColumns := []string{"AU", "AT", "AS", "AR", "AQ"} // <-- ИЗМЕНИТЕ НА НУЖНЫЕ ВАМ СТОЛБЦЫ

	photos := make([]string, 0, len(photoColumns))

	for _, col := range photoColumns {
		link := getCellValueByColumn(f, sheetName, col, rowNum)
		// Добавляем только непустые ссылки
		if strings.TrimSpace(link) != "" {
			photos = append(photos, strings.TrimSpace(link))
		}
	}

	return photos
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
	return changed
}

func getRecordByVINV2(vin string) (LeasingRecordV2, bool) {
	var rec LeasingRecordV2
	var brand, model, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
	var year, mileage, city, actualPrice, oldPrice sql.NullString

	err := db.QueryRow(`
       SELECT id, brand, model, vin, exposure_period, vehicle_type, vehicle_subtype,
              year, mileage, city, actual_price, old_price,
               COALESCE(photos, '{}'), is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records_v2 WHERE vin=$1
    `, vin).Scan(
		&rec.ID,
		&brand, &model, &rec.VIN, &exposurePeriod, &vehicleType, &vehicleSubtype,
		&year, &mileage, &city, &actualPrice,
		&oldPrice,
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

	return rec, true
}

func insertRecordV2(record LeasingRecordV2) (int, error) {
	var id int
	err := db.QueryRow(`
       INSERT INTO leasing_records_v2
       (brand, 
        model, 
        vin, 
        exposure_period, 
        vehicle_type, 
        vehicle_subtype, 
        year, mileage, 
        city,
        actual_price, 
        old_price, 
        photos, 
        is_new, 
        changed_columns)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
       RETURNING id
    `,
		record.Brand, record.Model,
		record.VIN,
		record.ExposurePeriod,
		record.VehicleType,
		record.VehicleSubtype,
		record.Year,
		record.Mileage,
		record.City,
		record.ActualPrice,
		record.OldPrice,
		pq.Array(record.Photos),
		record.IsNew,
		pq.Array(record.ChangedColumns),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func updateRecordV2(record LeasingRecordV2) error {
	_, err := db.Exec(`
       UPDATE leasing_records_v2 SET
          brand           = $1,
          model           = $2,
          exposure_period = $3,
          vehicle_type    = $4,
          vehicle_subtype = $5,
          year            = $6,
          mileage         = $7,
          city            = $8,
          actual_price    = $9,
          old_price       = $10,
          photos          = $11, 
          is_new          = $12,
          changed_columns = $13,
          updated_at      = CURRENT_TIMESTAMP
       WHERE vin = $14
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
		pq.Array(record.Photos),
		record.IsNew,
		pq.Array(record.ChangedColumns),
		record.VIN,
	)
	return err
}

func getRecordsHandlerV2(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
       SELECT id, brand, 
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
             COALESCE(photos, '{}'), is_new, COALESCE(changed_columns, '{}')
       FROM leasing_records_v2 ORDER BY updated_at DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := make([]LeasingRecordV2, 0)

	for rows.Next() {
		var r LeasingRecordV2
		var photos []string
		var changedCols []string
		var brand,
			model,
			exposurePeriod,
			vehicleType,
			vehicleSubtype sql.NullString
		var year,
			mileage,
			city,
			actualPrice,
			oldPrice sql.NullString

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
       SELECT brand, 
              model, 
              vin, 
              exposure_period, 
              vehicle_type, 
              vehicle_subtype,
              year, 
           mileage, 
           city, 
           actual_price
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

	headers := []string{"Марка", "Модель", "VIN", "Срок экспозиции (дн.)", "Вид ТС", "Подвид ТС", "Год выпуска", "Пробег", "Город", "Текущая цена", "Старая цена", "Разница"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	rowNum := 2
	for rows.Next() {
		var brand, model, vin, exposurePeriod, vehicleType, vehicleSubtype sql.NullString
		var year, mileage, city, actualPrice, oldPrice sql.NullString

		err := rows.Scan(&brand, &model, &vin, &exposurePeriod, &vehicleType, &vehicleSubtype,
			&year, &mileage, &city, &actualPrice, &oldPrice)
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
