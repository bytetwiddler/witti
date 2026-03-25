package witti

import (
	"fmt"
	"strings"
	"time"
)

func parseQueryTermsAndReferenceTime(args []string, now time.Time, local *time.Location) ([]queryTerm, time.Time, bool, error) {
	queryTerms := make([]queryTerm, 0, len(args))
	referenceTime := now
	hasProjectedTime := false

	for _, arg := range args {
		projected, isProjection, err := parseProjectedLocalTime(arg, local)
		if err != nil {
			return nil, time.Time{}, false, err
		}
		if isProjection {
			if hasProjectedTime {
				return nil, time.Time{}, false, fmt.Errorf("multiple projected local times provided; use only one datetime argument")
			}
			referenceTime = projected
			hasProjectedTime = true
			continue
		}

		term, err := parseQueryTerm(arg)
		if err != nil {
			return nil, time.Time{}, false, err
		}
		if term.raw == "" {
			continue
		}
		queryTerms = append(queryTerms, term)
	}

	return queryTerms, referenceTime, hasProjectedTime, nil
}

func parseProjectedLocalTime(arg string, local *time.Location) (time.Time, bool, error) {
	s := strings.TrimSpace(arg)
	if s == "" {
		return time.Time{}, false, nil
	}

	layouts := []string{
		"01/02/2006 15:04:05",
		"1/2/2006 15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, local); err == nil {
			return t, true, nil
		}
	}

	if looksLikeDateTimeInput(s) {
		return time.Time{}, true, fmt.Errorf("invalid projected local time %q (expected formats like MM/DD/YYYY HH:MM:SS)", arg)
	}

	return time.Time{}, false, nil
}

func looksLikeDateTimeInput(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	return strings.Contains(s, "/") || strings.Contains(s, "-") || strings.ContainsRune(s, 'T')
}
