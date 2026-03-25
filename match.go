package witti

import (
	"strings"
	"time"
)

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

func zoneOffsetAt(zoneName string, t time.Time) (int, bool) {
	loc, err := time.LoadLocation(zoneName)
	if err != nil {
		return 0, false
	}
	_, offset := t.In(loc).Zone()
	return offset, true
}
