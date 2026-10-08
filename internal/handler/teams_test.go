package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTeamRoutes_MutationsRequireAdmin(t *testing.T) {
	// Nil service on purpose: the role check must reject before any handler runs.
	h := &TeamHandler{}
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/teams/t1/members"},
		{http.MethodPost, "/teams/t1/invite"},
		{http.MethodDelete, "/teams/t1"},
	} {
		e := newTestEcho()
		h.RegisterRoutes(e.Group("/teams", withRole("viewer")))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s as viewer: status %d, want 403", tc.method, tc.path, rec.Code)
		}
	}
}
