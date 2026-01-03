package main

import (
	"database/sql"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Преобразование sql.NullString в строку
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Получение значения ячейки по букве столбца и номеру строки (1-based)
func getCellValueByColumn(f *excelize.File, sheetName string, colLetter string, rowNum int) string {
	cellName := fmt.Sprintf("%s%d", colLetter, rowNum)
	value, err := f.GetCellValue(sheetName, cellName)
	if err != nil {
		return ""
	}
	return value
}

// Поиск фотографий по VIN (заглушка, возвращает пустой массив)
func searchPhotos(vin string) []string {
	return []string{}
}
