This is a Fullstack technical test from iQuartz

# StockFlow

StockFlow is a small warehouse app. It manages products, stock and sales orders. The API is Go with PostgreSQL. The UI is React with TypeScript.

## Repository layout

```
stockflow/
├── backend/            Go service with cmd and internal packages
├── frontend/           React and TypeScript app built with Vite
├── docker-compose.yml  Postgres and the Go API
└── README.md
```

This checkout uses the StockFlow folder as the repository root.

## What is done

- Go API with product CRUD, stock changes, movement history and orders
- PostgreSQL migrations and seed data applied when the API starts
- React UI with a product table, summary cards, filters, a stock modal, an order form and order detail
- Docker Compose for Postgres and the API

## How to run

### Option one, Docker

Run this from the repository root.

```bash
docker compose up -d --build
```

Postgres and the API start together. The API applies migrations and seed data on startup, so there is no separate database step.

Postgres uses host port 5433 because many machines already use 5432. The API uses host port 8081. You can change both with `PG_PORT` and `API_PORT`.

Check that the stack works.

```bash
curl http://localhost:8081/healthz
curl 'http://localhost:8081/api/products?limit=3'
```

### Option two, local processes

Use any reachable Postgres. On a fresh database the API applies migrations and seed data at startup.

```bash
cd backend
export DATABASE_URL='postgres://stockflow:stockflow@localhost:5432/stockflow?sslmode=disable'
go run ./cmd/api
```

The Go server listens on 8080 unless you set `PORT`. Then start the UI.

```bash
cd ../frontend
npm install
cp .env.example .env
npm run dev
```

The example env file points `VITE_API_TARGET` at http://localhost:8081 for the Docker setup. If the API runs locally on 8080, change that value to http://localhost:8080. The browser only calls relative `/api` paths and Vite forwards them during development.

The seed has 14 products and two orders. One product has stock 6 so the low stock filter shows something. The seed SQL is idempotent, so restarting the API does not add rows twice.

## API

All responses use JSON. Errors use one shape everywhere.

```json
{"error": {"code": "BAD_REQUEST", "message": "validation failed", "details": ["sku is required"]}}
```

### GET /api/products

The query parameters are `search`, `page` and `limit`. Search matches sku or name without case sensitivity. Page starts at 1. Limit goes from 1 to 100 and defaults to 20. Each product has `last_movement_at` as an ISO string or null.

```bash
curl 'http://localhost:8081/api/products?search=kopi&page=1&limit=20'
```

The response carries `data`, `page`, `limit` and `total`. Invalid paging returns 400.

### POST /api/products

The body carries `sku`, `name`, `price` and an optional `is_active`. The sku and name are required after trimming. Price must be an integer of zero or more. Unknown fields return 400. A duplicate sku returns 409 with code `SKU_ALREADY_EXISTS`. Success returns 201 and the new product.

```bash
curl -X POST http://localhost:8081/api/products \
  -H 'Content-Type: application/json' \
  -d '{"sku":"KOP-002","name":"Kopi Robusta","price":18000}'
```

### PUT /api/products/{id}

This uses the same body. Leaving out `is_active` keeps the current value. A missing product returns 404 with code `PRODUCT_NOT_FOUND`.

### DELETE /api/products/{id}

This returns 204 and an empty body. A product with movement history or order items returns 409 with code `PRODUCT_HAS_HISTORY`. History is never deleted.

### POST /api/products/{id}/stock

The body carries `type`, `qty` and an optional `note`. Type is `in` or `out` and qty is an integer above zero. The stock change and the movement row are written in one transaction. Success returns 200 with the product and the movement. An overdraw returns 409 with the available quantity and writes nothing.

```bash
curl -X POST http://localhost:8081/api/products/1/stock \
  -H 'Content-Type: application/json' \
  -d '{"type":"out","qty":3,"note":"picked for order"}'
```

### GET /api/products/{id}/movements

This returns `data` with movements newest first. Note is null when the caller did not send one.

### POST /api/orders

The body carries `customer` and `items` with `sku` and `qty`. The client never sends a price. Validation runs before any write and returns 400 with details when something is wrong. Unit prices come from products and the server computes the total. All decrements, the order and its items are written in one transaction. Success returns 201 with the order and its items.

