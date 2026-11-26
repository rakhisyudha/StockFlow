package order

import "time"

type CreateItem struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type CreateInput struct {
	Customer string       `json:"customer"`
	Items    []CreateItem `json:"items"`
}

type Item struct {
	ID        int    `json:"id"`
	SKU       string `json:"sku"`
	Qty       int    `json:"qty"`
	UnitPrice int    `json:"unit_price"`
}

type Order struct {
	ID          int        `json:"id"`
	Customer    string     `json:"customer"`
	Status      string     `json:"status"`
	Total       int        `json:"total"`
	CreatedAt   time.Time  `json:"created_at"`
	CancelledAt *time.Time `json:"cancelled_at"`
	Items       []Item     `json:"items"`
}
