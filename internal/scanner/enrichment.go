// internal/scanner/enrichment.go
package scanner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultNVDBaseURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"

type CVEEnricher struct {
	baseURL string
	client  *http.Client
}

func NewCVEEnricher(baseURL string) *CVEEnricher {
	if baseURL == "" {
		baseURL = defaultNVDBaseURL
	}
	return &CVEEnricher{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type NVDResponse struct {
	Vulnerabilities []NVDVuln `json:"vulnerabilities"`
}

type NVDVuln struct {
	CVE CVEData `json:"cve"`
}

type CVEData struct {
	ID           string           `json:"id"`
	Descriptions []CVEDescription `json:"descriptions"`
	References   []CVEReference   `json:"references"`
	Metrics      json.RawMessage  `json:"metrics"`
}

type CVEDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type CVEReference struct {
	URL    string `json:"url"`
	Source string `json:"source"`
}

func (e *CVEEnricher) Fetch(cveID string) (*CVEData, error) {
	url := fmt.Sprintf("%s?cveId=%s", e.baseURL, cveID)
	resp, err := e.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("NVD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NVD returned status %d", resp.StatusCode)
	}

	var nvdResp NVDResponse
	if err := json.NewDecoder(resp.Body).Decode(&nvdResp); err != nil {
		return nil, fmt.Errorf("failed to decode NVD response: %w", err)
	}

	if len(nvdResp.Vulnerabilities) == 0 {
		return nil, fmt.Errorf("CVE %s not found", cveID)
	}

	return &nvdResp.Vulnerabilities[0].CVE, nil
}
