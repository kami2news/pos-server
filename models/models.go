package models

import (
	"time"

	"github.com/lib/pq"
)


type Store struct {
	ID           string    `db:"id"            json:"id"`
	Name         string    `db:"name"          json:"name"`
	PasswordHash string    `db:"password_hash" json:"-"`
	ThemeColor   int       `db:"theme_color"   json:"theme_color"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

type BusinessDay struct {
	ID       string     `db:"id"        json:"id"`
	StoreID  string     `db:"store_id"  json:"store_id"`
	OpenedAt time.Time  `db:"opened_at" json:"opened_at"`
	ClosedAt *time.Time `db:"closed_at" json:"closed_at"`
	IsActive bool       `db:"is_active" json:"is_active"`
}

type Product struct {
	ID           string    `db:"id"            json:"id"`
	StoreID      string    `db:"store_id"      json:"store_id"`
	Name         string    `db:"name"          json:"name"`
	Price        int       `db:"price"         json:"price"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	IsActive     bool      `db:"is_active"     json:"is_active"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

type Order struct {
	ID             string     `db:"id"              json:"id"`
	StoreID        string     `db:"store_id"        json:"store_id"`
	BusinessDayID  string     `db:"business_day_id" json:"business_day_id"`
	OrderNumber    string     `db:"order_number"    json:"order_number"`
	Subtotal       int        `db:"subtotal"        json:"subtotal"`
	DiscountAmount       int `db:"discount_amount"        json:"discount_amount"`
	ManualDiscountAmount int `db:"manual_discount_amount" json:"manual_discount_amount"`
	TotalAmount    int        `db:"total_amount"    json:"total_amount"`
	Status         string     `db:"status"          json:"status"`
	PaymentMethod  *string    `db:"payment_method"  json:"payment_method"`
	ReceivedAmount *int       `db:"received_amount" json:"received_amount"`
	OrderedAt      time.Time  `db:"ordered_at"      json:"ordered_at"`
	UpdatedAt      time.Time  `db:"updated_at"      json:"updated_at"`
}

type DiscountRule struct {
	ID               string         `db:"id"                  json:"id"`
	StoreID          string         `db:"store_id"            json:"store_id"`
	Name             string         `db:"name"                json:"name"`
	IsActive         bool           `db:"is_active"           json:"is_active"`
	ConditionType    string         `db:"condition_type"      json:"condition_type"`
	TargetProductIDs pq.StringArray `db:"target_product_ids"  json:"target_product_ids"`
	MinQuantity      int            `db:"min_quantity"        json:"min_quantity"`
	MinAmount        int            `db:"min_amount"          json:"min_amount"`
	DiscountType     string         `db:"discount_type"       json:"discount_type"`
	DiscountValue    int            `db:"discount_value"      json:"discount_value"`
	CreatedAt        time.Time      `db:"created_at"          json:"created_at"`
}

type OrderItem struct {
	ID          string  `db:"id"           json:"id"`
	OrderID     string  `db:"order_id"     json:"order_id"`
	ProductID   *string `db:"product_id"   json:"product_id"`
	ProductName string  `db:"product_name" json:"product_name"`
	UnitPrice   int     `db:"unit_price"   json:"unit_price"`
	Quantity    int     `db:"quantity"     json:"quantity"`
}
