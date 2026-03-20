// internal/report/pdf.go
package report

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

func GeneratePDF(data ReportData) ([]byte, error) {
	m := maroto.New()

	// Title
	m.AddRows(
		row.New(20).Add(
			col.New(12).Add(
				text.New(data.Title, props.Text{
					Size:  18,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		),
	)

	// Summary row
	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New("Generated: "+data.GeneratedAt, props.Text{
					Size:  9,
					Align: align.Center,
					Color: &props.Color{Red: 128, Green: 128, Blue: 128},
				}),
			),
		),
	)

	// Header row
	headerProps := props.Text{Size: 9, Style: fontstyle.Bold, Color: &props.Color{Red: 255, Green: 255, Blue: 255}}

	m.AddRows(
		row.New(8).Add(
			col.New(2).Add(text.New("Severity", headerProps)),
			col.New(4).Add(text.New("Title", headerProps)),
			col.New(2).Add(text.New("Host", headerProps)),
			col.New(1).Add(text.New("Port", headerProps)),
			col.New(2).Add(text.New("CVE", headerProps)),
			col.New(1).Add(text.New("CVSS", headerProps)),
		),
	)

	// Data rows
	cellProps := props.Text{Size: 8}
	for _, v := range data.Vulnerabilities {
		port := ""
		if v.AffectedPort != nil {
			port = fmt.Sprintf("%d", *v.AffectedPort)
		}
		cve := ""
		if v.CveID != nil {
			cve = *v.CveID
		}
		cvss := ""
		if v.CvssScore != nil {
			cvss = fmt.Sprintf("%.1f", *v.CvssScore)
		}
		host := ""
		if v.AffectedHost != nil {
			host = *v.AffectedHost
		}

		m.AddRows(
			row.New(7).Add(
				col.New(2).Add(text.New(string(v.Severity), cellProps)),
				col.New(4).Add(text.New(v.Title, cellProps)),
				col.New(2).Add(text.New(host, cellProps)),
				col.New(1).Add(text.New(port, cellProps)),
				col.New(2).Add(text.New(cve, cellProps)),
				col.New(1).Add(text.New(cvss, cellProps)),
			),
		)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}
