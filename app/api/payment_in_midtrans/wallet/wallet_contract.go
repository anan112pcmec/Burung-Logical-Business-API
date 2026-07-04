package payment_in_wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/redis/go-redis/v9"
)

// //////////////////////////////////////////////////////////////////////////////////////////
// Kontrak Interface Utama
// //////////////////////////////////////////////////////////////////////////////////////////

type Response interface {
	Pembayaran() (sot_models.Pembayaran, bool)
	Pending(rds *redis.Client, id_user int64) bool
}

// //////////////////////////////////////////////////////////////////////////////////////////
// Implementasi Pembayaran
// //////////////////////////////////////////////////////////////////////////////////////////

func Bayar(r Response) (sot_models.Pembayaran, bool) {
	return r.Pembayaran()
}

func (b *WalletResponse) Pembayaran() (sot_models.Pembayaran, bool) {
	m := sot_models.Pembayaran{}
	var s bool = true

	fmt.Println("[TRACE] Mulai proses Pembayaran WalletResponse")
	fmt.Printf("[TRACE] Data masuk: OrderId=%s, TransactionId=%s, PaymentType=%s, GrossAmount=%s\n",
		b.OrderId, b.TransactionId, b.PaymentType, b.GrossAmount)

	if b.OrderId == "" || b.TransactionId == "" || b.PaymentType != "qris" {
		fmt.Println("[TRACE] Data tidak valid: salah satu field kosong atau PaymentType bukan qris")
		s = false
		return m, s
	}

	grossFloat, err := strconv.ParseFloat(b.GrossAmount, 64)
	if err != nil {
		fmt.Printf("[TRACE] Error konversi GrossAmount (%s): %v\n", b.GrossAmount, err)
		s = false
		return m, s
	}

	fmt.Printf("[TRACE] GrossAmount berhasil dikonversi ke float64: %.2f\n", grossFloat)
	grossInt := int32(grossFloat)
	fmt.Printf("[TRACE] GrossAmount dikonversi ke int32: %d\n", grossInt)

	fmt.Println("[TRACE] Membuat objek sot_models.Pembayaran...")

	m = sot_models.Pembayaran{
		KodeTransaksiPG: b.TransactionId,
		KodeOrderSistem: b.OrderId,
		Provider:        b.PaymentType,
		Total:           grossInt,
		PaymentType:     "wallet",
		PaidAt:          b.TransactionTime,
	}

	fmt.Printf("[TRACE] Pembayaran selesai dibuat: %+v\n", m)
	fmt.Println("[TRACE] Selesai proses Pembayaran WalletResponse")

	return m, s
}

// //////////////////////////////////////////////////////////////////////////////////////////
// Implementasi Pending
// //////////////////////////////////////////////////////////////////////////////////////////

func CachePending(r Response, rds *redis.Client, id_user int64) bool {
	return r.Pending(rds, id_user)
}

const CBPENDING = 4

func (b *WalletResponse) Pending(rds *redis.Client, id_user int64) bool {
	key := fmt.Sprintf("tp:%v:%v", id_user, b.TransactionId)
	status := true

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CBPENDING)
	defer cancel()

	// marshal struct ke JSON
	data, err := json.Marshal(b)
	if err != nil {
		return false
	}

	// simpan ke redis
	if err := rds.Set(ctx, key, data, time.Second*CBPENDING).Err(); err != nil {
		status = false
	}

	return status
}
