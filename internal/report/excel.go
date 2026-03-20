// internal/report/excel.go
package report

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func GenerateExcel(data ReportData) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Vulnerabilities"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{"Severity", "Title", "Host", "Port", "CVE", "CVSS", "Status", "Description", "Solution"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Header style
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"16213E"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "I1", style)

	// Data
	for i, v := range data.Vulnerabilities {
		rowNum := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), string(v.Severity))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), v.Title)
		if v.AffectedHost != nil {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), *v.AffectedHost)
		}
		if v.AffectedPort != nil {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), *v.AffectedPort)
		}
		if v.CveID != nil {
			f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), *v.CveID)
		}
		if v.CvssScore != nil {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), *v.CvssScore)
		}
		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), string(v.Status))
		if v.Description != nil {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), *v.Description)
		}
		if v.Solution != nil {
			f.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), *v.Solution)
		}
	}

	// Auto-width columns
	for i := range headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, colName, colName, 18)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
