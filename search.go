package witti

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SearchRequest describes one timezone search/projection operation.
type SearchRequest struct {
	QueryTerms         []string `json:"queryTerms"`
	ZoneinfoRoot       string   `json:"zoneinfoRoot,omitempty"`
	Format             string   `json:"format,omitempty"`
	Use12Hour          bool     `json:"use12Hour,omitempty"`
	Limit              int      `json:"limit,omitempty"`
	ShowPath           bool     `json:"showPath,omitempty"`
	ProjectedLocalTime string   `json:"projectedLocalTime,omitempty"`
	LocalTimeZone      string   `json:"localTimeZone,omitempty"`
	FormatProvided     bool     `json:"-"`
}

// SearchMatch is one formatted timezone result.
type SearchMatch struct {
	ZoneName         string    `json:"zoneName"`
	DisplayName      string    `json:"displayName"`
	Time             time.Time `json:"time"`
	FormattedTime    string    `json:"formattedTime"`
	UTCOffsetSeconds int       `json:"utcOffsetSeconds"`
	Abbreviation     string    `json:"abbreviation"`
}

// SearchResponse contains normalized metadata and results for a search.
type SearchResponse struct {
	QuerySummary  string        `json:"querySummary"`
	SourceDesc    string        `json:"source"`
	ReferenceTime time.Time     `json:"referenceTime"`
	ProjectedTime bool          `json:"projectedTime"`
	OffsetMode    []string      `json:"offsetMode,omitempty"`
	NoMatches     bool          `json:"noMatches"`
	Results       []SearchMatch `json:"results"`
}

// OffsetSummary returns the same compact offset summary used by CLI info messages.
func (r SearchResponse) OffsetSummary() string {
	return strings.Join(r.OffsetMode, "; ")
}

// Search executes timezone matching/projection logic and returns structured results.
func Search(req SearchRequest, now func() time.Time, local *time.Location) (SearchResponse, error) {
	if now == nil {
		now = time.Now
	}
	if local == nil {
		local = time.Local
	}

	parseLocal := local
	if req.LocalTimeZone != "" {
		loc, err := time.LoadLocation(req.LocalTimeZone)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
		}
		parseLocal = loc
	}

	queryTerms, referenceTime, hasProjectedTime, err := parseQueryTermsAndReferenceTime(req.QueryTerms, now(), parseLocal)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if req.ProjectedLocalTime != "" {
		projectedTime, isProjection, err := parseProjectedLocalTime(req.ProjectedLocalTime, parseLocal)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
		}
		if !isProjection {
			return SearchResponse{}, fmt.Errorf("%w: invalid projected local time %q (expected formats like MM/DD/YYYY HH:MM:SS)", ErrInvalidRequest, req.ProjectedLocalTime)
		}
		if hasProjectedTime {
			return SearchResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, ErrMultipleProjectedTimes)
		}
		referenceTime = projectedTime
		hasProjectedTime = true
	}
	if len(queryTerms) == 0 {
		return SearchResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, ErrEmptyQuery)
	}

	querySummaryParts := make([]string, 0, len(queryTerms))
	offsetSummaryParts := make([]string, 0)
	for _, term := range queryTerms {
		querySummaryParts = append(querySummaryParts, term.raw)
		if term.isOffset {
			offsetSummaryParts = append(offsetSummaryParts, term.raw+" -> "+formatUTCOffset(term.offsetSeconds))
		}
	}

	effectiveFormat := req.Format
	if req.Use12Hour && !req.FormatProvided {
		effectiveFormat = default12HourFormat
	} else if effectiveFormat == "" {
		effectiveFormat = default24HourFormat
	}

	entries, sourceDesc, err := zoneCandidates(req.ZoneinfoRoot)
	if err != nil {
		return SearchResponse{}, err
	}

	matches := collectMatches(entries, queryTerms, referenceTime)
	resp := SearchResponse{
		QuerySummary:  strings.Join(querySummaryParts, ", "),
		SourceDesc:    sourceDesc,
		ReferenceTime: referenceTime,
		ProjectedTime: hasProjectedTime,
		OffsetMode:    offsetSummaryParts,
		Results:       make([]SearchMatch, 0, len(matches)),
	}
	if len(matches) == 0 {
		resp.NoMatches = true
		return resp, nil
	}

	sort.Strings(matches)
	if req.Limit > 0 && req.Limit < len(matches) {
		matches = matches[:req.Limit]
	}

	for _, zoneName := range matches {
		loc, err := time.LoadLocation(zoneName)
		if err != nil {
			continue
		}
		t := referenceTime.In(loc)
		abbr, offset := t.Zone()
		displayName := zoneName
		if req.ShowPath && req.ZoneinfoRoot != "" {
			displayName = filepath.Join(req.ZoneinfoRoot, zoneName)
		}
		resp.Results = append(resp.Results, SearchMatch{
			ZoneName:         zoneName,
			DisplayName:      displayName,
			Time:             t,
			FormattedTime:    t.Format(effectiveFormat),
			UTCOffsetSeconds: offset,
			Abbreviation:     abbr,
		})
	}

	return resp, nil
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
