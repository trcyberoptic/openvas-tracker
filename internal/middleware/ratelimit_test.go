package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// The window must reset on schedule even under continuous traffic — the old
// implementation refreshed lastSeen per request, so it never did.
func TestRateLimiterWindowResetsUnderSteadyTraffic(t *testing.T) {
	e := echo.New()
	rl := NewRateLimiter(2, 100*time.Millisecond)
	h := rl.Middleware()(func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	call := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		if err := h(e.NewContext(req, rec)); err != nil {
			return err.(*echo.HTTPError).Code
		}
		return rec.Code
	}

	if call() != 200 || call() != 200 {
		t.Fatal("first two requests should pass")
	}
	if got := call(); got != http.StatusTooManyRequests {
		t.Fatalf("3rd request in window: got %d, want 429", got)
	}
	// Hammer continuously across the window boundary: the old code refreshed
	// lastSeen on every one of these, so the reset branch never fired.
	deadline := time.Now().Add(250 * time.Millisecond)
	allowed := 0
	for time.Now().Before(deadline) {
		if call() == 200 {
			allowed++
		}
		time.Sleep(10 * time.Millisecond)
	}
	if allowed == 0 {
		t.Fatal("window never reset under steady traffic: every request stayed 429")
	}
}
