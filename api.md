# witti REST API

This document describes the HTTP API exposed by `cmd/witti-api`.

## Base URL

By default, the server listens on:

- `http://localhost:8080`

Run locally:

```bash
make run-api ARGS="-addr :8080"
```

## Endpoints

## `GET /healthz`

Liveness/readiness probe.

### Response

- `200 OK`

```json
{"status":"ok"}
```

## `POST /v1/search`

Executes the same timezone search/projection logic as the CLI.

### Request body

```json
{
  "queryTerms": ["tokyo"],
  "zoneinfoRoot": "",
  "format": "Mon 2006-01-02 15:04:05 MST -07:00",
  "use12Hour": false,
  "limit": 10,
  "showPath": false,
  "projectedLocalTime": "02/17/2027 07:07:00",
  "localTimeZone": "America/Los_Angeles"
}
```

### Field notes

- `queryTerms` (**required**): query terms with OR semantics.
  - text term: `"new york"`
  - GMT offset term: `"gmt-7"`, `"gmt+5:30"`
  - inline projected datetime term: `"02/17/2027 07:07:00"`
- `projectedLocalTime` (optional): explicit projected local datetime.
- `localTimeZone` (optional): IANA zone used to parse projected local datetime.
- `zoneinfoRoot` (optional): discovery root override.
- `format` (optional): Go time layout format.
- `use12Hour` (optional): use default 12h format when `format` is not explicitly provided.
- `limit` (optional): max number of returned results.
- `showPath` (optional): include zoneinfo path in `displayName` when `zoneinfoRoot` is set.

### Successful response

- `200 OK`

```json
{
  "success": true,
  "data": {
    "querySummary": "02/17/2027 07:07:00, new york",
    "source": "built-in IANA zone list",
    "referenceTime": "2027-02-17T07:07:00-08:00",
    "projectedTime": true,
    "results": [
      {
        "zoneName": "America/New_York",
        "displayName": "America/New_York",
        "time": "2027-02-17T10:07:00-05:00",
        "formattedTime": "Wed 2027-02-17 10:07:00 EST -05:00",
        "utcOffsetSeconds": -18000,
        "abbreviation": "EST"
      }
    ]
  }
}
```

### Error response

- `400 Bad Request` for malformed/invalid input
- `405 Method Not Allowed` for wrong HTTP method
- `500 Internal Server Error` for unexpected server-side failures

```json
{
  "success": false,
  "error": "invalid JSON request"
}
```

## curl examples

Projected local time conversion:

```bash
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "queryTerms": ["02/17/2027 07:07:00", "new york"],
    "limit": 1
  }'
```

Offset mode:

```bash
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "queryTerms": ["gmt-7"],
    "limit": 3
  }'
```

