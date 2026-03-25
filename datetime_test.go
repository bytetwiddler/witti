package witti

import (
	"strings"
	"testing"
	"time"
)

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
