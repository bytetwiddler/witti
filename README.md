# witti

> **What Is The Time In (witti)?** — A fast, cross-platform CLI for searching IANA timezone names and displaying the current local time in each matched zone.

![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20FreeBSD-blue)

---

## Features

- 🔍 **Fuzzy name search** — `"Los Angeles"` matches `America/Los_Angeles`
- 🌍 **Offset-aware queries** — `gmt-7`, `gmt+5:30`, `gmt-07:07` match zones by current UTC offset
- 📋 **Multiple queries** — search several cities or offsets in a single command
- 🧭 **Projected local time** — pass one local datetime argument to convert that instant across matched zones
- 🕐 **12 / 24-hour clock** — 24-hour by default, switch with `-12h`
- 🖥️ **Truly cross-platform** — bundles IANA tzdata via `time/tzdata`; no OS timezone files required
- 🔀 **Flexible flag placement** — flags may appear before or after query terms
- ⚙️ **Custom format** — full Go time-format string support via `-format`

---

## Installation

### Pre-built binaries

Download the binary for your platform from the [Releases](../../releases) page.

| Platform | Binary |
|---|---|
| Linux x86-64 | `witti-linux-amd64` |
| Linux ARM64 | `witti-linux-arm64` |
| macOS Intel | `witti-darwin-amd64` |
| macOS Apple Silicon | `witti-darwin-arm64` |
| Windows x86-64 | `witti-windows-amd64.exe` |
| Windows ARM64 | `witti-windows-arm64.exe` |
| FreeBSD x86-64 | `witti-freebsd-amd64` |

### Build from source

