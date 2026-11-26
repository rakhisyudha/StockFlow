package product

import "time"

type Movement struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	Type      string    `json:"type"`
	Qty       int       `json:"qty"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type StockResponse struct {
	Product  Product  `json:"product"`
	Movement Movement `json:"movement"`
}

type MovementListResponse struct {
	Data []Movement `json:"data"`
}
