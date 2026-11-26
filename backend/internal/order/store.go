package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/apierr"
)

type Store struct{ db *pgxpool.Pool }

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

func (s *Store) Create(ctx context.Context, in CreateInput) (Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin order: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	skus := make([]string, 0, len(in.Items))
	for _, item := range in.Items {
		skus = append(skus, item.SKU)
	}
	rows, err := tx.Query(ctx, `SELECT sku, price FROM products WHERE sku = ANY($1)`, skus)
	if err != nil {
		return Order{}, fmt.Errorf("load products: %w", err)
	}
	prices := make(map[string]int, len(skus))
	for rows.Next() {
		var sku string
		var price int
		if err := rows.Scan(&sku, &price); err != nil {
			rows.Close()
			return Order{}, fmt.Errorf("scan product price: %w", err)
		}
		prices[sku] = price
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Order{}, fmt.Errorf("iterate products: %w", err)
	}
	rows.Close()

	var unknown []string
	for _, item := range in.Items {
		if _, ok := prices[item.SKU]; !ok {
			unknown = append(unknown, fmt.Sprintf("unknown sku %s", item.SKU))
		}
	}
	if len(unknown) > 0 {
		return Order{}, apierr.BadRequest("validation failed", map[string]any{"details": unknown})
	}

	items := make([]Item, 0, len(in.Items))
	total := 0
	for _, item := range in.Items {
		var available int
		err := tx.QueryRow(ctx,
			`UPDATE products SET stock = stock - $2
			 WHERE sku = $1 AND stock >= $2 RETURNING stock`,
			item.SKU, item.Qty,
		).Scan(&available)
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE sku = $1)`, item.SKU).Scan(&exists); err != nil {
				return Order{}, fmt.Errorf("check product: %w", err)
			}
			if !exists {
				return Order{}, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
			}
			if err := tx.QueryRow(ctx, `SELECT stock FROM products WHERE sku = $1 FOR UPDATE`, item.SKU).Scan(&available); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return Order{}, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
				}
				return Order{}, fmt.Errorf("read available stock: %w", err)
			}
			return Order{}, apierr.Conflict("INSUFFICIENT_STOCK", "insufficient stock",
				map[string]any{"sku": item.SKU, "available": available})
		}
		if err != nil {
			return Order{}, fmt.Errorf("decrement stock: %w", err)
		}

		unitPrice := prices[item.SKU]
		total += unitPrice * item.Qty
		items = append(items, Item{SKU: item.SKU, Qty: item.Qty, UnitPrice: unitPrice})
	}

	var orderID int
	if err := tx.QueryRow(ctx, `INSERT INTO sales_orders (customer, total) VALUES ($1, $2) RETURNING id`, in.Customer, total).Scan(&orderID); err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO sales_order_items (order_id, sku, qty, unit_price) VALUES ($1, $2, $3, $4)`,
			orderID, item.SKU, item.Qty, item.UnitPrice,
		); err != nil {
			return Order{}, fmt.Errorf("insert order item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit order: %w", err)
	}

	return s.Get(ctx, orderID)
}

func (s *Store) Get(ctx context.Context, id int) (Order, error) {
	var o Order
	err := s.db.QueryRow(ctx,
		`SELECT id, customer, status, total, created_at, cancelled_at
		 FROM sales_orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.Customer, &o.Status, &o.Total, &o.CreatedAt, &o.CancelledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, apierr.NotFound("ORDER_NOT_FOUND", "order not found")
	}
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, sku, qty, unit_price FROM sales_order_items WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return Order{}, fmt.Errorf("load order items: %w", err)
	}
	defer rows.Close()

	o.Items = make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.SKU, &item.Qty, &item.UnitPrice); err != nil {
			return Order{}, fmt.Errorf("scan order item: %w", err)
		}
		o.Items = append(o.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate order items: %w", err)
	}
	return o, nil
}
