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

- `backend/` Go service with `GET /healthz` and `GET /api/products`.
- `frontend/` Vite React-TS scaffold, strict TypeScript, `/api` dev proxy configured.
- `docker-compose.yml` reserved placeholder, filled in during the Docker part.

## Completed parts

1. **Boilerplate**
2. **Database + first endpoint** (`GET /api/products`)

## Planned parts

3. Remaining product endpoints (create / update / delete / stock / movements)
4. Order endpoints + transactional stock safety
5. Seed data and Docker Compose
6. Backend concurrency tests
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

## Tests

```bash
cd backend
go test ./...                                   # unit tests; DB test skips
DATABASE_URL='postgres://...' go test ./... -count=1   # + httptest integration
```

The integration test uses `TRUNCATE`, so point it at a disposable database.

## Improvements / honest hours

To be filled in at the end.
