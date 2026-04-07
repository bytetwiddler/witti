# witti

> **What Is The Time In (witti)?** — A Go package for searching IANA timezone names and projecting times around the world. The repository also provides ready-to-run examples of a command-line tool, a REST API server, and a web interface built on top of the same package.

![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20FreeBSD-blue)
![Podman](https://img.shields.io/badge/container-Podman-892CA0?logo=podman&logoColor=white)

---

## Features

- 🔍 **Fuzzy name search** — `"Los Angeles"` matches `America/Los_Angeles`
- 🌍 **Offset-aware queries** — `gmt-7`, `gmt+5:30`, `gmt-07:07` match zones by current UTC offset
- 📋 **Multiple queries** — search several cities or offsets in a single command
- 🧭 **Projected local time** — pass one local datetime argument to convert that instant across matched zones
- 🕐 **12 / 24-hour clock** — 24-hour by default, switch with `-12h`
- 🖥️ **Truly cross-platform** — bundles IANA tzdata via `time/tzdata`; no OS timezone files required
- 🌐 **REST API** — exposes the same search/projection features over HTTP JSON
- ✨ **Web UI (Go + HTMX + Tailwind)** — server-rendered interface on top of the same search API
- 🔀 **Flexible flag placement** — flags may appear before or after query terms
- ⚙️ **Custom format** — full Go time-format string support via `-format`
- 🐳 **Container-ready** — multi-stage `Dockerfile` + `compose.yaml` for rootless Podman

---

## Installation

### Import the package

```bash
go get github.com/bytetwiddler/witti/v2
```

### Build the example binaries from source

Requires [Go 1.24+](https://go.dev/dl/).

```bash
git clone https://github.com/bytetwiddler/witti.git
cd witti
make build        # CLI binary
make build-api    # REST API server
make build-web    # combined web + API server
```

The binary is placed in the project root (`witti` / `witti.exe`).

### Run in a container

Requires [Podman](https://podman.io/docs/installation) and
[podman-compose](https://github.com/containers/podman-compose#installation)
(`pip install podman-compose` or your distro's package).

```bash
# Build the image and start the combined web + API server
make image-build
make up
```

The container exposes port **8080** and runs as a non-root `witti` user inside
a `debian:bookworm-slim` image. Both the web server (`witti-app`) and the CLI
tool (`witti`) are installed at `/usr/local/bin/` in the image.

---

## Using the Go package

Import path: `github.com/bytetwiddler/witti/v2`

Full reference: [`pkg.go.dev/github.com/bytetwiddler/witti/v2`](https://pkg.go.dev/github.com/bytetwiddler/witti/v2)

### Key types

| Type | Description |
| ---- | ----------- |
| `SearchRequest` | Input: query terms, format options, optional projected time |
| `SearchResponse` | Output: metadata + slice of `SearchMatch` results |
| `SearchMatch` | One matched timezone: formatted local time, UTC time, offset, abbreviation |

### Search for the current time in matched zones

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/bytetwiddler/witti/v2"
)

func main() {
    resp, err := witti.Search(witti.SearchRequest{
        QueryTerms: []string{"new york", "tokyo", "london"},
    }, time.Now, time.Local)
    if err != nil {
        log.Fatal(err)
    }
    for _, m := range resp.Results {
        fmt.Printf("%-32s  %s\n", m.ZoneName, m.FormattedTime)
        fmt.Printf("%-32s  %s\n", "", m.UTCTime)
    }
}
```

Output:

```text
America/New_York                  Mon 2026-04-06 17:17:54 EDT -04:00 (GMT-4)
                                  Mon 2026-04-06 21:17:54 UTC
Asia/Tokyo                        Tue 2026-04-07 06:17:54 JST +09:00 (GMT+9)
                                  Mon 2026-04-06 21:17:54 UTC
Europe/London                     Mon 2026-04-06 22:17:54 BST +01:00 (GMT+1)
                                  Mon 2026-04-06 21:17:54 UTC
```

### Project a fixed datetime across zones

```go
resp, err := witti.Search(witti.SearchRequest{
    QueryTerms:         []string{"new york", "tokyo"},
    ProjectedLocalTime: "02/17/2027 07:07:00",   // MM/DD/YYYY HH:MM:SS
    LocalTimeZone:      "America/Los_Angeles",
}, time.Now, time.Local)
```

### Match zones by UTC offset

```go
resp, err := witti.Search(witti.SearchRequest{
    QueryTerms: []string{"gmt-7"},  // offset mode: all zones at UTC-07:00
    Limit:      5,
}, time.Now, time.Local)
```

### Use 12-hour clock or a custom format

```go
resp, err := witti.Search(witti.SearchRequest{
    QueryTerms: []string{"paris"},
    Use12Hour:  true,
    // Or supply any Go time-format string:
    // Format: "2006-01-02 15:04 MST",
}, time.Now, time.Local)
```

### List all discoverable IANA zone names

```go
zones, err := witti.AllZones()
// zones is a sorted []string of every IANA zone name in the embedded tzdata.
```

### SearchRequest fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| `QueryTerms` | `[]string` | One or more search terms (name substring, `gmt±offset`, or inline datetime) |
| `Limit` | `int` | Maximum results; `0` = no limit |
| `Use12Hour` | `bool` | 12-hour AM/PM output (ignored when `Format` is set) |
| `Format` | `string` | Go time-format string; overrides `Use12Hour` |
| `ProjectedLocalTime` | `string` | Datetime string to project (`MM/DD/YYYY HH:MM:SS` or ISO variants) |
| `LocalTimeZone` | `string` | IANA name for the timezone that owns `ProjectedLocalTime` |
| `ZoneinfoRoot` | `string` | Path to a custom IANA zoneinfo directory; empty uses built-in tzdata |
| `ShowPath` | `bool` | Prefix `DisplayName` with the full zoneinfo file path (requires `ZoneinfoRoot`) |

### SearchMatch fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| `ZoneName` | `string` | IANA zone identifier, e.g. `America/New_York` |
| `DisplayName` | `string` | Zone name (or full path when `ShowPath` is set) |
| `Time` | `time.Time` | The instant in this zone |
| `FormattedTime` | `string` | Human-readable local time using the requested format |
| `GMTLabel` | `string` | Compact offset label, e.g. `GMT-7` |
| `UTCTime` | `string` | Same instant formatted in UTC |
| `UTCOffsetSeconds` | `int` | UTC offset in seconds |
| `Abbreviation` | `string` | Zone abbreviation, e.g. `EST`, `JST` |

---

## Usage

```
witti <query...> [options]
```

### Options

| Flag               | Default                              | Description                                    |
| ------------------ | ------------------------------------ | ---------------------------------------------- |
| `-12h`             | off                                  | Use 12-hour (AM/PM) clock output               |
| `-format <string>` | `Mon 2006-01-02 15:04:05 MST -07:00` | Go time format string                          |
| `-limit <n>`       | `0` (no limit)                       | Cap number of results                          |
| `-showpath`        | off                                  | Show full zoneinfo path (requires `-zoneinfo`) |
| `-zoneinfo <path>` | built-in                             | Override IANA timezone data root for discovery |

## Web UI

The project includes a server-rendered web interface built with:

- **Go** HTTP server
- **HTMX** for dynamic partial updates (no frontend build step)
- **Tailwind CSS** via CDN

Run the combined web + API server:

```bash
make run-web ARGS="-addr :8080"
```

or directly:

```bash
go run ./cmd/witti-web -addr :8080
```

Open:

- `http://localhost:8080/` → web UI
- `http://localhost:8080/v1/search` → REST API endpoint
- `http://localhost:8080/healthz` → health check

The web UI posts form data to `/ui/search`, which returns HTML fragments rendered server-side from the same library logic used by CLI/API. Each result card shows the local formatted time on the first line and the same instant in UTC on the second line.

## REST API

See detailed docs in `api.md` and OpenAPI schema in `openapi.yaml`.

Start the API server:

```bash
make run-api ARGS="-addr :8080"
```

Health endpoint:

```bash
curl http://localhost:8080/healthz
```

Search endpoint (`POST /v1/search`):

```bash
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{
	"queryTerms": ["02/17/2027 07:07:00", "new york"],
	"limit": 1,
	"use12Hour": false
  }'
```

Request JSON fields mirror library/CLI behavior:

- `queryTerms` (required): array of terms; supports text, `gmt+/-` offset terms, and inline projected datetime term
- `projectedLocalTime` (optional): explicit projected local datetime
- `localTimeZone` (optional): IANA timezone name for parsing projected local datetime
- `zoneinfoRoot`, `format`, `use12Hour`, `limit`, `showPath`

Response includes metadata (`querySummary`, `referenceTime`, `projectedTime`, `offsetMode`) and structured `results`. Each result object includes the primary display time in `formattedTime` and the UTC companion value in `utcTime`.

Example response shape:

```json
{
  "success": true,
  "data": {
	"querySummary": "tokyo",
	"results": [
	  {
		"zoneName": "Asia/Tokyo",
		"formattedTime": "Tue 2026-04-07 06:16:43 JST +09:00",
		"gmtLabel": "GMT+9",
		"utcTime": "Mon 2026-04-06 21:16:43 UTC"
	  }
	]
  }
}
```

---

## Examples

All timestamped output below is **sample output**. Your actual results depend on the current date/time and timezone rules.

> Offsets and abbreviations (for example `PST`/`PDT`, `EST`/`EDT`) can change by date due to DST and regional timezone rule updates.

In the CLI and web UI, each match is rendered as two lines: the local time line first, followed by the same instant formatted in UTC. In the API, the same UTC value is returned in each result object's `utcTime` field.

### Search by city name

```text
$ witti "Los Angeles"
America/Los_Angeles               Mon 2026-04-06 14:17:54 PDT -07:00 (GMT-7)
                                  Mon 2026-04-06 21:17:54 UTC
```

Natural-language names with spaces match underscore-separated IANA zone IDs automatically.

### Search by region

```text
$ witti america -limit 5
America/Anchorage                 Mon 2026-04-06 13:17:54 AKDT -08:00 (GMT-8)
                                  Mon 2026-04-06 21:17:54 UTC
America/Argentina/Buenos_Aires    Mon 2026-04-06 18:17:54 -03 -03:00 (GMT-3)
                                  Mon 2026-04-06 21:17:54 UTC
America/Bogota                    Mon 2026-04-06 16:17:54 -05 -05:00 (GMT-5)
                                  Mon 2026-04-06 21:17:54 UTC
America/Chicago                   Mon 2026-04-06 16:17:54 CDT -05:00 (GMT-5)
                                  Mon 2026-04-06 21:17:54 UTC
America/Denver                    Mon 2026-04-06 15:17:54 MDT -06:00 (GMT-6)
                                  Mon 2026-04-06 21:17:54 UTC
```

### Multiple queries at once

```text
$ witti "buenos aires" "new york" Anchorage
America/Anchorage                 Mon 2026-04-06 13:17:54 AKDT -08:00 (GMT-8)
                                  Mon 2026-04-06 21:17:54 UTC
America/Argentina/Buenos_Aires    Mon 2026-04-06 18:17:54 -03 -03:00 (GMT-3)
                                  Mon 2026-04-06 21:17:54 UTC
America/New_York                  Mon 2026-04-06 17:17:54 EDT -04:00 (GMT-4)
                                  Mon 2026-04-06 21:17:54 UTC
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
America/New_York                  Wed 2027-02-17 10:07:00 EST -05:00 (GMT-5)
                                  Wed 2027-02-17 15:07:00 UTC
```

### Offset-aware query

Queries beginning with `gmt+` or `gmt-` switch to **offset mode** — matching every zone whose current UTC offset equals the requested value. Useful when you know the offset but not the zone name.

```text
$ witti "gmt-7" -limit 3
info: offset-aware mode active (gmt-7 -> UTC-07:00)
America/Los_Angeles               Mon 2026-04-06 14:17:54 PDT -07:00 (GMT-7)
                                  Mon 2026-04-06 21:17:54 UTC
America/Phoenix                   Mon 2026-04-06 14:17:54 MST -07:00 (GMT-7)
                                  Mon 2026-04-06 21:17:54 UTC
America/Vancouver                 Mon 2026-04-06 14:17:54 PDT -07:00 (GMT-7)
                                  Mon 2026-04-06 21:17:54 UTC
```

Supported offset formats:

| Input       | Interpreted as |
| ----------- | -------------- |
| `gmt-7`     | UTC−07:00      |
| `gmt-7:07`  | UTC−07:07      |
| `gmt-07:07` | UTC−07:07      |
| `gmt+5:30`  | UTC+05:30      |
| `gmt+0530`  | UTC+05:30      |

> **Note on DST/date:** offset matching is evaluated at the moment the command runs (or at the projected instant when you pass a datetime argument). A zone such as `America/Los_Angeles` appears in `gmt-7` results during summer (PDT) and in `gmt-8` results during winter (PST).

### 12-hour clock

```text
$ witti tokyo -limit 1 -12h
Asia/Tokyo                        Tue 2026-04-07 06:17:54 AM JST +09:00 (GMT+9)
                                  Mon 2026-04-06 09:17:54 PM UTC
```

The `-12h` flag is ignored when `-format` is also provided, since `-format` always takes precedence.

### Custom time format

Uses [Go time format syntax](https://pkg.go.dev/time#Layout).

```text
$ witti paris -format "2006-01-02 15:04 MST"
Europe/Paris                      2026-04-06 23:17 CEST (GMT+2)
                                  Mon 2026-04-06 21:17:54 UTC
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
# Current platform CLI binary
make build

# REST API server binary
make build-api

# Web UI + API server binary
make build-web

# All platforms -> ./bin/
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

## Container (Podman)

The project ships a multi-stage `Dockerfile` and a `compose.yaml`.
The runtime image is based on **`debian:bookworm-slim`** — the smallest
Linux image with `apt` support (~97 MB total with binaries).

### Image contents

| Binary | Path in image | Purpose |
| ------ | ------------- | ------- |
| `witti-app` | `/usr/local/bin/witti-app` | Server entrypoint (`witti-web` by default) |
| `witti` | `/usr/local/bin/witti` | CLI tool — always present for interactive use |

### Podman Makefile targets

```bash
# Build (or rebuild) the container image
make image-build

# Start the web + API server in the background (port 8080)
make up

# Stop and remove containers
make down

# Show running compose services
make ps

# Follow container logs  (use ARGS='--tail 50' to limit output)
make logs
make logs ARGS='--tail 50'

# Open an interactive shell as the witti user
make shell

# Open an interactive shell as root (no password needed)
make shell-root

# Remove the local container image
make image-clean
```

> **Note on capability warnings:** rootless Podman prints
> `can't raise ambient capability …` lines before each build step.
> These are harmless host-side warnings from running without root on
> the host and do not indicate any failure inside the container.

### Overridable variables

| Variable | Default | Example override |
| -------- | ------- | ---------------- |
| `IMAGE` | `witti-web` | `make image-build IMAGE=myrepo/witti` |
| `IMAGE_TAG` | `latest` | `make image-build IMAGE_TAG=v1.2.0` |
| `COMPOSE_FILE` | `compose.yaml` | `make up COMPOSE_FILE=compose.prod.yaml` |

### Build a different server target

To containerise the standalone REST API server instead of the combined
web + API server, pass `TARGET` as a build argument:

```bash
podman build --build-arg TARGET=witti-api -t witti-api:latest .
```

---

## Project structure

```
witti
|-- LICENSE
|-- Makefile
|-- README.md
|-- api.md
|-- api_embed.go
|-- cmd/
|   |-- witti/
|   |   `-- main.go
|   |-- witti-api/
|   |   `-- main.go
|   `-- witti-web/
|       `-- main.go
|-- compose.yaml
|-- datetime.go
|-- datetime_test.go
|-- Dockerfile
|-- errors.go
|-- go.mod
|-- go.sum
|-- internal/
|   |-- cli/
|   |   |-- flags.go
|   |   `-- flags_test.go
|   |-- httpapi/
|   |   |-- server.go
|   |   `-- server_test.go
|   `-- webui/
|       |-- docs.go
|       |-- server.go
|       |-- server_test.go
|       `-- web/
|           `-- index.html
|-- match.go
|-- openapi.yaml
|-- query.go
|-- query_test.go
|-- run_test.go
|-- search.go
|-- witti.go
|-- zones_default.go
|-- zones_fs.go
|-- zones_source.go
`-- zones_test.go
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
