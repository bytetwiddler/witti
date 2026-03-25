package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// normalizeForMatch
// ---------------------------------------------------------------------------

func TestNormalizeForMatch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Separator replacement
		{"America/Los_Angeles", "america los angeles"},
		{"Europe/Paris", "europe paris"},
		{"Etc/GMT+8", "etc gmt 8"},
		// Spaces collapse
		{"  New   York  ", "new york"},
		// Mixed separators
		{"Some-Zone.Name/With_Underscores", "some zone name with underscores"},
		// Case independence
		{"TOKYO", "tokyo"},
		{"Tokyo", "tokyo"},
		{"tokyo", "tokyo"},
		// Already normalized
		{"utc", "utc"},
		// Empty
		{"", ""},
	}
	for _, tc := range tests {
		got := normalizeForMatch(tc.input)
		if got != tc.want {
			t.Errorf("normalizeForMatch(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// isASCIIUnsignedInt
// ---------------------------------------------------------------------------

func TestIsASCIIUnsignedInt(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0", true},
		{"07", true},
		{"123", true},
		{"", false},
		{"1a", false},
		{"-1", false},
		{"1.5", false},
	}
	for _, tc := range tests {
		got := isASCIIUnsignedInt(tc.input)
		if got != tc.want {
			t.Errorf("isASCIIUnsignedInt(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// formatUTCOffset
// ---------------------------------------------------------------------------

func TestFormatUTCOffset(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "UTC+00:00"},
		{3600, "UTC+01:00"},
		{-3600, "UTC-01:00"},
		{-7 * 3600, "UTC-07:00"},
		{5*3600 + 30*60, "UTC+05:30"},
		{-(7*3600 + 7*60), "UTC-07:07"},
		{14 * 3600, "UTC+14:00"},
		{-12 * 3600, "UTC-12:00"},
	}
	for _, tc := range tests {
		got := formatUTCOffset(tc.seconds)
		if got != tc.want {
			t.Errorf("formatUTCOffset(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// parseGMTOffsetQuery
// ---------------------------------------------------------------------------

func TestParseGMTOffsetQueryValid(t *testing.T) {
	tests := []struct {
		query   string
		wantSec int
	}{
		// Integer hour, positive
		{"gmt+0", 0},
		{"gmt+1", 1 * 3600},
		{"gmt+14", 14 * 3600},
		// Integer hour, negative
		{"gmt-0", 0},
		{"gmt-7", -7 * 3600},
		{"gmt-12", -12 * 3600},
		// HHMM forms
		{"gmt+0530", 5*3600 + 30*60},
		{"gmt-0800", -8 * 3600},
		{"gmt+130", 1*3600 + 30*60},
		// Colon: single-digit hour
		{"gmt-7:07", -(7*3600 + 7*60)},
		{"gmt+5:30", 5*3600 + 30*60},
		// Colon: two-digit hour
		{"gmt-07:07", -(7*3600 + 7*60)},
		{"gmt+05:30", 5*3600 + 30*60},
		{"gmt+00:00", 0},
		// Uppercase (input normalised before calling)
		{"gmt+8", 8 * 3600},
	}
	for _, tc := range tests {
		offset, isOffset, err := parseGMTOffsetQuery(tc.query)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.query, err)
			continue
		}
		if !isOffset {
			t.Errorf("%s: expected isOffset=true", tc.query)
			continue
		}
		if offset != tc.wantSec {
			t.Errorf("%s: got %d, want %d", tc.query, offset, tc.wantSec)
		}
	}
}

func TestParseGMTOffsetQueryNotOffset(t *testing.T) {
	// These do not start with gmt+/gmt- so should return (0, false, nil).
	notOffsets := []string{
		"paris",
		"america/los_angeles",
		"los angeles",
		"utc",
		"gmt",     // no sign
		"gmt 8",   // space, not a flag prefix
	}
	for _, q := range notOffsets {
		_, isOffset, err := parseGMTOffsetQuery(q)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", q, err)
		}
		if isOffset {
			t.Errorf("%q: expected isOffset=false", q)
		}
	}
}

func TestParseGMTOffsetQueryInvalid(t *testing.T) {
	tests := []struct {
		query       string
		errContains string
	}{
		// Empty after sign
		{"gmt+", "invalid GMT offset"},
		{"gmt-", "invalid GMT offset"},
		// Non-numeric
		{"gmt+abc", "invalid GMT offset"},
		{"gmt-x", "invalid GMT offset"},
		// Colon with single-digit minutes
		{"gmt-7:7", "invalid GMT offset"},
		// Colon with three-digit minutes
		{"gmt+1:007", "invalid GMT offset"},
		// Multiple colons
		{"gmt+1:2:3", "invalid GMT offset"},
		// Non-numeric characters in colon form
		{"gmt+a:00", "invalid GMT offset"},
		{"gmt+07:0x", "invalid GMT offset"},
		// Three-digit hour in colon form
		{"gmt+100:00", "invalid GMT offset"},
		// Too many digits without colon
		{"gmt+123456", "invalid GMT offset"},
		// Out of range
		{"gmt+15", "valid range"},
		{"gmt-15", "valid range"},
		{"gmt+14:01", "valid range"},
		// Minutes out of range
		{"gmt+00:60", "valid range"},
		{"gmt+00:99", "valid range"},
	}
	for _, tc := range tests {
		_, _, err := parseGMTOffsetQuery(tc.query)
		if err == nil {
			t.Errorf("%q: expected error containing %q, got nil", tc.query, tc.errContains)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("%q: error = %q, want it to contain %q", tc.query, err.Error(), tc.errContains)
		}
	}
}

// ---------------------------------------------------------------------------
// parseQueryTerm
// ---------------------------------------------------------------------------

func TestParseQueryTermText(t *testing.T) {
	tests := []struct {
		input          string
		wantRaw        string
		wantNormalized string
	}{
		{"Paris", "paris", "paris"},
		{"Los Angeles", "los angeles", "los angeles"},
		{"America/New_York", "america/new_york", "america new york"},
		{"TOKYO", "tokyo", "tokyo"},
		// Trim whitespace
		{"  berlin  ", "berlin", "berlin"},
	}
	for _, tc := range tests {
		term, err := parseQueryTerm(tc.input)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.input, err)
			continue
		}
		if term.isOffset {
			t.Errorf("%q: expected text term, got offset", tc.input)
		}
		if term.raw != tc.wantRaw {
			t.Errorf("%q: raw = %q, want %q", tc.input, term.raw, tc.wantRaw)
		}
		if term.normalized != tc.wantNormalized {
			t.Errorf("%q: normalized = %q, want %q", tc.input, term.normalized, tc.wantNormalized)
		}
	}
}

func TestParseQueryTermOffset(t *testing.T) {
	tests := []struct {
		input         string
		wantSec       int
	}{
		{"gmt-7", -7 * 3600},
		{"GMT-7", -7 * 3600}, // uppercase normalised by caller; parseQueryTerm lowercases
		{"gmt+5:30", 5*3600 + 30*60},
		{"gmt-07:07", -(7*3600 + 7*60)},
	}
	for _, tc := range tests {
		term, err := parseQueryTerm(tc.input)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.input, err)
			continue
		}
		if !term.isOffset {
			t.Errorf("%q: expected offset term", tc.input)
			continue
		}
		if term.offsetSeconds != tc.wantSec {
			t.Errorf("%q: offsetSeconds = %d, want %d", tc.input, term.offsetSeconds, tc.wantSec)
		}
	}
}

func TestParseQueryTermEmpty(t *testing.T) {
	term, err := parseQueryTerm("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if term.raw != "" {
		t.Errorf("expected empty raw, got %q", term.raw)
	}
}

func TestParseQueryTermInvalidOffset(t *testing.T) {
	_, err := parseQueryTerm("gmt+99")
	if err == nil {
		t.Error("expected error for gmt+99, got nil")
	}
}

// ---------------------------------------------------------------------------
// projected local time parsing
// ---------------------------------------------------------------------------

func TestParseProjectedLocalTime(t *testing.T) {
	loc := time.UTC
	tests := []struct {
		name       string
		input      string
		want       time.Time
		wantParsed bool
		wantErr    bool
	}{
		{
			name:       "mm slash dd",
			input:      "02/17/2027 07:07:00",
			want:       time.Date(2027, 2, 17, 7, 7, 0, 0, loc),
			wantParsed: true,
		},
		{
			name:       "single digit month day",
			input:      "2/7/2027 07:07:00",
			want:       time.Date(2027, 2, 7, 7, 7, 0, 0, loc),
			wantParsed: true,
		},
		{
			name:       "iso space",
			input:      "2027-02-17 07:07:00",
			want:       time.Date(2027, 2, 17, 7, 7, 0, 0, loc),
			wantParsed: true,
		},
		{
			name:       "iso t",
			input:      "2027-02-17T07:07:00",
			want:       time.Date(2027, 2, 17, 7, 7, 0, 0, loc),
			wantParsed: true,
		},
		{
			name:       "regular query text",
			input:      "new york",
			wantParsed: false,
		},
		{
			name:       "empty string is not projection",
			input:      "   ",
			wantParsed: false,
		},
		{
			name:       "invalid datetime shape errors",
			input:      "02/17/2027 07:07",
			wantParsed: true,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, parsed, err := parseProjectedLocalTime(tc.input, loc)
			if parsed != tc.wantParsed {
				t.Fatalf("parsed=%v, want %v", parsed, tc.wantParsed)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantParsed && !got.Equal(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseQueryTermsAndReferenceTime(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	loc := time.UTC

	tests := []struct {
		name            string
		args            []string
		wantQueries     []string
		wantHasProj     bool
		wantRef         time.Time
		wantErrContains string
	}{
		{
			name:        "no projection uses now",
			args:        []string{"new york"},
			wantQueries: []string{"new york"},
			wantHasProj: false,
			wantRef:     now,
		},
		{
			name:        "projection then query",
			args:        []string{"02/17/2027 07:07:00", "new york"},
			wantQueries: []string{"new york"},
			wantHasProj: true,
			wantRef:     time.Date(2027, 2, 17, 7, 7, 0, 0, loc),
		},
		{
			name:        "query then projection",
			args:        []string{"new york", "02/17/2027 07:07:00"},
			wantQueries: []string{"new york"},
			wantHasProj: true,
			wantRef:     time.Date(2027, 2, 17, 7, 7, 0, 0, loc),
		},
		{
			name:            "multiple projections error",
			args:            []string{"02/17/2027 07:07:00", "2027-02-17 08:00:00", "new york"},
			wantErrContains: "multiple projected local times",
		},
		{
			name:            "invalid datetime-like input errors",
			args:            []string{"02/17/2027 07:07", "new york"},
			wantErrContains: "invalid projected local time",
		},
		{
			name:        "whitespace-only args are ignored",
			args:        []string{"   ", "new york"},
			wantQueries: []string{"new york"},
			wantHasProj: false,
			wantRef:     now,
		},
		{
			name:            "invalid offset query is surfaced",
			args:            []string{"gmt+99", "new york"},
			wantErrContains: "valid range",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			terms, ref, hasProj, err := parseQueryTermsAndReferenceTime(tc.args, now, loc)
			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasProj != tc.wantHasProj {
				t.Fatalf("hasProj=%v, want %v", hasProj, tc.wantHasProj)
			}
			if !ref.Equal(tc.wantRef) {
				t.Fatalf("reference time=%v, want %v", ref, tc.wantRef)
			}
			if len(terms) != len(tc.wantQueries) {
				t.Fatalf("query count=%d, want %d", len(terms), len(tc.wantQueries))
			}
			for i := range terms {
				if terms[i].raw != tc.wantQueries[i] {
					t.Fatalf("term[%d]=%q, want %q", i, terms[i].raw, tc.wantQueries[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// reorderArgsForFlagParse
// ---------------------------------------------------------------------------

func TestReorderArgsForFlagParse(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr bool
	}{
		{
			name:  "flags before query",
			input: []string{"-limit", "5", "tokyo"},
			want:  []string{"-limit", "5", "tokyo"},
		},
		{
			name:  "flags after query",
			input: []string{"tokyo", "-limit", "5"},
			want:  []string{"-limit", "5", "tokyo"},
		},
		{
			name:  "bool flag after query",
			input: []string{"tokyo", "-12h"},
			want:  []string{"-12h", "tokyo"},
		},
		{
			name:  "bool flag before query",
			input: []string{"-12h", "tokyo"},
			want:  []string{"-12h", "tokyo"},
		},
		{
			name:  "showpath flag",
			input: []string{"tokyo", "-showpath"},
			want:  []string{"-showpath", "tokyo"},
		},
		{
			name:  "inline value flag",
			input: []string{"tokyo", "-limit=3"},
			want:  []string{"-limit=3", "tokyo"},
		},
		{
			name:  "multiple queries flags interspersed",
			input: []string{"paris", "-limit", "2", "tokyo"},
			want:  []string{"-limit", "2", "paris", "tokyo"},
		},
		{
			name:  "double dash stops flag parsing",
			input: []string{"-limit", "2", "--", "-notaflag"},
			want:  []string{"-limit", "2", "-notaflag"},
		},
		{
			name:    "missing value for limit",
			input:   []string{"-limit"},
			wantErr: true,
		},
		{
			name:    "missing value for format",
			input:   []string{"-format"},
			wantErr: true,
		},
		{
			name:  "format flag with value",
			input: []string{"tokyo", "-format", "15:04"},
			want:  []string{"-format", "15:04", "tokyo"},
		},
		{
			name:  "zoneinfo flag with value",
			input: []string{"tokyo", "-zoneinfo", "/usr/share/zoneinfo"},
			want:  []string{"-zoneinfo", "/usr/share/zoneinfo", "tokyo"},
		},
		{
			name:  "lone hyphen treated as query term",
			input: []string{"-"},
			want:  []string{"-"},
		},
		{
			name:  "unknown flag passed through as flag arg",
			input: []string{"tokyo", "-unknown"},
			want:  []string{"-unknown", "tokyo"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reorderArgsForFlagParse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (result: %v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("arg[%d] = %q, want %q (full: %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// zoneOffsetAt
// ---------------------------------------------------------------------------

func TestZoneOffsetAtKnownZones(t *testing.T) {
	tests := []struct {
		zone        string
		wantSeconds int
		fixedTime   time.Time // use a fixed winter time (no DST) for stable offsets
	}{
		// UTC is always +0
		{"UTC", 0, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
		// Kolkata is always UTC+05:30
		{"Asia/Kolkata", 5*3600 + 30*60, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
		// Tokyo is always UTC+09:00
		{"Asia/Tokyo", 9 * 3600, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)},
		// Los Angeles in January is PST = UTC-08:00
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

// ---------------------------------------------------------------------------
// zoneCandidates
// ---------------------------------------------------------------------------

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
	// Spot-check a few canonical zones are present
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

func TestZoneCandidatesInvalidRoot(t *testing.T) {
	_, _, err := zoneCandidates("/no/such/path/xyz")
	if err == nil {
		t.Error("expected error for non-existent zoneinfo root")
	}
}

// ---------------------------------------------------------------------------
// wasFlagProvided
// ---------------------------------------------------------------------------

func TestWasFlagProvided(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("format", "", "")
	fs.String("limit", "0", "")

	// Parse with only -format provided.
	if err := fs.Parse([]string{"-format", "15:04"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if !wasFlagProvided(fs, "format") {
		t.Error("expected wasFlagProvided=true for 'format'")
	}
	if wasFlagProvided(fs, "limit") {
		t.Error("expected wasFlagProvided=false for 'limit' (not passed)")
	}
	if wasFlagProvided(fs, "notexist") {
		t.Error("expected wasFlagProvided=false for unknown flag")
	}
}

// ---------------------------------------------------------------------------
// collectZones
// ---------------------------------------------------------------------------

func TestCollectZones(t *testing.T) {
	// Build a small temp zoneinfo-like tree:
	//   root/
	//     UTC            <- valid zone file
	//     leap-seconds.tab <- should be skipped (.tab)
	//     zone.zi        <- should be skipped (.zi)
	//     README.txt     <- should be skipped (.txt)
	//     America/
	//       New_York     <- valid zone file
	//       Chicago      <- valid zone file
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

	// Build a set for easy lookup.
	got := make(map[string]bool, len(zones))
	for _, z := range zones {
		got[z] = true
	}

	// Expected inclusions (slash-normalised).
	for _, want := range []string{"UTC", "America/New_York", "America/Chicago"} {
		if !got[want] {
			t.Errorf("expected %q in results, got: %v", want, zones)
		}
	}

	// Expected exclusions.
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

// ---------------------------------------------------------------------------
// zoneCandidates (custom root)
// ---------------------------------------------------------------------------

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

