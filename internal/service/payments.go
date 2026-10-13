package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrAlreadyPaid = errors.New("Pesanan sudah lunas. Pembayaran kedua tidak diperbolehkan.")

type PaymentInput struct {
	Method         string    `json:"method"`
	AmountReceived *int64    `json:"amount_received"`
	PaidAt         time.Time `json:"paid_at"`
	Notes          string    `json:"notes"`
	Confirmed      bool      `json:"confirmed"`
	RequestKey     string    `json:"request_key"`
}
type Payment struct {
	ID             int64     `json:"id"`
	OrderID        int64     `json:"order_id"`
	OrderNumber    string    `json:"order_number"`
	Method         string    `json:"method"`
	AmountReceived int64     `json:"amount_received"`
	AmountSettled  int64     `json:"amount_settled"`
	ChangeAmount   int64     `json:"change_amount"`
	PaidAt         time.Time `json:"paid_at"`
	CreatedBy      int64     `json:"created_by"`
	CreatorName    string    `json:"creator_name"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *Orders) Pay(ctx context.Context, actor, id int64, input PaymentInput) (Payment, error) {
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Method != "CASH" && input.Method != "TRANSFER" && input.Method != "QRIS" {
		return Payment{}, Invalid("method", "Pilih Tunai, Transfer bank, atau QRIS.")
	}
	if input.AmountReceived == nil || *input.AmountReceived < 0 || *input.AmountReceived > MaxOrderTotal {
		return Payment{}, Invalid("amount_received", "Nominal diterima wajib berupa rupiah bulat yang valid.")
	}
	if input.PaidAt.IsZero() || input.PaidAt.After(time.Now().Add(time.Minute)) {
		return Payment{}, Invalid("paid_at", "Tanggal pembayaran wajib diisi dan tidak boleh di masa depan.")
	}
	if input.Method != "CASH" && !input.Confirmed {
		return Payment{}, Invalid("confirmed", "Konfirmasi dana telah diterima.")
	}
	if utf8.RuneCountInString(input.Notes) > 1000 {
		return Payment{}, Invalid("notes", "Catatan maksimal 1000 karakter.")
	}
	if !requestKeyPattern.MatchString(input.RequestKey) {
		return Payment{}, Invalid("request_key", "Kunci permintaan tidak valid.")
	}
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return s.store.InsertPayment(ctx, actor, id, input, hex.EncodeToString(sum[:]))
}