Requires [Go 1.24+](https://go.dev/dl/).

```bash
git clone https://github.com/bytetwiddler/witti.git
cd witti
make build
```

The binary is placed in the project root (`witti` / `witti.exe`).

---

## Usage

```
witti <query...> [options]
```

### Options

| Flag | Default | Description |
|---|---|---|
| `-12h` | off | Use 12-hour (AM/PM) clock output |
| `-format <string>` | `Mon 2006-01-02 15:04:05 MST -07:00` | Go time format string |
| `-limit <n>` | `0` (no limit) | Cap number of results |
| `-showpath` | off | Show full zoneinfo path (requires `-zoneinfo`) |
| `-zoneinfo <path>` | built-in | Override IANA timezone data root for discovery |

---

## Examples

All timestamped output below is **sample output**. Your actual results depend on the current date/time and timezone rules.

> Offsets and abbreviations (for example `PST`/`PDT`, `EST`/`EDT`) can change by date due to DST and regional timezone rule updates.

### Search by city name

```text
$ witti "Los Angeles"
America/Los_Angeles               Wed 2026-03-25 12:40:39 PDT -07:00
```

Natural-language names with spaces match underscore-separated IANA zone IDs automatically.

### Search by region

```text
$ witti america -limit 5
America/Anchorage                 Wed 2026-03-25 11:40:39 AKDT -08:00
America/Argentina/Buenos_Aires    Wed 2026-03-25 16:40:39 -03 -03:00
America/Bogota                    Wed 2026-03-25 14:40:39 -05 -05:00
America/Chicago                   Wed 2026-03-25 13:40:39 CDT -05:00
America/Denver                    Wed 2026-03-25 12:40:39 MDT -06:00
```

### Multiple queries at once

```text
$ witti "buenos aires" "new york" Anchorage
America/Anchorage                 Wed 2026-03-25 11:40:39 AKDT -08:00
America/Argentina/Buenos_Aires    Wed 2026-03-25 16:40:39 -03 -03:00
America/New_York                  Wed 2026-03-25 15:40:39 EDT -04:00
```

Results from all query terms are combined, deduplicated, and sorted.

### Project a local datetime into matched zones

If one positional argument looks like a local datetime, `witti` uses that instant instead of the current time.

Supported projected-time input formats:

- `MM/DD/YYYY HH:MM:SS`
- `M/D/YYYY HH:MM:SS`
- `YYYY-MM-DD HH:MM:SS`
- `YYYY-MM-DDTHH:MM:SS`

Only one projected datetime argument is allowed per command.

```text
$ witti "02/17/2027 07:07:00" "new york"
info: projecting local time 2027-02-17 07:07:00 PST
America/New_York                  Wed 2027-02-17 10:07:00 EST -05:00
```

### Offset-aware query

Queries beginning with `gmt+` or `gmt-` switch to **offset mode** — matching every zone whose current UTC offset equals the requested value. Useful when you know the offset but not the zone name.

```text
$ witti "gmt-7" -limit 3
info: offset-aware mode active (gmt-7 -> UTC-07:00)
America/Los_Angeles               Wed 2026-03-25 12:40:39 PDT -07:00
America/Phoenix                   Wed 2026-03-25 12:40:39 MST -07:00
America/Vancouver                 Wed 2026-03-25 12:40:39 PDT -07:00
```

Supported offset formats:

| Input | Interpreted as |
|---|---|
| `gmt-7` | UTC−07:00 |
| `gmt-7:07` | UTC−07:07 |
| `gmt-07:07` | UTC−07:07 |
| `gmt+5:30` | UTC+05:30 |
| `gmt+0530` | UTC+05:30 |

> **Note on DST/date:** offset matching is evaluated at the moment the command runs (or at the projected instant when you pass a datetime argument). A zone such as `America/Los_Angeles` appears in `gmt-7` results during summer (PDT) and in `gmt-8` results during winter (PST).

### 12-hour clock

```text
$ witti tokyo -limit 1 -12h
Asia/Tokyo                        Thu 2026-03-26 04:40:39 AM JST +09:00
```

The `-12h` flag is ignored when `-format` is also provided, since `-format` always takes precedence.

### Custom time format

Uses [Go time format syntax](https://pkg.go.dev/time#Layout).

```text
$ witti paris -format "2006-01-02 15:04 MST"
Europe/Paris                      2026-03-25 21:40 CET
```

### Flags before or after the query

Both of the following are equivalent:

```text
$ witti -limit 3 asia
$ witti asia -limit 3
```

Use `--` to pass a query that starts with a hyphen:

```text
$ witti -- -myquery
```

---

## How matching works

1. **Offset mode** — triggered when a query starts with `gmt+` or `gmt-`. Compares each zone's current UTC offset to the requested value.
2. **Raw substring** — case-insensitive match against the IANA zone name (e.g. `tokyo` → `Asia/Tokyo`).
3. **Normalized substring** — separators (`/`, `_`, `-`, `.`, `+`) are replaced with spaces before comparison, so `"Los Angeles"` matches `America/Los_Angeles`.

Multiple query terms use **OR** semantics — a zone is included if it matches any term.

---

## Building

```bash
# Current platform
make build

# All platforms → ./bin/
make build-all

# Run tests
make test

# Run tests with coverage summary
make test-coverage

# Remove built binaries
make clean
```

`make build-all` produces binaries in `./bin/` named `witti-<os>-<arch>[.exe]`.

---

## Project structure

```
witti/
├── cmd/
│   └── witti/
│       └── main.go        # Thin CLI entry point
├── internal/
│   └── cli/
│       ├── flags.go       # CLI-only flag wiring and arg reordering
│       └── flags_test.go
├── witti.go       # Library entry point and run orchestration
├── query.go       # Query and GMT-offset parsing
├── datetime.go    # Projected local datetime parsing
├── match.go       # Matching and normalization helpers
├── zones_fs.go    # Filesystem-based zone discovery
├── zones_source.go
├── zones_default.go # Built-in IANA zone name list
├── query_test.go
├── datetime_test.go
├── zones_test.go
├── run_test.go     # Run orchestration and end-to-end behavior tests
├── Makefile       # Build, cross-compile, test, and clean targets
├── go.mod
└── README.md
```

---

## Contributing

1. Fork the repository and create a feature branch.
2. Run `make test` to ensure all tests pass before opening a pull request.
3. Keep new functions covered by table-driven tests in the corresponding `*_test.go` file.

---

## License

This project is licensed under the [MIT License](LICENSE).

```
MIT License

Copyright (c) 2026 witti contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

