package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/apierr"
)

type Store struct{ db *pgxpool.Pool }

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

const (
	productColumns = `id, sku, name, price, stock, is_active, created_at,
		NULL::timestamptz AS last_movement_at`

	productWithMovement = `p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at,
		max(m.created_at) FILTER (WHERE m.id IS NOT NULL) AS last_movement_at
	FROM products p LEFT JOIN stock_movements m ON m.product_id = p.id`

	whereClause = `WHERE ($1 = '' OR p.sku ILIKE $2 ESCAPE '\' OR p.name ILIKE $2 ESCAPE '\')`

	listQuery = `SELECT ` + productWithMovement + ` ` + whereClause + `
		GROUP BY p.id ORDER BY p.id LIMIT $3 OFFSET $4`

	getQuery = `SELECT ` + productWithMovement + ` WHERE p.id = $1 GROUP BY p.id`

	movementColumns = `id, product_id, type, qty, note, created_at`
)

type scanner interface {
	Scan(dest ...any) error
}

func scanProduct(row scanner) (Product, error) {
	var p Product
	err := row.Scan(
		&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock,
		&p.IsActive, &p.CreatedAt, &p.LastMovementAt,
	)
	return p, err
}

func scanMovement(row scanner) (Movement, error) {
	var m Movement
	err := row.Scan(&m.ID, &m.ProductID, &m.Type, &m.Qty, &m.Note, &m.CreatedAt)
	return m, err
}

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
		item, err := scanProduct(rows)
		if err != nil {
			return ListResponse{}, fmt.Errorf("scan product: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResponse{}, fmt.Errorf("iterate products: %w", err)
	}

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM products p `+whereClause, p.Search, pattern).Scan(&total); err != nil {
		return ListResponse{}, fmt.Errorf("count products: %w", err)
	}
	return ListResponse{Data: items, Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *Store) Get(ctx context.Context, id int) (Product, error) {
	p, err := scanProduct(s.db.QueryRow(ctx, getQuery, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (s *Store) Create(ctx context.Context, in Input) (Product, error) {
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}
	p, err := scanProduct(s.db.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, is_active) VALUES ($1, $2, $3, $4)
		 RETURNING `+productColumns,
		in.SKU, in.Name, in.Price, isActive,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return Product{}, apierr.Conflict("SKU_ALREADY_EXISTS", "sku already exists", map[string]any{"sku": in.SKU})
		}
		return Product{}, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

func (s *Store) Update(ctx context.Context, id int, in Input) (Product, error) {
	p, err := scanProduct(s.db.QueryRow(ctx,
		`UPDATE products SET sku = $1, name = $2, price = $3, is_active = COALESCE($4, is_active)
		 WHERE id = $5 RETURNING `+productColumns,
		in.SKU, in.Name, in.Price, in.IsActive, id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
	}
	if err != nil {
		if isUniqueViolation(err) {
			return Product{}, apierr.Conflict("SKU_ALREADY_EXISTS", "sku already exists", map[string]any{"sku": in.SKU})
		}
		return Product{}, fmt.Errorf("update product: %w", err)
	}
	return p, nil
}

func (s *Store) Delete(ctx context.Context, id int) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedID int
	if err := tx.QueryRow(ctx, `SELECT id FROM products WHERE id = $1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return fmt.Errorf("lock product: %w", err)
	}

	var hasHistory bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_movements WHERE product_id = $1)`, id).Scan(&hasHistory); err != nil {
		return fmt.Errorf("check history: %w", err)
	}
	if hasHistory {
		return apierr.Conflict("PRODUCT_HAS_HISTORY", "product has movement history", nil)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM products WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete: %w", err)
	}
	return nil
}

func (s *Store) AdjustStock(ctx context.Context, id int, in StockInput) (StockResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return StockResponse{}, fmt.Errorf("begin stock adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	delta := in.Qty
	if in.Type == "out" {
		delta = -in.Qty
	}

	var updatedStock int
	err = tx.QueryRow(ctx,
		`UPDATE products SET stock = stock + $2
		 WHERE id = $1 AND stock + $2 >= 0 RETURNING stock`,
		id, delta,
	).Scan(&updatedStock)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, id).Scan(&exists); err != nil {
			return StockResponse{}, fmt.Errorf("check product: %w", err)
		}
		if !exists {
			return StockResponse{}, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		var available int
		if err := tx.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1 FOR UPDATE`, id).Scan(&available); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return StockResponse{}, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
			}
			return StockResponse{}, fmt.Errorf("read available stock: %w", err)
		}
		return StockResponse{}, apierr.Conflict("INSUFFICIENT_STOCK", "insufficient stock", map[string]any{"available": available})
	}
	if err != nil {
		return StockResponse{}, fmt.Errorf("update stock: %w", err)
	}

	var note *string
	if in.Note != "" {
		note = &in.Note
	}
	movement, err := scanMovement(tx.QueryRow(ctx,
		`INSERT INTO stock_movements (product_id, type, qty, note)
		 VALUES ($1, $2, $3, $4) RETURNING `+movementColumns,
		id, in.Type, in.Qty, note,
	))
	if err != nil {
		return StockResponse{}, fmt.Errorf("insert movement: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return StockResponse{}, fmt.Errorf("commit stock adjustment: %w", err)
	}

	product, err := s.Get(ctx, id)
	if err != nil {
		return StockResponse{}, err
	}
	return StockResponse{Product: product, Movement: movement}, nil
}

func (s *Store) Movements(ctx context.Context, id int) ([]Movement, error) {
	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, id).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check product: %w", err)
	}
	if !exists {
		return nil, apierr.NotFound("PRODUCT_NOT_FOUND", "product not found")
	}

	rows, err := s.db.Query(ctx,
		`SELECT `+movementColumns+` FROM stock_movements WHERE product_id = $1
		 ORDER BY created_at DESC, id DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	items := make([]Movement, 0)
	for rows.Next() {
		m, err := scanMovement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movements: %w", err)
	}
	return items, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
