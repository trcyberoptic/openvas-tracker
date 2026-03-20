// internal/scanner/enrichment_test.go
package scanner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCVE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := NVDResponse{
			Vulnerabilities: []NVDVuln{
				{
					CVE: CVEData{
						ID: "CVE-2024-0001",
						Descriptions: []CVEDescription{
							{Lang: "en", Value: "Test vulnerability"},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	enricher := NewCVEEnricher(server.URL)
	data, err := enricher.Fetch("CVE-2024-0001")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if data.ID != "CVE-2024-0001" {
		t.Errorf("expected CVE-2024-0001, got %s", data.ID)
	}
}
