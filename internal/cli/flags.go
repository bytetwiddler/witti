package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type Options struct {
	ZoneinfoRoot string
	Format       string
	Use12Hour    bool
	Limit        int
	ShowPath     bool
}

func NewFlagSet(stderr io.Writer, defaultFormat string) (*flag.FlagSet, *Options) {
	opts := &Options{}
	fs := flag.NewFlagSet("witti", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.StringVar(&opts.ZoneinfoRoot, "zoneinfo", "", "optional path to IANA timezone data root used for discovery")
	fs.StringVar(&opts.Format, "format", defaultFormat, "time format (Go time format syntax)")
	fs.BoolVar(&opts.Use12Hour, "12h", false, "use 12-hour clock output (default is 24-hour)")
	fs.IntVar(&opts.Limit, "limit", 0, "limit number of results (0 = no limit)")
	fs.BoolVar(&opts.ShowPath, "showpath", false, "show full filesystem path (only when -zoneinfo is set)")

	fs.Usage = func() {
		fmt.Fprintf(stderr, `witti - search IANA timezone names and print current times

Usage:
  witti <substring...> [options]

Examples:
  witti -limit 5 tokyo
  witti Paris
  witti "buenos aires" "new york" Anchorage
  witti "02/17/2027 07:07:00" "new york"
  witti tokyo -12h
  witti america
  witti tokyo -format "2006-01-02 15:04 MST"

Options:
`)
		fs.PrintDefaults()
	}

	return fs, opts
}

func ReorderArgsForFlagParse(args []string) ([]string, error) {
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

func WasFlagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
