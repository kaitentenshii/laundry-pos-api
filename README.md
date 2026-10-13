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

`PUT /customers/{id}` updates the complete editable customer profile. IDs and creation timestamps remain unchanged; invalid input returns field errors and unknown IDs return 404.

Customer list accepts `q` (literal case-insensitive name or normalized phone substring), `consent=true|false`, `page` (default 1), and `page_size` (default 20, maximum 100). Returns `{data,total,page,page_size}` with consistent totals and deterministic newest-first ordering. Search metacharacters are treated literally.

`GET /customers/{id}` returns the current customer profile or 404. `order_count` is zero until Sprint 2 order storage is implemented. Frontend detail links preserve `customer_id` when navigating to the future order creation route.

### Laundry services

Admins manage `POST /services` and `PUT /services/{id}` with `name`, `unit` (`KG` or `ITEM`), integer rupiah `price` (0–999999999), `estimated_hours` (1–8760), and `is_active`. Create defaults to active; updates require an explicit status. `GET /services` allows admins and cashiers; cashier responses always exclude inactive records, even with `active=false`. UI labels KG as kg and ITEM as pcs; 24 hours equals one day. Migration 000005 is required. IDs remain stable and services are never hard-deleted through the API.

### Price history and the order integration contract

Migration 000006 records immutable versions of name, unit, price, and processing estimate. Every service write records changed pricing atomically; status-only changes keep the current version. Responses include `price_version_id`. Admins can inspect `GET /services/{id}/prices`. Existing versions cannot be updated. When Sprint 2 introduces order items, it must store the selected version ID plus name/unit/unit-price snapshots at order creation and calculate old totals from those item snapshots, never by joining mutable master prices. Actual order creation remains Sprint 2 scope.

`PATCH /services/{id}/status` accepts an explicit boolean `is_active` and requires admin access. Status changes preserve identity and all price versions. Cashier `GET /services` always excludes inactive services. The Sprint 2 order creation handler must recheck active status inside its write transaction; a previously loaded selector is not authorization to order an inactive service.

### Order creation

Migration 000007 adds orders and immutable transaction item snapshots. Authenticated admin/cashier `POST /orders` accepts `customer_id`, RFC3339 `due_at`, optional `notes`, one or more `items: [{service_id, quantity: "4.125"}]`, and a 16–100 character `request_key`. Quantity is decimal text with up to three digits after the decimal point; ITEM requires an integer. Money is integer rupiah, rounded half-up independently per line, then summed. Only active services can be selected; current name/unit/price/version are read and locked inside the transaction. Initial statuses are RECEIVED and UNPAID. Identical retries by the same user return the original order; a reused key with different data is rejected. `GET /orders/{id}` returns stored transaction snapshots, not current master prices. Totals are capped at 9,000,000,000,000 rupiah.

Orders support up to 30 service lines. Each KG line accepts three decimal places; ITEM lines require whole pieces. If any line is invalid the entire transaction is rejected.
The frontend sends expected_total. A mismatch with the locked current prices rejects creation and prompts the user to review refreshed prices. Both layers use integer arithmetic and half-up rounding for each line.
Order numbers use the database identity sequence (LD-<id>) and a unique database constraint. Concurrent inserts cannot share an identity; concurrent retries with one request key resolve to one order.
GET /orders returns data,total,page,page_size. Filters: status, payment_status, from/to (inclusive calendar dates in UTC+7), page and page_size (max100). Ordering is newest-first. Totals and items are read under one repeatable-read snapshot.
Order detail includes creator name, contact snapshots, all item snapshots, total, paid_amount, balance, due time, and notes. Customer order_count now counts persisted orders; no static example count remains.
Order list q searches order number and the transaction customer-name snapshot case-insensitively. SQL wildcard characters are escaped as literal text; search combines with all list filters and pagination.

### Manual payment

Migration 000008 adds one full settlement per order. POST /orders/{id}/payments accepts method CASH/TRANSFER/QRIS, integer amount_received, RFC3339 paid_at, optional notes, confirmed (required true for TRANSFER/QRIS), and request_key. Underpayment is rejected. Cash change equals received minus the full balance; noncash change is zero. Payment row and PAID status commit atomically while laundry status stays unchanged. Replaying the same request returns the same payment; a different request after settlement returns 409. The order row lock and unique order_id protect concurrent requests. Future payment dates and dates before order creation are rejected.
Paid amounts in list/detail are derived from persisted settlements. Laundry status and payment status remain independent; detail includes the payment receipt for method, time, recorder, received funds, settlement, and cash change.

### Laundry status

Migration 000009 adds history and initial RECEIVED records. PATCH /orders/{id}/status accepts from_status, status, request_key. Allowed sequence is RECEIVED -> WASHING -> COMPLETED -> PICKED_UP, with PAID required for pickup. Orders are locked; stale expected states, jumps, and reversals return 409. A successful transition updates the order and inserts actor/status/time history atomically. Identical retries create no extra history. WhatsApp delivery is manual/outside this scope and never blocks status changes.
Order responses expose allowed_next_statuses and pickup_blocked_reason. The API is authoritative: skipping/reversing states and stale actions cannot mutate orders or history; COMPLETED -> PICKED_UP additionally requires PAID.
