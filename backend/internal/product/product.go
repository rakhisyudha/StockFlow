package product

import "time"

type Product struct {
	ID             int        `json:"id"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Price          int        `json:"price"`
	Stock          int        `json:"stock"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	LastMovementAt *time.Time `json:"last_movement_at"`
}

type ListParams struct {
	Search string
	Page   int
	Limit  int
}

type ListResponse struct {
	Data  []Product `json:"data"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
	Total int       `json:"total"`
}
