package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

const default24HourFormat = "Mon 2006-01-02 15:04:05 MST -07:00"
const default12HourFormat = "Mon 2006-01-02 03:04:05 PM MST -07:00"

func main() {
	// Flags
	zoneinfoRoot := flag.String("zoneinfo", "",
		"optional path to IANA timezone data root used for discovery")
	format := flag.String("format", default24HourFormat,
		"time format (Go time format syntax)")
	use12Hour := flag.Bool("12h", false, "use 12-hour clock output (default is 24-hour)")
	limit := flag.Int("limit", 0, "limit number of results (0 = no limit)")
	showPath := flag.Bool("showpath", false, "show full filesystem path (only when -zoneinfo is set)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `whattime - search IANA timezone names and print current times

Usage:
	  whattime <substring...> [options]

Examples:
	  whattime -limit 5 tokyo
  whattime Paris
	  whattime "buenos aires" "new york" Anchorage
	  whattime tokyo -12h
  whattime america
  whattime tokyo -format "2006-01-02 15:04 MST"

Options:
`)
		flag.PrintDefaults()
	}
	parseArgs, err := reorderArgsForFlagParse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if err := flag.CommandLine.Parse(parseArgs); err != nil {
		os.Exit(2)
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	queryTerms := make([]queryTerm, 0, len(flag.Args()))
	for _, arg := range flag.Args() {
		term, err := parseQueryTerm(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		if term.raw == "" {
			continue
		}
		queryTerms = append(queryTerms, term)
	}
	if len(queryTerms) == 0 {
		fmt.Fprintln(os.Stderr, "error: empty query")
		os.Exit(2)
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
	if len(offsetSummaryParts) > 0 {
		fmt.Fprintf(os.Stderr, "info: offset-aware mode active (%s)\n", strings.Join(offsetSummaryParts, "; "))
	}

	effectiveFormat := *format
	if *use12Hour && !wasFlagProvided(flag.CommandLine, "format") {
		effectiveFormat = default12HourFormat
	}

	entries, sourceDesc, err := zoneCandidates(*zoneinfoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: collecting zones: %v\n", err)
		os.Exit(1)
	}

	// Filter by case-insensitive substring match
	matches := make([]string, 0, len(entries))
	now := time.Now()
	for _, name := range entries {
		lowerName := strings.ToLower(name)
		normalizedName := normalizeForMatch(name)
		zoneOffset, ok := zoneOffsetAt(name, now)
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

	if len(matches) == 0 {
		fmt.Printf("No timezone entries found containing any of %q in %s\n", querySummary, sourceDesc)
		os.Exit(0)
	}

	// Stable output: sort matches
	sort.Strings(matches)

	// Respect limit
	if *limit > 0 && *limit < len(matches) {
		matches = matches[:*limit]
	}

	// Print current time for each matched zone
	printed := 0
	for _, zoneName := range matches {
		loc, err := time.LoadLocation(zoneName)
		if err != nil {
			// Keep discovery broad but only print zones that can be loaded.
			continue
		}
		printed++
		t := time.Now().In(loc)
		if *showPath && *zoneinfoRoot != "" {
			fmt.Printf("%s\t%s\n", filepath.Join(*zoneinfoRoot, zoneName), t.Format(effectiveFormat))
		} else {
			fmt.Printf("%-32s  %s\n", zoneName, t.Format(effectiveFormat))
		}
	}

	if printed == 0 {
		fmt.Fprintf(os.Stderr, "No loadable timezone entries found for any of %q\n", querySummary)
		os.Exit(1)
	}
}

func reorderArgsForFlagParse(args []string) ([]string, error) {
	boolFlags := map[string]bool{
		"12h":      true,
		"showpath": true,
	}
	valueFlags := map[string]bool{
		"zoneinfo": true,
		"format":   true,
		"limit":    true,
	}

	flagArgs := make([]string, 0, len(args))
	queryArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		tok := args[i]
		if tok == "--" {
			queryArgs = append(queryArgs, args[i+1:]...)
			break
		}

		if strings.HasPrefix(tok, "-") && tok != "-" {
			nameAndValue := strings.TrimLeft(tok, "-")
			name := nameAndValue
			hasInlineValue := false
			if idx := strings.Index(nameAndValue, "="); idx >= 0 {
				name = nameAndValue[:idx]
				hasInlineValue = true
			}

			if boolFlags[name] {
				flagArgs = append(flagArgs, tok)
				continue
			}

			if valueFlags[name] {
				flagArgs = append(flagArgs, tok)
				if !hasInlineValue {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for flag %q", tok)
					}
					i++
					flagArgs = append(flagArgs, args[i])
				}
				continue
			}

			// Keep unknown -x style tokens as flags so the flag package can report errors.
			flagArgs = append(flagArgs, tok)
			continue
		}

		queryArgs = append(queryArgs, tok)
	}

	return append(flagArgs, queryArgs...), nil
}

func wasFlagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}

type queryTerm struct {
	raw          string
	normalized   string
	isOffset     bool
	offsetSeconds int
}

func parseQueryTerm(arg string) (queryTerm, error) {
	raw := strings.ToLower(strings.TrimSpace(arg))
	if raw == "" {
		return queryTerm{}, nil
	}

	offsetSeconds, isOffset, err := parseGMTOffsetQuery(raw)
	if err != nil {
		return queryTerm{}, err
	}

	if isOffset {
		return queryTerm{raw: raw, isOffset: true, offsetSeconds: offsetSeconds}, nil
	}

	return queryTerm{raw: raw, normalized: normalizeForMatch(raw)}, nil
}

func parseGMTOffsetQuery(q string) (int, bool, error) {
	if !(strings.HasPrefix(q, "gmt+") || strings.HasPrefix(q, "gmt-")) {
		return 0, false, nil
	}

	sign := 1
	if q[3] == '-' {
		sign = -1
	}
	rest := strings.TrimSpace(q[4:])
	if rest == "" {
		return 0, true, fmt.Errorf("invalid GMT offset query %q (expected gmt+H, gmt-H, gmt+HH:MM, or gmt-HHMM)", q)
	}

	hours := 0
	minutes := 0
	if strings.Contains(rest, ":") {
		parts := strings.Split(rest, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return 0, true, fmt.Errorf("invalid GMT offset query %q", q)
		}
		if len(parts[0]) < 1 || len(parts[0]) > 2 || len(parts[1]) != 2 {
			return 0, true, fmt.Errorf("invalid GMT offset query %q", q)
		}
		if !isASCIIUnsignedInt(parts[0]) || !isASCIIUnsignedInt(parts[1]) {
			return 0, true, fmt.Errorf("invalid GMT offset query %q", q)
		}
		hours, _ = strconv.Atoi(parts[0])
		minutes, _ = strconv.Atoi(parts[1])
	} else {
		if !isASCIIUnsignedInt(rest) {
			return 0, true, fmt.Errorf("invalid GMT offset query %q", q)
		}
		switch len(rest) {
		case 1, 2:
			hours, _ = strconv.Atoi(rest)
		case 3, 4:
			hours, _ = strconv.Atoi(rest[:len(rest)-2])
			minutes, _ = strconv.Atoi(rest[len(rest)-2:])
		default:
			return 0, true, fmt.Errorf("invalid GMT offset query %q", q)
		}
	}

	if hours > 14 || minutes > 59 || (hours == 14 && minutes != 0) {
		return 0, true, fmt.Errorf("invalid GMT offset query %q (valid range is UTC-14:00 to UTC+14:00)", q)
	}

	return sign * (hours*3600 + minutes*60), true, nil
}

func isASCIIUnsignedInt(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func formatUTCOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, hours, minutes)
}

func zoneOffsetAt(zoneName string, t time.Time) (int, bool) {
	loc, err := time.LoadLocation(zoneName)
	if err != nil {
		return 0, false
	}
	_, offset := t.In(loc).Zone()
	return offset, true
}

var matchSeparators = strings.NewReplacer(
	"/", " ",
	"_", " ",
	"-", " ",
	".", " ",
	"+", " ",
)

func normalizeForMatch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = matchSeparators.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func zoneCandidates(zoneinfoRoot string) ([]string, string, error) {
	if zoneinfoRoot == "" {
		return defaultZoneNames, "built-in IANA zone list", nil
	}

	zones, err := collectZones(zoneinfoRoot)
	if err != nil {
		return nil, "", err
	}

	return zones, fmt.Sprintf("zoneinfo root %s", zoneinfoRoot), nil
}

// collectZones walks the zoneinfo root and returns relative zone names like "Europe/Paris".
func collectZones(root string) ([]string, error) {
	var zones []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				// Root itself is unreadable or missing; surface the error.
				return err
			}
			// Skip other unreadable entries silently.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// Skip obvious non-zone files
		name := d.Name()
		if strings.HasSuffix(name, ".tab") || strings.HasSuffix(name, ".zi") || strings.HasSuffix(name, ".txt") {
			return nil
		}
		// Resolve relative zone name (e.g., "Europe/Paris")
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		// Exclude files directly at root that aren't standard zones (optional).
		// But keep common ones like "UTC", "GMT".
		// We'll include everything and rely on LoadLocation; non-loadables are skipped.
		// Convert Windows separators to '/' for Go's LoadLocation
		zoneName := filepath.ToSlash(rel)

		// Some platforms include "posix/" and "right/" trees; these names are still valid
		// if Go's tzdata knows them. We'll include and let LoadLocation decide.
		zones = append(zones, zoneName)
		return nil
	})
	return zones, err
}
