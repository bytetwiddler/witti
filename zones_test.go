package witti

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestZoneOffsetAtKnownZones(t *testing.T) {
	tests := []struct {
		zone        string
		wantSeconds int
		fixedTime   time.Time
	}{
		{"UTC", 0, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
		{"Asia/Kolkata", 5*3600 + 30*60, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
		{"Asia/Tokyo", 9 * 3600, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
		{"America/Los_Angeles", -8 * 3600, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		offset, ok := zoneOffsetAt(tc.zone, tc.fixedTime)
		if !ok {
			t.Errorf("%s: expected ok=true", tc.zone)
			continue
		}
		if offset != tc.wantSeconds {
			t.Errorf("%s: offset = %d (%s), want %d (%s)",
				tc.zone, offset, formatUTCOffset(offset),
				tc.wantSeconds, formatUTCOffset(tc.wantSeconds))
		}
	}
}

func TestZoneOffsetAtInvalidZone(t *testing.T) {
	_, ok := zoneOffsetAt("Not/A/Real/Zone", time.Now())
	if ok {
		t.Error("expected ok=false for invalid zone")
	}
}

func TestZoneCandidatesDefaultList(t *testing.T) {
	zones, desc, err := zoneCandidates("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(desc, "built-in") {
		t.Errorf("desc = %q, expected it to contain 'built-in'", desc)
	}
	if len(zones) == 0 {
		t.Error("expected non-empty default zone list")
	}
	wanted := []string{"America/New_York", "Europe/London", "Asia/Tokyo", "UTC"}
	zoneSet := make(map[string]bool, len(zones))
	for _, z := range zones {
		zoneSet[z] = true
	}
	for _, w := range wanted {
		if !zoneSet[w] {
			t.Errorf("expected %q in default zone list", w)
		}
	}
}

func TestAllZones(t *testing.T) {
	zones, err := AllZones()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) == 0 {
		t.Fatal("expected non-empty zone list")
	}
	for i := 1; i < len(zones); i++ {
		if zones[i] < zones[i-1] {
			t.Fatalf("zones not sorted: %q before %q", zones[i-1], zones[i])
		}
	}
	zoneSet := make(map[string]bool, len(zones))
	for _, z := range zones {
		zoneSet[z] = true
	}
	for _, want := range []string{"UTC", "America/Los_Angeles", "Europe/London"} {
		if !zoneSet[want] {
			t.Errorf("expected %q in AllZones results", want)
		}
	}
}

func TestZoneCandidatesInvalidRoot(t *testing.T) {
	_, _, err := zoneCandidates("/no/such/path/xyz")
	if err == nil {
		t.Error("expected error for non-existent zoneinfo root")
	}
}

func TestCollectZones(t *testing.T) {
	root := t.TempDir()

	files := []string{
		"UTC",
		"leap-seconds.tab",
		"zone.zi",
		"README.txt",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte{}, 0600); err != nil {
			t.Fatalf("creating %s: %v", f, err)
		}
	}
	americaDir := filepath.Join(root, "America")
	if err := os.Mkdir(americaDir, 0700); err != nil {
		t.Fatalf("creating America dir: %v", err)
	}
	for _, z := range []string{"New_York", "Chicago"} {
		if err := os.WriteFile(filepath.Join(americaDir, z), []byte{}, 0600); err != nil {
			t.Fatalf("creating America/%s: %v", z, err)
		}
	}

	zones, err := collectZones(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := make(map[string]bool, len(zones))
	for _, z := range zones {
		got[z] = true
	}

	for _, want := range []string{"UTC", "America/New_York", "America/Chicago"} {
		if !got[want] {
			t.Errorf("expected %q in results, got: %v", want, zones)
		}
	}

	for _, skip := range []string{"leap-seconds.tab", "zone.zi", "README.txt"} {
		if got[skip] {
			t.Errorf("expected %q to be skipped, but it appeared in results", skip)
		}
	}
}

func TestCollectZonesInvalidRoot(t *testing.T) {
	_, err := collectZones("/no/such/path/xyz")
	if err == nil {
		t.Error("expected error for non-existent root")
	}
}

func TestZoneCandidatesCustomRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "UTC"), []byte{}, 0600); err != nil {
		t.Fatalf("creating UTC: %v", err)
	}

	zones, desc, err := zoneCandidates(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(desc, root) {
		t.Errorf("desc = %q, expected it to contain %q", desc, root)
	}
	found := false
	for _, z := range zones {
		if z == "UTC" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'UTC' in custom root zone list, got: %v", zones)
	}
}
