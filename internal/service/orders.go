package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type OrderItemInput struct {
	ServiceID int64  `json:"service_id"`
	Quantity  string `json:"quantity"`
}
type OrderInput struct {
	CustomerID int64            `json:"customer_id"`
	DueAt      time.Time        `json:"due_at"`
	Notes      string           `json:"notes"`
	Items      []OrderItemInput `json:"items"`
	RequestKey string           `json:"request_key"`
}
type OrderItem struct {
	ID             int64  `json:"id"`
	ServiceID      int64  `json:"service_id"`
	PriceVersionID int64  `json:"price_version_id"`
	ServiceName    string `json:"service_name"`
	Unit           string `json:"unit"`
	UnitPrice      int64  `json:"unit_price"`
	QuantityMilli  int64  `json:"quantity_milli"`
	Subtotal       int64  `json:"subtotal"`
}
type Order struct {
	ID               int64       `json:"id"`
	Number           string      `json:"order_number"`
	CustomerID       int64       `json:"customer_id"`
	CustomerName     string      `json:"customer_name"`
	CustomerWhatsapp string      `json:"customer_whatsapp"`
	CustomerOptIn    bool        `json:"customer_opt_in"`
	CreatedBy        int64       `json:"created_by"`
	CreatorName      string      `json:"creator_name"`
	DueAt            time.Time   `json:"due_at"`
	Notes            string      `json:"notes"`
	Status           string      `json:"status"`
	PaymentStatus    string      `json:"payment_status"`
	Total            int64       `json:"total"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	Items            []OrderItem `json:"items"`
}
type OrderStore interface {
	InsertOrder(context.Context, int64, OrderInput, string) (Order, error)
	GetOrder(context.Context, int64) (Order, error)
}
type Orders struct{ store OrderStore }

func NewOrders(store OrderStore) *Orders { return &Orders{store} }

var quantityPattern = regexp.MustCompile(`^[0-9]{1,7}(\.[0-9]{1,3})?$`)
var requestKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,100}$`)

const MaxOrderTotal int64 = 9000000000000

func ParseQuantity(value, unit string) (int64, error) {
	if !quantityPattern.MatchString(value) {
		return 0, Invalid("items", "Berat/jumlah harus positif, maksimal tiga angka desimal.")
	}
	parts := strings.Split(value, ".")
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	milli := whole * 1000
	if len(parts) == 2 {
		fraction, _ := strconv.ParseInt(parts[1]+strings.Repeat("0", 3-len(parts[1])), 10, 64)
		milli += fraction
	}
	if milli < 1 || milli > 1000000000 || unit == "ITEM" && milli%1000 != 0 {
		return 0, Invalid("items", "Gunakan berat positif untuk kg dan jumlah bulat positif untuk pcs.")
	}
	return milli, nil
}
func ItemSubtotal(price, milli int64) (int64, error) {
	if price < 0 || price > 999999999 || milli < 1 || milli > 1000000000 {
		return 0, Invalid("items", "Harga atau jumlah di luar batas.")
	}
	result := (price*milli + 500) / 1000
	if result > MaxOrderTotal {
		return 0, Invalid("items", "Subtotal melebihi batas transaksi.")
	}
	return result, nil
}
func (s *Orders) Create(ctx context.Context, actor int64, input OrderInput) (Order, error) {
	input.Notes = strings.TrimSpace(input.Notes)
	if input.CustomerID < 1 {
		return Order{}, Invalid("customer_id", "Pilih pelanggan terdaftar.")
	}
	if input.DueAt.IsZero() || input.DueAt.Before(time.Now().Add(-time.Minute)) {
		return Order{}, Invalid("due_at", "Isi tanggal dan jam estimasi selesai yang belum lewat.")
	}
	if utf8.RuneCountInString(input.Notes) > 2000 {
		return Order{}, Invalid("notes", "Catatan maksimal 2000 karakter.")
	}
	if len(input.Items) < 1 || len(input.Items) > 30 {
		return Order{}, Invalid("items", "Pesanan wajib memiliki 1–30 layanan.")
	}
	for i, item := range input.Items {
		if item.ServiceID < 1 {
			return Order{}, Invalid("items", "Pilih layanan aktif.")
		}
		input.Items[i].Quantity = strings.TrimSpace(item.Quantity)
		if _, err := ParseQuantity(input.Items[i].Quantity, "KG"); err != nil {
			return Order{}, err
		}
	}
	if !requestKeyPattern.MatchString(input.RequestKey) {
		return Order{}, Invalid("request_key", "Kunci permintaan tidak valid.")
	}
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return s.store.InsertOrder(ctx, actor, input, hex.EncodeToString(sum[:]))
}
func (s *Orders) Get(ctx context.Context, id int64) (Order, error) { return s.store.GetOrder(ctx, id) }
