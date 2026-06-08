package scanner

import (
	"strings"
	"testing"
)

const sampleFeeds = `<get_feeds_response status="200" status_text="OK"><feed><type>NVT</type><name>Greenbone Community Feed</name><version>202606051742</version><description>x</description></feed><feed><type>SCAP</type><name>Greenbone SCAP Data Feed</name><version>202606050631</version></feed><feed><type>CERT</type><name>Greenbone CERT Data Feed</name><version>202606051004</version></feed><feed><type>GVMD_DATA</type><name>Greenbone Data Objects Feed</name><version>202606050632</version></feed></get_feeds_response>`

func TestParseFeeds(t *testing.T) {
	feeds, err := ParseFeeds(strings.NewReader(sampleFeeds))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != 4 {
		t.Fatalf("expected 4 feeds, got %d", len(feeds))
	}
	want := map[string]string{
		"NVT": "202606051742", "SCAP": "202606050631",
		"CERT": "202606051004", "GVMD_DATA": "202606050632",
	}
	for _, f := range feeds {
		if want[f.Type] != f.Version {
			t.Errorf("feed %s version = %q, want %q", f.Type, f.Version, want[f.Type])
		}
	}
	if feeds[0].Name != "Greenbone Community Feed" {
		t.Errorf("NVT name = %q", feeds[0].Name)
	}
}

func TestParseFeeds_BadXML(t *testing.T) {
	if _, err := ParseFeeds(strings.NewReader("not xml")); err == nil {
		t.Error("expected error for non-XML input")
	}
}

func TestParseFeedVersionTime(t *testing.T) {
	tm, ok := ParseFeedVersionTime("202606051742")
	if !ok {
		t.Fatal("expected ok for valid version")
	}
	if tm.Year() != 2026 || tm.Month() != 6 || tm.Day() != 5 || tm.Hour() != 17 || tm.Minute() != 42 {
		t.Errorf("parsed time wrong: %v", tm)
	}
	if _, ok := ParseFeedVersionTime("garbage"); ok {
		t.Error("expected !ok for garbage")
	}
}
