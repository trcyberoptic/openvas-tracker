// internal/scanner/openvas.go
package scanner

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

type OpenVASResult struct {
	Title       string
	Host        string
	Port        string
	Severity    string
	CVSSScore   float64
	Description string
	Solution    string
	CVE         string
	OID         string
}

type ovasReport struct {
	XMLName xml.Name    `xml:"report"`
	Results ovasResults `xml:"results"`
}

type ovasResults struct {
	Results []ovasResult `xml:"result"`
}

type ovasResult struct {
	Name        string  `xml:"name"`
	Host        string  `xml:"host"`
	Port        string  `xml:"port"`
	Threat      string  `xml:"threat"`
	Severity    float64 `xml:"severity"`
	Description string  `xml:"description"`
	NVT         ovasNVT `xml:"nvt"`
}

type ovasNVT struct {
	OID      string `xml:"oid,attr"`
	Name     string `xml:"name"`
	CVSSBase string `xml:"cvss_base"`
	CVE      string `xml:"cve"`
	Solution string `xml:"solution"`
}

func ParseOpenVASXML(r io.Reader) ([]OpenVASResult, error) {
	var report ovasReport
	if err := xml.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to parse OpenVAS XML: %w", err)
	}

	var results []OpenVASResult
	for _, res := range report.Results.Results {
		cvss := res.Severity
		if cvss == 0 {
			cvss, _ = strconv.ParseFloat(res.NVT.CVSSBase, 64)
		}

		// Parse port number from "443/tcp" format
		portStr := res.Port

		results = append(results, OpenVASResult{
			Title:       res.Name,
			Host:        res.Host,
			Port:        portStr,
			Severity:    res.Threat,
			CVSSScore:   cvss,
			Description: res.Description,
			Solution:    res.NVT.Solution,
			CVE:         res.NVT.CVE,
			OID:         res.NVT.OID,
		})
	}
	return results, nil
}
