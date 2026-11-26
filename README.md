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

- `backend/` Go service with full product, stock and order lifecycle.
- `frontend/` Vite React-TS scaffold, strict TypeScript, configurable `/api` dev proxy.
- `docker-compose.yml` brings up Postgres + API with migrations and seed applied.

## Completed parts

1. **Boilerplate**
2. **Database + first endpoint** (`GET /api/products`)
3. **Product write endpoints** (create / update / delete)
4. **Stock adjustment + movement history**
5a. **Order creation** (validation, server-side pricing, atomic decrements)
5b. **Order detail + cancel** (stock restore)
6. **Seed data + Docker Compose**

## Planned parts

7. Frontend foundations: shared types + API client + error envelope
8. Frontend: products page, summary cards, adjust-stock modal
9. Frontend: create order + order detail/cancel flows
10. Docs polish and final verification

## How to run

### Option A — one command (Docker)

```bash
docker compose up -d --build
```

This starts Postgres and the Go API. The API applies migrations and seed data on
startup, so no manual DB step is needed.

| Service | Host port | Notes |
| --- | --- | --- |
| Postgres | `5433` (override `PG_PORT`) | local 5432 is left free |
| API | `8081` (override `API_PORT`) | local 8080 is left free |

```bash
curl http://localhost:8081/healthz
curl 'http://localhost:8081/api/products?limit=3'
```

### Option B — local processes

```bash
# any reachable Postgres; on a fresh DB migrations + seed run at startup
cd backend
export DATABASE_URL='postgres://stockflow:stockflow@localhost:5432/stockflow?sslmode=disable'
go run ./cmd/api            # listens on 8080 unless PORT is set

cd ../frontend
npm install
cp .env.example .env        # VITE_API_TARGET defaults to http://localhost:8081
npm run dev
```

Seed data: 14 realistic products (one with stock 6 for the low-stock filter),
movement rows for a few of them, one `CREATED` order and one `CANCELLED` order.
Seed SQL is idempotent, so restarting the API does not duplicate rows.

## API

### `GET /api/products`

