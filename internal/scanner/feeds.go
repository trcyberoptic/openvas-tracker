package scanner

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// FeedVersion is one Greenbone feed and its current version, parsed from a
// GMP <get_feeds/> response. Version is a YYYYMMDDHHMM timestamp string.
type FeedVersion struct {
	Type    string // e.g. NVT, SCAP, CERT, GVMD_DATA (passed through verbatim from gvmd)
	Name    string
	Version string
}

type getFeedsResponse struct {
	XMLName xml.Name   `xml:"get_feeds_response"`
	Feeds   []feedElem `xml:"feed"`
}

type feedElem struct {
	Type    string `xml:"type"`
	Name    string `xml:"name"`
	Version string `xml:"version"`
}

// ParseFeeds parses a GMP <get_feeds_response> document into feed versions.
func ParseFeeds(r io.Reader) ([]FeedVersion, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read feeds XML: %w", err)
	}
	var resp getFeedsResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse feeds XML: %w", err)
	}
	var feeds []FeedVersion
	for _, f := range resp.Feeds {
		t := strings.TrimSpace(f.Type)
		if t == "" {
			continue
		}
		feeds = append(feeds, FeedVersion{
			Type:    t,
			Name:    strings.TrimSpace(f.Name),
			Version: strings.TrimSpace(f.Version),
		})
	}
	return feeds, nil
}

// ParseFeedVersionTime converts a YYYYMMDDHHMM feed version string into a UTC
// time. Returns ok=false if the string is not in that format.
func ParseFeedVersionTime(version string) (time.Time, bool) {
	t, err := time.Parse("200601021504", strings.TrimSpace(version))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
