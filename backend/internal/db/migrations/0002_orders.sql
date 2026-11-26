CREATE TABLE sales_orders (
    id           SERIAL PRIMARY KEY,
    customer     TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'CREATED' CHECK (status IN ('CREATED', 'CANCELLED')),
    total        INTEGER NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    cancelled_at TIMESTAMPTZ
);

CREATE TABLE sales_order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INTEGER NOT NULL REFERENCES sales_orders(id),
    sku        TEXT NOT NULL REFERENCES products(sku),
    qty        INTEGER NOT NULL CHECK (qty > 0),
    unit_price INTEGER NOT NULL
);

CREATE INDEX idx_sales_order_items_order ON sales_order_items (order_id);
