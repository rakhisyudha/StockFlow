CREATE TABLE products (
    id         SERIAL PRIMARY KEY,
    sku        TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    price      INTEGER NOT NULL CHECK (price >= 0),
    stock      INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE stock_movements (
    id         SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id),
    type       TEXT NOT NULL CHECK (type IN ('in', 'out')),
    qty        INTEGER NOT NULL CHECK (qty > 0),
    note       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_movements_product_created
    ON stock_movements (product_id, created_at DESC);
