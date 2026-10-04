# Laundry POS API

Backend API for the Laundry POS application, built with Go and Chi.

## Requirements

- Go 1.22 or newer

## Run locally

Copy `.env.example` to `.env`, export its values in your shell, and run:

```sh
go mod download
go run ./cmd/api
```

The server listens on port `8080` by default. Configuration is read from these environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | `development` | Application environment name |
| `PORT` | `8080` | HTTP port, from 1 to 65535 |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |

Check that the API is running:

```sh
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## Project structure

```text
cmd/api/          Application entry point
internal/app/     Application lifecycle and HTTP server
internal/config/  Environment-based configuration
internal/httpapi/ HTTP routing, handlers, and responses
internal/service/ Business logic
internal/store/   Database access
```

## Test

```sh
go test ./...
```

