This is a Fullstack technical test from iQuartz

# StockFlow

Warehouse supervisor app: manage products, adjust stock, and record stock-consuming
sales orders. Go + PostgreSQL API, React + TypeScript UI.

## Repository layout

```
stockflow/
├── backend/            # Go service (cmd/, internal/)
├── frontend/           # React + TypeScript (Vite)
├── docker-compose.yml  # Postgres + migrations + seed
└── README.md
```

In this checkout the repository root is the `StockFlow/` directory itself and maps
to `stockflow/` in the layout above.

## Current state

- `backend/` Go service with product CRUD, stock adjustment and movement history.
- `frontend/` Vite React-TS scaffold, strict TypeScript, `/api` dev proxy configured.
- `docker-compose.yml` reserved placeholder, filled in during the Docker part.

## Completed parts

1. **Boilerplate**
2. **Database + first endpoint** (`GET /api/products`)
3. **Product write endpoints** (create / update / delete)
4. **Stock adjustment + movement history**

## Planned parts

5. Order endpoints + transactional stock safety
6. Seed data and Docker Compose
7. Frontend foundations: shared types + API client + error envelope
8. Frontend: products page, summary cards, adjust-stock modal
9. Frontend: create order + order detail/cancel flows
10. Frontend unit tests, docs polish, final verification

## How to run

```bash
# backend (needs a reachable Postgres)
cd backend
export DATABASE_URL='postgres://stockflow:stockflow@localhost:5432/stockflow?sslmode=disable'
go run ./cmd/api

# frontend
cd frontend && npm install && npm run dev
```

Migrations are applied automatically on startup, so a fresh database is enough.

## API

### `GET /api/products`

