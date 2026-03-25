package cli

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestNewFlagSet(t *testing.T) {
	var errBuf bytes.Buffer
	fs, opts := NewFlagSet(&errBuf, "2006-01-02 15:04:05")

	if opts.Format != "2006-01-02 15:04:05" {
		t.Fatalf("unexpected default format: %q", opts.Format)
	}
	if opts.Limit != 0 {
		t.Fatalf("unexpected default limit: %d", opts.Limit)
	}

	if err := fs.Parse([]string{"-limit", "2", "tokyo"}); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if opts.Limit != 2 {
		t.Fatalf("limit=%d, want 2", opts.Limit)
	}

	fs.Usage()
	usage := errBuf.String()
	if !strings.Contains(usage, "Usage:") || !strings.Contains(usage, "Options:") {
		t.Fatalf("usage text missing expected sections: %q", usage)
	}
}

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
			got, err := ReorderArgsForFlagParse(tc.input)
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

func TestWasFlagProvided(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("format", "", "")
	fs.String("limit", "0", "")

	if err := fs.Parse([]string{"-format", "15:04"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if !WasFlagProvided(fs, "format") {
		t.Error("expected WasFlagProvided=true for 'format'")
	}
	if WasFlagProvided(fs, "limit") {
		t.Error("expected WasFlagProvided=false for 'limit' (not passed)")
	}
	if WasFlagProvided(fs, "notexist") {
		t.Error("expected WasFlagProvided=false for unknown flag")
	}
}
