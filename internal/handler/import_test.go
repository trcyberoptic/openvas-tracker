package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestHandleOpenVAS_BadXML(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/import/openvas", strings.NewReader("not xml"))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &ImportHandler{}
	err := h.HandleOpenVAS(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", err)
	}
}

func TestHandleFeeds_BadXML(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/import/feeds", strings.NewReader("not xml"))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &ImportHandler{}
	err := h.HandleFeeds(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", err)
	}
}

func TestShouldImportReport(t *testing.T) {
	cutoff := time.Date(2026, 10, 17, 13, 43, 46, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		known   bool
		cutoff  *time.Time
		scanEnd time.Time
		want    bool
	}{
		{"first import ever", false, nil, cutoff, true},
		{"ended after newest import", false, &cutoff, cutoff.Add(time.Second), true},
		{"ended exactly at cutoff", false, &cutoff, cutoff, false},
		{"stale: ended before newest import", false, &cutoff, cutoff.Add(-48 * time.Hour), false},
		{"already imported", true, &cutoff, cutoff.Add(time.Hour), false},
	} {
		if got, reason := shouldImportReport(tc.known, tc.cutoff, tc.scanEnd); got != tc.want || (!got && reason == "") {
			t.Errorf("%s: got %v (%q), want %v with reason when false", tc.name, got, reason, tc.want)
		}
	}
}

func TestCheckOpenVASReport_BadParams(t *testing.T) {
	for _, q := range []string{
		"scan_end=2026-10-17T13:44:51Z",                  // no report_id
		"report_id=nope&scan_end=2026-10-17T13:44:51Z",   // not a UUID
		"report_id=44424b9f-0d3d-47f8-b3da-348f5e654ffd", // no scan_end
		"report_id=44424b9f-0d3d-47f8-b3da-348f5e654ffd&scan_end=yesterday",
	} {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/import/openvas/check?"+q, nil)
		c := e.NewContext(req, httptest.NewRecorder())
		err := (&ImportHandler{}).CheckOpenVASReport(c)
		if he, ok := err.(*echo.HTTPError); !ok || he.Code != http.StatusBadRequest {
			t.Errorf("%q: expected 400, got %v", q, err)
		}
	}
}

// A scan whose scanner had no VTs loaded (prod 2026-10-22) is "Done" in GVM but holds only the
// hostname-determination result per host. Importing it would count every ticket on those hosts
// as fixed-in-this-scan.
func TestHandleOpenVAS_RejectsReportWithoutTests(t *testing.T) {
	result := func(host string) string {
		return `<result><name>Hostname Determination Reporting</name><host>` + host + `</host><port>general/tcp</port>
<threat>Log</threat><severity>0.0</severity><nvt oid="1.3.6.1.4.1.25623.1.0.1"><name>Hostname Determination Reporting</name></nvt></result>`
	}
	xml := `<report><results>` + result("10.0.0.1") + result("10.0.0.2") + `</results></report>`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/import/openvas", strings.NewReader(xml))
	c := e.NewContext(req, httptest.NewRecorder())

	err := (&ImportHandler{}).HandleOpenVAS(c) // nil importSvc: reaching the import would panic
	if he, ok := err.(*echo.HTTPError); !ok || he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", err)
	}
}
