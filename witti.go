package witti

import (
	"errors"
	"fmt"
	"io"
	"time"
	_ "time/tzdata"

	"github.com/bytetwiddler/witti/internal/cli"
)

const default24HourFormat = "Mon 2006-01-02 15:04:05 MST -07:00"
const default12HourFormat = "Mon 2006-01-02 03:04:05 PM MST -07:00"

// gmtUTCDetail* formats are used for the supplemental GMT-offset and UTC lines.
// They intentionally omit the redundant numeric offset suffix (-07:00) since the
// zone abbreviation (e.g. "GMT-7" or "UTC") already encodes the offset.
const gmtUTCDetailFormat24h = "Mon 2006-01-02 15:04:05 MST"
const gmtUTCDetailFormat12h = "Mon 2006-01-02 03:04:05 PM MST"

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

	formatWasProvided := cli.WasFlagProvided(fs, "format")
	resp, err := Search(SearchRequest{
		QueryTerms:     fs.Args(),
		ZoneinfoRoot:   opts.ZoneinfoRoot,
		Format:         opts.Format,
		Use12Hour:      opts.Use12Hour,
		Limit:          opts.Limit,
		ShowPath:       opts.ShowPath,
		FormatProvided: formatWasProvided,
	}, now, local)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, ErrInvalidRequest) {
			return 2
		}
		return 1
	}

	if resp.ProjectedTime {
		fmt.Fprintf(stderr, "info: projecting local time %s\n", resp.ReferenceTime.Format("2006-01-02 15:04:05 MST"))
	}
	if len(resp.OffsetMode) > 0 {
		fmt.Fprintf(stderr, "info: offset-aware mode active (%s)\n", resp.OffsetSummary())
	}
	if resp.NoMatches {
		fmt.Fprintf(stdout, "No timezone entries found containing any of %q in %s\n", resp.QuerySummary, resp.SourceDesc)
		return 0
	}
	if len(resp.Results) == 0 {
		fmt.Fprintf(stderr, "No loadable timezone entries found for any of %q\n", resp.QuerySummary)
		return 1
	}

	for _, match := range resp.Results {
		if opts.ShowPath && opts.ZoneinfoRoot != "" {
			fmt.Fprintf(stdout, "%s\t%s (%s)\n", match.DisplayName, match.FormattedTime, match.GMTLabel)
			fmt.Fprintf(stdout, "\t%s\n", match.UTCTime)
		} else {
			fmt.Fprintf(stdout, "%-32s  %s (%s)\n", match.DisplayName, match.FormattedTime, match.GMTLabel)
			fmt.Fprintf(stdout, "%-32s  %s\n", "", match.UTCTime)
		}
	}

	return 0
}
