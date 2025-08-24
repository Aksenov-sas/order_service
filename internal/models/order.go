// Пакет models содержит структуры данных для работы с заказами
package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Order представляет структуру заказа
type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	ShardKey          string    `json:"shardkey"`
	SMID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OOFShard          string    `json:"oof_shard"`
}

// Delivery представляет информацию о доставке
type Delivery struct {
	OrderUID string `json:"-"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Zip      string `json:"zip"`
	City     string `json:"city"`
	Address  string `json:"address"`
	Region   string `json:"region"`
	Email    string `json:"email"`
}

// Payment представляет информацию о платеже
type Payment struct {
	OrderUID     string `json:"-"`
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDT    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

// Item представляет товар в заказе
type Item struct {
	OrderUID    string `json:"-"`
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	RID         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NMID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

// TimeToPgType преобразует time.Time в pgtype.Timestamp для работы с PostgreSQL
func TimeToPgType(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

// PgTypeToTime преобразует pgtype.Timestamp в time.Time
func PgTypeToTime(ts pgtype.Timestamp) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}
