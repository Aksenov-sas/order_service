// Package models содержит структуры данных для работы с заказами
package models

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
)

// Экземпляр кастомного валидатора
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Order представляет структуру заказа
type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,alphanum,len=32"`
	TrackNumber       string    `json:"track_number" validate:"required"`
	Entry             string    `json:"entry" validate:"required"`
	Delivery          Delivery  `json:"delivery" validate:"required,dive"`
	Payment           Payment   `json:"payment" validate:"required,dive"`
	Items             []Item    `json:"items" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" validate:"required"`
	ShardKey          string    `json:"shardkey" validate:"required"`
	SMID              int       `json:"sm_id" validate:"required,gt=0"`
	DateCreated       time.Time `json:"date_created"`
	OOFShard          string    `json:"oof_shard" validate:"required"`
}

// Validate выполняет строгую проверку заказа, полученного от брокера.
func (o *Order) Validate() error {
	if o == nil {
		return errors.New("order is nil")
	}
	return validate.Struct(o)
}

// Delivery представляет информацию о доставке
type Delivery struct {
	OrderUID string `json:"-"`
	Name     string `json:"name" validate:"required"`
	Phone    string `json:"phone" validate:"required"`
	Zip      string `json:"zip" validate:"required"`
	City     string `json:"city" validate:"required"`
	Address  string `json:"address" validate:"required"`
	Region   string `json:"region" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
}

// Подтверждение деталей доставки.
func (d *Delivery) Validate() error {
	return validate.Struct(d)
}

// Payment представляет информацию о платеже
type Payment struct {
	OrderUID     string `json:"-"`
	Transaction  string `json:"transaction" validate:"required"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency" validate:"required"`
	Provider     string `json:"provider" validate:"required"`
	Amount       int    `json:"amount" validate:"min=0"`
	PaymentDT    int64  `json:"payment_dt" validate:"gt=0"`
	Bank         string `json:"bank" validate:"required"`
	DeliveryCost int    `json:"delivery_cost" validate:"min=0"`
	GoodsTotal   int    `json:"goods_total" validate:"min=0"`
	CustomFee    int    `json:"custom_fee" validate:"min=0"`
}

// Подтверждение платежа.
func (p *Payment) Validate() error {
	return validate.Struct(p)
}

// Item представляет товар в заказе
type Item struct {
	OrderUID    string `json:"-"`
	ChrtID      int    `json:"chrt_id" validate:"gt=0"`
	TrackNumber string `json:"track_number" validate:"required"`
	Price       int    `json:"price" validate:"min=0"`
	RID         string `json:"rid" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Sale        int    `json:"sale"`
	Size        string `json:"size" validate:"required"`
	TotalPrice  int    `json:"total_price" validate:"min=0"`
	NMID        int    `json:"nm_id" validate:"gt=0"`
	Brand       string `json:"brand" validate:"required"`
	Status      int    `json:"status"`
}

// Подтверждение отдельного товара.
func (it *Item) Validate() error {
	return validate.Struct(it)
}