Query parameters: `search` (matches `sku` or `name`, case-insensitive, literal
`%`, `_` and `\` are escaped), `page` (>= 1, default 1), `limit` (1–100, default 20).

```bash
curl 'http://localhost:8080/api/products?search=kopi&page=1&limit=20'
```

```json
{
  "data": [
    {
      "id": 1,
      "sku": "KOP-001",
      "name": "Kopi Arabika",
      "price": 25000,
      "stock": 50,
      "is_active": true,
      "created_at": "2026-11-26T07:23:33.123456Z",
      "last_movement_at": "2026-11-26T07:23:33.123456Z"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 1
}
```

Invalid paging returns `400` with the shared error envelope:

```json
{"error": {"code": "BAD_REQUEST", "message": "limit must be an integer between 1 and 100"}}
```

### `POST /api/products`

Body: `{"sku":"...","name":"...","price":n,"is_active":true}`. `sku` and `name` are
required after trimming, `price` must be an integer >= 0, `is_active` defaults to
`true`. Unknown fields are rejected with `400 BAD_REQUEST`; field problems are
returned as `{"error":{"code":"BAD_REQUEST","message":"validation failed","details":[...]}}`.
A duplicate sku returns `409 SKU_ALREADY_EXISTS` with `{"sku":"..."}`. Success is
`201` with the created product (`stock: 0`, `last_movement_at: null`).

```bash
curl -X POST http://localhost:8080/api/products \
  -H 'Content-Type: application/json' \
  -d '{"sku":"KOP-002","name":"Kopi Robusta","price":18000}'
```

### `PUT /api/products/{id}`

Same body as create. `is_active` may be omitted to keep the current value.
Returns `200` with the updated product, `404 PRODUCT_NOT_FOUND` when missing,
or the same validation/duplicate errors as create.

### `DELETE /api/products/{id}`

Returns `204` with an empty body, `404 PRODUCT_NOT_FOUND` when missing, or
`409 PRODUCT_HAS_HISTORY` when any movement references the product. History is
never deleted.

```bash
curl -i -X DELETE http://localhost:8080/api/products/1
```

### `POST /api/products/{id}/stock`

Body: `{"type":"in"|"out","qty":n,"note":"..."}`. `qty` must be an integer > 0;
`type` must be `in` or `out`; `note` is optional. The stock change and the
`stock_movements` row are written in one transaction. Success returns `200` with
`{"product": {...}, "movement": {...}}`.

```bash
curl -X POST http://localhost:8080/api/products/1/stock \
  -H 'Content-Type: application/json' \
  -d '{"type":"out","qty":3,"note":"picked for order"}'
```

Overdraw returns `409`:

```json
{"error": {"code": "INSUFFICIENT_STOCK", "message": "insufficient stock", "available": 2}}
```

Nothing is written on that path — no movement row, no stock change. Missing product
returns `404 PRODUCT_NOT_FOUND`.

### `GET /api/products/{id}/movements`

Returns `{"data": [...]}` with movements newest first (`created_at DESC, id DESC`).
`note` is `null` when omitted.

## Design decisions

- **Migrations are embedded in the Go binary** (`//go:embed`) and applied through a
  `schema_migrations` table guarded by a Postgres advisory lock. *Trade-off:* the
  schema version is coupled to the binary (no external migration tool), but local
  runs, tests and containers all share one source of truth and concurrent starts
  cannot double-apply.
- **`last_movement_at` is derived**, via `LEFT JOIN stock_movements` + `MAX` in the
  list query. *Trade-off:* one extra join/aggregate per list request, but no
  duplicated state and no second write path that could drift. Revisit with a
  maintained column only if list latency becomes a problem.
- **Search escapes LIKE wildcards** and uses unanchored `ILIKE`. *Trade-off:* simple
  and correct for literal user input, but a btree index cannot help. A `pg_trgm`
  GIN index is the upgrade path if the catalogue grows.
- **One error pipeline:** `apierr.Error` is transport-neutral and carries
  status/code/message/extra fields; `api.WriteError` is the only place that builds
  `{"error": {...}}`. Unknown errors become `INTERNAL_ERROR` and are logged
  server-side rather than leaked. New endpoints reuse this path unchanged.
- **Fixed package pattern:** every domain gets `model + store + handler` in its own
  package, wired by `cmd/api`, routed by `internal/server`. *Trade-off:* more files
  than a flat design, but predictable navigation and a clean seam for tests.
- **Bounded pool and paging:** `pgxpool` capped at 1–10 connections and `limit`
  capped at 100. *Trade-off:* strict 400s for bad parameters instead of silent
  coercion, which prevents accidental unbounded scans.
- **One shared input decoder** for create/update: trims strings, rejects unknown
  fields, and collects validation problems into `details`. *Trade-off:* stricter
  than a permissive decoder (a typo'd field is a 400), but bad writes cannot slip
  through and both endpoints cannot drift apart.
- **Uniqueness is enforced by the DB constraint**, then the `23505` error is mapped
  to `409 SKU_ALREADY_EXISTS`. *Trade-off:* a pre-check would be simpler to read but
  racy; the constraint is the single source of truth.
- **Delete guards inside one transaction** with `SELECT ... FOR UPDATE` on the
  product row before checking movement history. *Trade-off:* one short lock per
  delete; prevents a concurrent movement from slipping in between the check and
  the delete, which would orphan history.
- **Stock is changed by a guarded UPDATE**, never a read-then-write:
  `UPDATE products SET stock = stock + $2 WHERE id = $1 AND stock + $2 >= 0
  RETURNING stock`. *Trade-off:* one conditional statement instead of a simpler
  SELECT/UPDATE pair; the database becomes the only arbiter of non-negative stock,
  so concurrent requests cannot oversell. The row is locked with `FOR UPDATE`
  **only on the failure path**, to report an accurate `available` value.
- **Stock response carries both the product and the movement.** *Trade-off:* a
  slightly larger body than a bare product; the UI can update stock and history
  from one response instead of a second round trip.

## Tests

```bash
cd backend
go test ./...                                   # unit tests; DB test skips
DATABASE_URL='postgres://...' go test ./... -count=1   # + httptest integration
```

The integration tests use `TRUNCATE`, so point them at a disposable database.
They cover list/search/validation, create/duplicate sku, update/404, the delete
history guard, stock happy path, overdraw-with-nothing-written, movement ordering,
and a 20-goroutine concurrent-decrement test (10 succeed, 10 get 409, stock ends
at 0 with exactly 10 movements).

`go test -race` additionally checks for Go data races. It needs `CGO_ENABLED=1`
and a C compiler; this Windows environment has neither gcc nor clang, so it could
not be run here — the concurrency test itself passes with the race detector off.

## Improvements / honest hours

To be filled in at the end.
