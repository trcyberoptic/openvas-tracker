package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/scanner"
)

type FeedHandler struct {
	q *queries.Queries
}

func NewFeedHandler(q *queries.Queries) *FeedHandler {
	return &FeedHandler{q: q}
}

type feedStatusResponse struct {
	FeedType    string  `json:"feed_type"`
	FeedName    string  `json:"feed_name"`
	Version     string  `json:"version"`
	VersionDate *string `json:"version_date"`
	LastSeen    string  `json:"last_seen"`
	LastChanged string  `json:"last_changed"`
}

// ListFeeds returns the latest observed Greenbone feed versions.
func (h *FeedHandler) ListFeeds(c echo.Context) error {
	feeds, err := h.q.ListFeedStatus(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list feeds")
	}
	out := make([]feedStatusResponse, 0, len(feeds))
	for _, f := range feeds {
		r := feedStatusResponse{
			FeedType:    f.FeedType,
			FeedName:    f.FeedName,
			Version:     f.Version,
			LastSeen:    f.LastSeen.UTC().Format(time.RFC3339),
			LastChanged: f.LastChanged.UTC().Format(time.RFC3339),
		}
		if vd, ok := scanner.ParseFeedVersionTime(f.Version); ok {
			s := vd.UTC().Format(time.RFC3339)
			r.VersionDate = &s
		}
		out = append(out, r)
	}
	return c.JSON(http.StatusOK, out)
}

func (h *FeedHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.ListFeeds)
}
