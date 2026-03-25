package witti

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCollectMatches(t *testing.T) {
	reference := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	entries := []string{"America/New_York", "America/Los_Angeles", "Asia/Tokyo"}

	textTerm, err := parseQueryTerm("new york")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	offsetTerm, err := parseQueryTerm("gmt-8")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	matches := collectMatches(entries, []queryTerm{textTerm, offsetTerm}, reference)
	got := strings.Join(matches, ",")
	if !strings.Contains(got, "America/New_York") {
		t.Fatalf("expected text match for New_York, got %v", matches)
	}
	if !strings.Contains(got, "America/Los_Angeles") {
		t.Fatalf("expected offset match for Los_Angeles, got %v", matches)
	}
}

func TestRun(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("loading test location: %v", err)
	}
	fixedNow := func() time.Time {
		return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name           string
		args           []string
		wantCode       int
		stdoutContains []string
		stderrContains []string
	}{
		{
			name:           "no args shows usage",
			args:           nil,
			wantCode:       2,
			stderrContains: []string{"Usage:", "Options:"},
		},
		{
			name:           "simple query",
			args:           []string{"tokyo", "-limit", "1"},
			wantCode:       0,
			stdoutContains: []string{"Asia/Tokyo"},
		},
		{
			name:           "projected local datetime",
			args:           []string{"02/17/2027 07:07:00", "new york", "-limit", "1"},
			wantCode:       0,
			stdoutContains: []string{"America/New_York", "10:07:00"},
			stderrContains: []string{"info: projecting local time"},
		},
		{
			name:           "invalid projected datetime",
			args:           []string{"02/17/2027 07:07", "new york"},
			wantCode:       2,
			stderrContains: []string{"invalid projected local time"},
		},
		{
			name:           "offset mode",
			args:           []string{"gmt-7", "-limit", "2"},
			wantCode:       0,
			stderrContains: []string{"offset-aware mode active"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer
			code := Run(tc.args, &out, &errOut, fixedNow, la)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout=%q\nstderr=%q", code, tc.wantCode, out.String(), errOut.String())
			}
			for _, s := range tc.stdoutContains {
				if !strings.Contains(out.String(), s) {
					t.Fatalf("stdout missing %q\nstdout=%q\nstderr=%q", s, out.String(), errOut.String())
				}
			}
			for _, s := range tc.stderrContains {
				if !strings.Contains(errOut.String(), s) {
					t.Fatalf("stderr missing %q\nstderr=%q", s, errOut.String())
				}
			}
		})
	}
}