```bash
curl -X POST http://localhost:8081/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Andi","items":[{"sku":"KOP-001","qty":2},{"sku":"TEH-001","qty":1}]}'
```

If one item is short, the API returns 409 with the sku and available quantity and nothing is written.

### GET /api/orders/{id}

This returns the order with items, status, total and `cancelled_at`. A missing order returns 404.

### POST /api/orders/{id}/cancel

This restores every item to stock and returns the cancelled order. A second cancel returns 409 with code `ORDER_ALREADY_CANCELLED`. The status change and all restores are in one transaction.

## Frontend code

- `src/api/types.ts` holds every server shape in one module and domain data is never typed as `any`.
- `src/api/client.ts` is the only place that calls `fetch` or builds a URL. It throws `ApiError` for every non ok response.
- `ApiError` carries `status`, `code`, `message`, `details`, `available` and `sku` so a component can show field level feedback.
- `src/lib/format.ts` formats prices and relative times.
- `ProductsPage` loads products and renders summary cards, filters, the table and the stock modal.
- `NewOrderPage` manages a customer and a list of SKU and qty rows. It highlights the row that failed with 409.
- `OrderDetailPage` shows items, status and total. It cancels after a confirm dialog and links back to products.

## Design decisions

- Migrations and seed SQL are embedded in the Go binary. This keeps one source of truth for schema and data. A restart cannot apply them twice.
- `last_movement_at` is derived with a join and a max on the movements table, so there is no second copy of that value to keep in sync.
- Product search escapes `%`, `_` and `\` before using `ILIKE`. User input stays literal, but a normal index cannot help this query.
- One error pipeline builds every `{"error": {...}}` body. Unknown errors become `INTERNAL_ERROR` and are logged on the server.
- Stock uses a conditional `UPDATE` instead of a select followed by an update. The database decides if stock can go down, so concurrent orders cannot oversell.
- Cancel uses a conditional `UPDATE` on status, so two cancels cannot restore stock twice.
- Orders do not create movement rows. The order is the record of that stock change. This means `last_movement_at` only reflects manual adjustments.
- The UI fetches all products with pages of 100 and computes summary cards from that data. It avoids a new endpoint, but a large catalogue would need more requests.
- Plain CSS is enough here. The API contract and data safety matter more than visual polish.

## Verification

There is no automated test suite. We removed it on purpose and verify by hand.

Build checks

```bash
cd backend
go vet ./...
go build ./...

cd ../frontend
npm run build
npm run lint
```

Manual checks with Compose running

```bash
curl http://localhost:8081/healthz
curl 'http://localhost:8081/api/products?limit=3'
curl -i 'http://localhost:8081/api/products?page=0'
curl -X POST http://localhost:8081/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Andi","items":[{"sku":"KOP-001","qty":2}]}'
curl -X POST http://localhost:8081/api/orders/1/cancel
```

I checked list, search, validation, create product, duplicate sku, stock in, stock out, overdraw, movements, order create, short stock on an order, order detail, cancel and seed counts after a restart.

## Time spent

I spent 45 minutes on the boilerplates on day one and continued the next day.

- Product list endpoint, migrations and pgx, around an hour
- Products endpoints themselves, around an hour
- Stock adjustment and movement history, around an hour
- Order creation with validation, server prices and atomic decrements, around an hour
- Cancel and order detail, around 90 minutes
- Seed data, Compose, manual verification and error handling, around 80 minutes
- Types, API client and shell, 30 minutes
- Summary, filters, search and stock adjustment on the UI, 20 minutes
- Order UI, detail and cancel, 40 minutes

That is 9 hours and 5 minutes in total.

This is the screenshot from the next day only


## Future improvements

- Polish the UI more. It is still a simple table and a few forms.
- Add an `Idempotency-Key` header on `POST /api/orders` so a retry never creates a second order.
- Put the frontend in Compose as well, so one command runs Postgres, the API and the UI.
- Add `GET /api/reports/daily-sales?date=` and a summary card on the products page.
- Add a unique constraint on `order_id` and `sku` in `sales_order_items` as a database level guard against duplicate lines.
- Add a `pg_trgm` index if search gets slow on a larger catalogue.
- Record order driven stock changes in `stock_movements` so the movement feed is complete.
