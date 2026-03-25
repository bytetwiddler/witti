package witti

import (
	"strings"
	"testing"
)

func TestNormalizeForMatch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"America/Los_Angeles", "america los angeles"},
		{"Europe/Paris", "europe paris"},
		{"Etc/GMT+8", "etc gmt 8"},
		{"  New   York  ", "new york"},
		{"Some-Zone.Name/With_Underscores", "some zone name with underscores"},
		{"TOKYO", "tokyo"},
		{"Tokyo", "tokyo"},
		{"tokyo", "tokyo"},
		{"utc", "utc"},
		{"", ""},
	}
	for _, tc := range tests {
		got := normalizeForMatch(tc.input)
		if got != tc.want {
			t.Errorf("normalizeForMatch(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

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

func TestParseGMTOffsetQueryValid(t *testing.T) {
	tests := []struct {
		query   string
		wantSec int
	}{
		{"gmt+0", 0},
		{"gmt+1", 1 * 3600},
		{"gmt+14", 14 * 3600},
		{"gmt-0", 0},
		{"gmt-7", -7 * 3600},
		{"gmt-12", -12 * 3600},
		{"gmt+0530", 5*3600 + 30*60},
		{"gmt-0800", -8 * 3600},
		{"gmt+130", 1*3600 + 30*60},
		{"gmt-7:07", -(7*3600 + 7*60)},
		{"gmt+5:30", 5*3600 + 30*60},
		{"gmt-07:07", -(7*3600 + 7*60)},
		{"gmt+05:30", 5*3600 + 30*60},
		{"gmt+00:00", 0},
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
	notOffsets := []string{
		"paris",
		"america/los_angeles",
		"los angeles",
		"utc",
		"gmt",
		"gmt 8",
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
		{"gmt+", "invalid GMT offset"},
		{"gmt-", "invalid GMT offset"},
		{"gmt+abc", "invalid GMT offset"},
		{"gmt-x", "invalid GMT offset"},
		{"gmt-7:7", "invalid GMT offset"},
		{"gmt+1:007", "invalid GMT offset"},
		{"gmt+1:2:3", "invalid GMT offset"},
		{"gmt+a:00", "invalid GMT offset"},
		{"gmt+07:0x", "invalid GMT offset"},
		{"gmt+100:00", "invalid GMT offset"},
		{"gmt+123456", "invalid GMT offset"},
		{"gmt+15", "valid range"},
		{"gmt-15", "valid range"},
		{"gmt+14:01", "valid range"},
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
		input   string
		wantSec int
	}{
		{"gmt-7", -7 * 3600},
		{"GMT-7", -7 * 3600},
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
