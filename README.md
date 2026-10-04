# Laundry POS API

Backend API for the Laundry POS application, built with Go and Chi.

## Requirements

- Go 1.22 or newer
- Docker with Compose, or a PostgreSQL 17 instance

## Run locally

Copy `.env.example` to `.env`, replace the placeholder database password, and export the values in your shell. Then start PostgreSQL:

```sh
docker compose up -d postgres
```

Run the API:

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
| `DATABASE_URL` | required | PostgreSQL connection string |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | Initial database connection timeout |
| `MIGRATIONS_PATH` | `file://migrations` | Migration file source URL |

Check that the API is running:

```sh
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

Verify that the API can reach PostgreSQL:

```sh
curl http://localhost:8080/ready
```

Expected response:

```json
{"status":"ready"}
```

If PostgreSQL cannot be reached, `/ready` returns HTTP `503`. The application also verifies the connection during startup and exits with a clear error when PostgreSQL is unavailable.

## Database migrations

Run every pending migration:

```sh
go run ./cmd/migrate up
```

Roll back the latest migration:

```sh
go run ./cmd/migrate down
```

Roll back a specific number of migrations:

```sh
go run ./cmd/migrate down 2
```

Migration files live in `migrations/` and use the sequential naming format `<version>_<description>.up.sql` and `<version>_<description>.down.sql`. The migration tool reports invalid configuration, connection failures, dirty schema state, and SQL errors without logging the database connection string.

## Project structure

```text
cmd/api/          Application entry point
internal/app/     Application lifecycle and HTTP server
internal/config/  Environment-based configuration
internal/httpapi/ HTTP routing, handlers, and responses
internal/migration/ Database migration runner
internal/service/ Business logic
internal/store/   Database access and PostgreSQL connection pool
migrations/       Versioned PostgreSQL schema changes
```

## Test

```sh
go test ./...
```

