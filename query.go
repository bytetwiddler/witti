package witti

import (
	"fmt"
	"strconv"
	"strings"
)

type queryTerm struct {
	raw           string
	normalized    string
	isOffset      bool
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