Query parameters: `search` (matches `sku` or `name`, case-insensitive, literal
`%`, `_` and `\` are escaped), `page` (>= 1, default 1), `limit` (1–100, default 20).

```bash
curl 'http://localhost:8081/api/products?search=kopi&page=1&limit=20'
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
curl -X POST http://localhost:8081/api/products \
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
curl -i -X DELETE http://localhost:8081/api/products/1
```

### `POST /api/products/{id}/stock`

Body: `{"type":"in"|"out","qty":n,"note":"..."}`. `qty` must be an integer > 0;
`type` must be `in` or `out`; `note` is optional. The stock change and the
`stock_movements` row are written in one transaction. Success returns `200` with
`{"product": {...}, "movement": {...}}`.

```bash
curl -X POST http://localhost:8081/api/products/1/stock \
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

### `POST /api/orders`

Body: `{"customer":"...","items":[{"sku":"...","qty":n}]}`. Prices are never
accepted from the client — an `unit_price` field is rejected as unknown.

Validation runs before any write: customer required, at least one item, every qty
an integer > 0, no duplicate skus, every sku must exist. Problems return
`400 BAD_REQUEST` with a `details` array, e.g.:

```json
{"error": {"code": "BAD_REQUEST", "message": "validation failed", "details": ["duplicate sku KOP-001"]}}
```

`unit_price` is read from `products` and `total` is computed server-side. All
decrements, the order row and its item rows are written in one transaction. Success
returns `201` with the order (`status: "CREATED"`, `cancelled_at: null`) and items
carrying server-computed prices.

```bash
curl -X POST http://localhost:8081/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Andi","items":[{"sku":"KOP-001","qty":2},{"sku":"TEH-001","qty":1}]}'
```

If any item is short, nothing is written and the failing sku is identified:

```json
{"error": {"code": "INSUFFICIENT_STOCK", "message": "insufficient stock", "sku": "GULA-001", "available": 3}}
```

### `GET /api/orders/{id}`

Returns the order with items, status, total and `cancelled_at`. `404 ORDER_NOT_FOUND`
when missing, `400 BAD_REQUEST` for a non-numeric id.

### `POST /api/orders/{id}/cancel`

Restores every item's quantity to stock and returns `200` with the cancelled order.
`409 ORDER_ALREADY_CANCELLED` on a second cancel, `404 ORDER_NOT_FOUND` when missing.
The status change and all restores are in one transaction, so a failed restore
leaves the order active.

```bash
curl -X POST http://localhost:8081/api/orders/1/cancel
```

## Design decisions

- **Migrations are embedded in the Go binary** (`//go:embed`) and applied through a
  `schema_migrations` table guarded by a Postgres advisory lock. *Trade-off:* the
  schema version is coupled to the binary (no external migration tool), but local
  runs and containers all share one source of truth and concurrent starts
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
  than a flat design, but predictable navigation and clear ownership per endpoint.
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
- **Order validation is two-phase:** shape/duplicates in the handler, existence of
  all skus via one `WHERE sku = ANY(...)` query in the store — both before any
  write. *Trade-off:* one extra round trip inside the transaction; gains clear,
  complete 400 messages and no half-created order.
- **Order decrements reuse the same guarded UPDATE as manual stock adjustment.**
  *Trade-off:* duplicated SQL text instead of a shared helper; keeps each store's
  transaction boundary explicit, and both paths inherit the same non-negative
  guarantee. A shared helper is the cleanup candidate if a third writer appears.
- **`DisallowUnknownFields` moved to `api.DecodeJSON`** and is shared by product
  and order domains. *Trade-off:* strict input contracts; a client-sent
  `unit_price` or typo'd field fails loudly instead of being silently ignored.
- **Cancel is guarded by `WHERE status = 'CREATED'`**, not a read-then-write.
  *Trade-off:* one conditional UPDATE; concurrent cancels cannot restore stock
  twice — exactly one wins, the rest get `409 ORDER_ALREADY_CANCELLED`.
- **Order creation and cancel intentionally do not write `stock_movements` rows.**
  *Trade-off:* `last_movement_at` reflects manual adjustments only, not order
  activity; the order row is the audit record for order-driven stock changes.
  This keeps the order paths from Part 5a unchanged and avoids mixing two kinds
  of history. Revisit if the UI needs a unified movement feed.
- **Delete history guard covers orders too**, via a second `EXISTS` on
  `sales_order_items`. *Trade-off:* one extra check per delete; otherwise an
  order-backed product would hit a raw FK violation (500) instead of the
  documented `409 PRODUCT_HAS_HISTORY`.
- **`api.ParseID` is shared** by product and order handlers. *Trade-off:* one
  small shared function instead of two copies that could drift.
- **Seeds are embedded SQL applied by the API**, not mounted into
  `/docker-entrypoint-initdb.d`. *Trade-off:* the API must be running before data
  exists, but schema and seed share the same versioned binary and the SQL is
  idempotent — restarting never duplicates rows and there is no second copy of
  the schema to keep in sync.
- **Compose uses host ports 5433/8081** via variables. *Trade-off:* non-default
  ports to remember; avoids colliding with an existing local Postgres or service
  on 8080, and both are overridable with `PG_PORT` / `API_PORT`.
- **API waits for a healthy Postgres** through `depends_on: condition: service_healthy`.
  *Trade-off:* slightly slower startup; avoids crash-looping on a database that
  is not accepting connections yet.
- **Vite proxy target is configurable** through `VITE_API_TARGET`. *Trade-off:*
  one extra env file; the same frontend works against the Compose API or a local
  `go run` without code edits.

## Verification (manual)

There is no automated test suite; verification is done against the running stack.
Build checks: `go vet ./... && go build ./...` in `backend/`, and
`npm run build && npm run lint` in `frontend/`.

```bash
docker compose up -d --build

curl http://localhost:8081/healthz
curl 'http://localhost:8081/api/products?limit=3'
curl -i 'http://localhost:8081/api/products?page=0'

curl -X POST http://localhost:8081/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Andi","items":[{"sku":"KOP-001","qty":2}]}'

curl -i -X POST http://localhost:8081/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Andi","items":[{"sku":"TEH-003","qty":99}]}'

curl -X POST http://localhost:8081/api/orders/1/cancel
curl http://localhost:8081/api/orders/1
```

These cover the happy path, validation errors, insufficient stock, cancellation
and stock restoration end to end.

## Improvements / honest hours

To be filled in at the end.
