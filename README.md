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


## Authentication

Apply migrations before starting the API. Migration `000002` adds users and token sessions.

Create the first administrator (or a cashier) using the local provisioning command. It reads the password from `LAUNDRY_USER_PASSWORD`, hashes it with bcrypt, and never prints or stores the plaintext password. Usernames use 3–64 letters, numbers, dots, underscores, or hyphens. Passwords must contain 12–72 bytes. Roles are `admin` and `cashier`.

```powershell
$accountPassword = Read-Host 'New account password' -AsSecureString
$env:LAUNDRY_USER_PASSWORD = [System.Net.NetworkCredential]::new('', $accountPassword).Password
try {
    go run ./cmd/create-user -name 'Admin Laundry' -username admin -email admin@laundry.test -role admin
} finally {
    Remove-Item Env:LAUNDRY_USER_PASSWORD
}
```

Set `DATABASE_URL` before provisioning. There are no default accounts or seeded passwords. Use `-role cashier` to create a cashier.

Additional configuration:

| Variable | Default | Description |
| --- | --- | --- |
| `AUTH_SESSION_TTL` | `8h` | Token lifetime, from 1 second to 24 hours |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` | Comma-separated exact frontend origins (no paths or wildcards) |

### Login contract

`POST /auth/login` with `Content-Type: application/json`:

```json
{"identifier":"admin@laundry.test","password":"<account password>"}
```

`identifier` accepts a username or email, case-insensitively. Success returns HTTP 200:

```json
{
  "access_token": "<opaque random token>",
  "token_type": "Bearer",
  "expires_in": 28800,
  "user": {
    "id": 1,
    "name": "Admin Laundry",
    "username": "admin",
    "email": "admin@laundry.test",
    "role": "admin"
  }
}
```

Errors use `{"error":{"code":"...","message":"..."}}`:

- HTTP 400 `invalid_request`: malformed JSON or missing/invalid fields.
- HTTP 401 `invalid_credentials`: unknown account or incorrect password.
- HTTP 403 `account_inactive`: correct credentials for an inactive account.
- HTTP 415 `invalid_content_type`: request must use JSON.
- HTTP 500 `internal_error`: authentication storage or server failure.

`GET /auth/me` requires `Authorization: Bearer <access_token>` and returns `{"user": {...}}`. Missing, expired, revoked, or inactive-user sessions return HTTP 401 `unauthorized`. Account status and role are read from the database on every protected request.

`POST /auth/logout` requires the same header, revokes that token, and returns HTTP 204. Authentication responses use `Cache-Control: no-store`. Password hashes and account-status internals are excluded from JSON responses.

Tokens contain 32 cryptographically random bytes. PostgreSQL stores only their SHA-256 hashes; no signing secret is required. Expired sessions cannot authenticate. Old session rows may be periodically pruned with `DELETE FROM laundry.auth_sessions WHERE expires_at <= CURRENT_TIMESTAMP`.

### PostgreSQL authentication integration test

Against a separate test database with migrations already applied:

```powershell
$env:TEST_DATABASE_URL = '<test database connection string>'
go test ./...
```

The PostgreSQL integration test inserts its own temporary user and deletes that user and its sessions afterward. Without `TEST_DATABASE_URL`, that integration test is skipped; unit and HTTP tests still run.

## User management (admin only)

- `GET /users` returns `{ "data": [...] }` with public account information and last activity.
- `POST /users` accepts `{ "name": "...", "identifier": "<username or email>", "role": "admin|cashier", "is_active": true, "password": "..." }`.
- `PUT /users/{id}` accepts the same profile fields without password. It updates name, login identity, role, and account status.

Username-only accounts have no email; email accounts receive a unique internal username. UI/API account creation requires at least 8 Unicode characters and at most 72 UTF-8 bytes for bcrypt. The initial local provisioning command retains its stricter 12-byte minimum.

Passwords/hashes are never returned. Disabling an account or changing its role revokes all that account's sessions; reactivation does not restore old tokens. The current admin cannot demote/deactivate itself, and the final active administrator is protected. Account records are preserved for future transaction references.

Migration `000003` allows username-only accounts and tracks last activity. Before rolling it back, assign email addresses to all username-only accounts; rollback refuses to invent login identities or discard them.

### Customers

Authenticated admins and cashiers can `GET /customers` and `POST /customers`.
Create JSON: `name`, `whatsapp`, optional `address` and `notes`, and `whatsapp_opt_in` (defaults to false). Local Indonesian `08...` numbers normalize to `+628...`; international numbers use E.164 with 8–15 digits. Shared household phone numbers are allowed. Name is limited to 120 characters, address to 1000, and notes to 2000. Run migration 000004 before starting the API. Consent is stored only; WhatsApp delivery is configured manually and is not triggered by customer registration.
`n`PUT /customers/{id}` updates the complete editable customer profile. IDs and creation timestamps remain unchanged; invalid input returns field errors and unknown IDs return 404.
