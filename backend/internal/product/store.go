package product

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ db *pgxpool.Pool }

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

const (
	whereClause = `WHERE ($1 = '' OR p.sku ILIKE $2 ESCAPE '\' OR p.name ILIKE $2 ESCAPE '\')`

	listQuery = `
SELECT p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at,
       max(m.created_at) FILTER (WHERE m.id IS NOT NULL) AS last_movement_at
FROM products p
LEFT JOIN stock_movements m ON m.product_id = p.id
` + whereClause + `
GROUP BY p.id
ORDER BY p.id
LIMIT $3 OFFSET $4`

	countQuery = `SELECT count(*) FROM products p ` + whereClause
)

func (s *Store) List(ctx context.Context, p ListParams) (ListResponse, error) {
	pattern := "%" + escapeLike(p.Search) + "%"
	offset := (p.Page - 1) * p.Limit

	rows, err := s.db.Query(ctx, listQuery, p.Search, pattern, p.Limit, offset)
	if err != nil {
		return ListResponse{}, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	items := make([]Product, 0)
	for rows.Next() {
		var item Product
		if err := rows.Scan(
			&item.ID, &item.SKU, &item.Name, &item.Price, &item.Stock,
			&item.IsActive, &item.CreatedAt, &item.LastMovementAt,
		); err != nil {
			return ListResponse{}, fmt.Errorf("scan product: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResponse{}, fmt.Errorf("iterate products: %w", err)
	}

	var total int
	if err := s.db.QueryRow(ctx, countQuery, p.Search, pattern).Scan(&total); err != nil {
		return ListResponse{}, fmt.Errorf("count products: %w", err)
	}
	return ListResponse{Data: items, Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
