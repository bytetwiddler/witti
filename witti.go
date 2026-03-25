package witti

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/bytetwiddler/witti/internal/cli"
)

const default24HourFormat = "Mon 2006-01-02 15:04:05 MST -07:00"
const default12HourFormat = "Mon 2006-01-02 03:04:05 PM MST -07:00"

// Run executes the CLI behavior with injectable dependencies for testing and reuse.
func Run(args []string, stdout io.Writer, stderr io.Writer, now func() time.Time, local *time.Location) int {
	if now == nil {
		now = time.Now
	}
	if local == nil {
		local = time.Local
	}

	fs, opts := cli.NewFlagSet(stderr, default24HourFormat)

	parseArgs, err := cli.ReorderArgsForFlagParse(args)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return 2
	}

	queryTerms, referenceTime, hasProjectedTime, err := parseQueryTermsAndReferenceTime(fs.Args(), now(), local)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	if len(queryTerms) == 0 {
		fmt.Fprintln(stderr, "error: empty query")
		return 2
	}

	querySummaryParts := make([]string, 0, len(queryTerms))
	offsetSummaryParts := make([]string, 0)
	for _, term := range queryTerms {
		querySummaryParts = append(querySummaryParts, term.raw)
		if term.isOffset {
			offsetSummaryParts = append(offsetSummaryParts, fmt.Sprintf("%s -> %s", term.raw, formatUTCOffset(term.offsetSeconds)))
		}
	}
	querySummary := strings.Join(querySummaryParts, ", ")
	if hasProjectedTime {
		fmt.Fprintf(stderr, "info: projecting local time %s\n", referenceTime.Format("2006-01-02 15:04:05 MST"))
	}
	if len(offsetSummaryParts) > 0 {
		fmt.Fprintf(stderr, "info: offset-aware mode active (%s)\n", strings.Join(offsetSummaryParts, "; "))
	}

	effectiveFormat := opts.Format
	if opts.Use12Hour && !cli.WasFlagProvided(fs, "format") {
		effectiveFormat = default12HourFormat
	}

	entries, sourceDesc, err := zoneCandidates(opts.ZoneinfoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: collecting zones: %v\n", err)
		return 1
	}

	matches := collectMatches(entries, queryTerms, referenceTime)
	if len(matches) == 0 {
		fmt.Fprintf(stdout, "No timezone entries found containing any of %q in %s\n", querySummary, sourceDesc)
		return 0
	}

	sort.Strings(matches)
	if opts.Limit > 0 && opts.Limit < len(matches) {
		matches = matches[:opts.Limit]
	}

	printed := 0
	for _, zoneName := range matches {
		loc, err := time.LoadLocation(zoneName)
		if err != nil {
			continue
		}
		printed++
		t := referenceTime.In(loc)
		if opts.ShowPath && opts.ZoneinfoRoot != "" {
			fmt.Fprintf(stdout, "%s\t%s\n", filepath.Join(opts.ZoneinfoRoot, zoneName), t.Format(effectiveFormat))
		} else {
			fmt.Fprintf(stdout, "%-32s  %s\n", zoneName, t.Format(effectiveFormat))
		}
	}

	if printed == 0 {
		fmt.Fprintf(stderr, "No loadable timezone entries found for any of %q\n", querySummary)
		return 1
	}

	return 0
}

func collectMatches(entries []string, queryTerms []queryTerm, referenceTime time.Time) []string {
	matches := make([]string, 0, len(entries))
	for _, name := range entries {
		lowerName := strings.ToLower(name)
		normalizedName := normalizeForMatch(name)
		zoneOffset, ok := zoneOffsetAt(name, referenceTime)
		for _, term := range queryTerms {
			if term.isOffset {
				if ok && zoneOffset == term.offsetSeconds {
					matches = append(matches, name)
					break
				}
				continue
			}

			if strings.Contains(lowerName, term.raw) || strings.Contains(normalizedName, term.normalized) {
				matches = append(matches, name)
				break
			}
		}
	}
	return matches
}
