// Пакет models содержит структуры данных для работы с заказами
package models

import (
	"errors"
	"strings"
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

// Validate выполняет строгую проверку заказа, полученного от брокера.
func (o *Order) Validate() error {
	if o == nil {
		return errors.New("order is nil")
	}
	if strings.TrimSpace(o.OrderUID) == "" {
		return errors.New("order_uid is required")
	}
	if strings.TrimSpace(o.TrackNumber) == "" {
		return errors.New("track_number is required")
	}
	if strings.TrimSpace(o.Entry) == "" {
		return errors.New("entry is required")
	}
	if strings.TrimSpace(o.Locale) == "" {
		return errors.New("locale is required")
	}
	if strings.TrimSpace(o.CustomerID) == "" {
		return errors.New("customer_id is required")
	}
	if strings.TrimSpace(o.DeliveryService) == "" {
		return errors.New("delivery_service is required")
	}
	if strings.TrimSpace(o.ShardKey) == "" {
		return errors.New("shardkey is required")
	}
	if o.SMID == 0 {
		return errors.New("sm_id must be non-zero")
	}
	if strings.TrimSpace(o.OOFShard) == "" {
		return errors.New("oof_shard is required")
	}
	if err := o.Delivery.Validate(); err != nil {
		return err
	}
	if err := o.Payment.Validate(); err != nil {
		return err
	}
	if len(o.Items) == 0 {
		return errors.New("items must contain at least one item")
	}
	for i := range o.Items {
		if err := o.Items[i].Validate(); err != nil {
			return err
		}
	}
	return nil
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

// Подтверждение деталей доставки.
func (d *Delivery) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("delivery.name is required")
	}
	if strings.TrimSpace(d.Phone) == "" {
		return errors.New("delivery.phone is required")
	}
	if strings.TrimSpace(d.Zip) == "" {
		return errors.New("delivery.zip is required")
	}
	if strings.TrimSpace(d.City) == "" {
		return errors.New("delivery.city is required")
	}
	if strings.TrimSpace(d.Address) == "" {
		return errors.New("delivery.address is required")
	}
	if strings.TrimSpace(d.Region) == "" {
		return errors.New("delivery.region is required")
	}
	if strings.TrimSpace(d.Email) == "" {
		return errors.New("delivery.email is required")
	}
	return nil
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

// Подтверждение платежа.
func (p *Payment) Validate() error {
	if strings.TrimSpace(p.Transaction) == "" {
		return errors.New("payment.transaction is required")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return errors.New("payment.currency is required")
	}
	if strings.TrimSpace(p.Provider) == "" {
		return errors.New("payment.provider is required")
	}
	if p.Amount < 0 {
		return errors.New("payment.amount must be >= 0")
	}
	if p.PaymentDT <= 0 {
		return errors.New("payment.payment_dt must be > 0 (unix seconds)")
	}
	if strings.TrimSpace(p.Bank) == "" {
		return errors.New("payment.bank is required")
	}
	if p.DeliveryCost < 0 || p.GoodsTotal < 0 || p.CustomFee < 0 {
		return errors.New("payment cost fields must be >= 0")
	}
	return nil
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

// Подтверждение отдельного товара.
func (it *Item) Validate() error {
	if it.ChrtID == 0 {
		return errors.New("item.chrt_id must be non-zero")
	}
	if strings.TrimSpace(it.TrackNumber) == "" {
		return errors.New("item.track_number is required")
	}
	if it.Price < 0 || it.TotalPrice < 0 {
		return errors.New("item price fields must be >= 0")
	}
	if strings.TrimSpace(it.RID) == "" {
		return errors.New("item.rid is required")
	}
	if strings.TrimSpace(it.Name) == "" {
		return errors.New("item.name is required")
	}
	if strings.TrimSpace(it.Size) == "" {
		return errors.New("item.size is required")
	}
	if it.NMID == 0 {
		return errors.New("item.nm_id must be non-zero")
	}
	if strings.TrimSpace(it.Brand) == "" {
		return errors.New("item.brand is required")
	}
	return nil
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
