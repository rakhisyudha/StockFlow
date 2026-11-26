INSERT INTO products (sku, name, price, stock, is_active) VALUES
    ('KOP-001', 'Kopi Arabika Gayo', 25000, 48, TRUE),
    ('KOP-002', 'Kopi Robusta Lampung', 18000, 72, TRUE),
    ('TEH-001', 'Teh Hijau Organik', 15000, 36, TRUE),
    ('TEH-002', 'Teh Hitam Premium', 17000, 24, TRUE),
    ('TEH-003', 'Teh Putih Premium', 22000, 6, TRUE),
    ('GUL-001', 'Gula Pasir 1 kg', 16000, 60, TRUE),
    ('GUL-002', 'Gula Aren 500 g', 25000, 18, TRUE),
    ('SUS-001', 'Susu Kental Manis', 12000, 90, TRUE),
    ('AIR-001', 'Air Mineral 600 ml', 4000, 120, TRUE),
    ('MIN-001', 'Minyak Goreng 1 L', 18000, 30, TRUE),
    ('BER-001', 'Beras Pandan Wangi 5 kg', 65000, 15, TRUE),
    ('GAR-001', 'Garam Halus 500 g', 5000, 40, TRUE),
    ('SAB-001', 'Sabun Mandi Batang', 3500, 80, TRUE),
    ('KER-001', 'Keripik Singkong Pedas', 10000, 25, TRUE)
ON CONFLICT (sku) DO NOTHING;

INSERT INTO stock_movements (product_id, type, qty, note)
SELECT p.id, 'in', p.stock, 'seed: initial stock'
FROM products p
WHERE p.sku IN ('KOP-001', 'GUL-001', 'BER-001', 'TEH-003')
  AND NOT EXISTS (
      SELECT 1 FROM stock_movements m
      WHERE m.product_id = p.id AND m.note = 'seed: initial stock'
  );

INSERT INTO sales_orders (customer, status, total)
SELECT 'Toko Buah Nusantara', 'CREATED', 25000 * 2 + 15000 * 1
WHERE NOT EXISTS (SELECT 1 FROM sales_orders WHERE customer = 'Toko Buah Nusantara');

INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
SELECT o.id, p.sku, 2, p.price
FROM sales_orders o CROSS JOIN products p
WHERE o.customer = 'Toko Buah Nusantara'
  AND p.sku = 'KOP-001'
  AND NOT EXISTS (
      SELECT 1 FROM sales_order_items i WHERE i.order_id = o.id AND i.sku = p.sku
  );

INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
SELECT o.id, p.sku, 1, p.price
FROM sales_orders o CROSS JOIN products p
WHERE o.customer = 'Toko Buah Nusantara'
  AND p.sku = 'TEH-001'
  AND NOT EXISTS (
      SELECT 1 FROM sales_order_items i WHERE i.order_id = o.id AND i.sku = p.sku
  );

INSERT INTO sales_orders (customer, status, total, created_at, cancelled_at)
SELECT 'Warung Bu Sari', 'CANCELLED', 18000 * 3 + 4000 * 2, now(), now()
WHERE NOT EXISTS (SELECT 1 FROM sales_orders WHERE customer = 'Warung Bu Sari');

INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
SELECT o.id, p.sku, 3, p.price
FROM sales_orders o CROSS JOIN products p
WHERE o.customer = 'Warung Bu Sari'
  AND p.sku = 'KOP-002'
  AND NOT EXISTS (
      SELECT 1 FROM sales_order_items i WHERE i.order_id = o.id AND i.sku = p.sku
  );

INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
SELECT o.id, p.sku, 2, p.price
FROM sales_orders o CROSS JOIN products p
WHERE o.customer = 'Warung Bu Sari'
  AND p.sku = 'AIR-001'
  AND NOT EXISTS (
      SELECT 1 FROM sales_order_items i WHERE i.order_id = o.id AND i.sku = p.sku
  );
