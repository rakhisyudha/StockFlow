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

- `backend/` Go module, runnable HTTP skeleton with `GET /healthz`.
- `frontend/` Vite React-TS scaffold, strict TypeScript, `/api` dev proxy configured.
- `docker-compose.yml` reserved placeholder, filled in during the Docker part.

## Planned parts

1. **Boilerplate** current
2. Database schema + first endpoint (`GET /api/products`)
3. Remaining product endpoints (create / update / delete / stock / movements)
4. Order endpoints + transactional stock safety
5. Migrations, seed data and Docker Compose
6. Backend tests (incl. concurrency)
7. Frontend foundations: shared types + API client + error envelope
8. Frontend: products page, summary cards, adjust-stock modal
9. Frontend: create order + order detail/cancel flows
10. Frontend unit tests, docs polish, final verification

## How to run (grows each part)

```bash
# backend
cd backend && go run ./cmd/api

# frontend
cd frontend && npm install && npm run dev
```

## API

To be documented as endpoints land.

## Design decisions

To be documented as decisions are made.

## Tests

To be documented as tests land.

## Improvements / honest hours

To be filled in at the end.
