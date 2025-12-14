package payment_in_gerai

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

// //////////////////////////////////////////////////////////////////////////////////////////
// Kontrak Interface Utama
// //////////////////////////////////////////////////////////////////////////////////////////

type Response interface {
	Pembayaran() (models.Pembayaran, bool)
	Pending(rds *redis.Client, id_user int64) bool
}

// //////////////////////////////////////////////////////////////////////////////////////////
// Implementasi Pembayaran
// //////////////////////////////////////////////////////////////////////////////////////////

func Bayar(r Response) (models.Pembayaran, bool) {
	return r.Pembayaran()
}

func (b *GeraiResponse) Pembayaran() (p models.Pembayaran, s bool) {
	s = true

	if b.PaymentType != "cstore" || b.OrderId == "" || b.FraudStatus == "" {
		s = false
	}

	grossFloat, err := strconv.ParseFloat(b.GrossAmount, 64)
	if err != nil {
		s = false
	}

	p = models.Pembayaran{
		KodeTransaksiPG: b.TransactionId,
		KodeOrderSistem: b.OrderId,
		Provider:        b.PaymentType,
		Total:           int32(grossFloat),
		PaymentType:     b.PaymentType,
		PaidAt:          b.TransactionTime,
	}
	return
}

// //////////////////////////////////////////////////////////////////////////////////////////
// Implementasi Pending
// //////////////////////////////////////////////////////////////////////////////////////////

const CBPENDING = 4

func CachePending(r Response, rds *redis.Client, id_user int64) bool {
	return r.Pending(rds, id_user)
}

func (b *GeraiResponse) Pending(rds *redis.Client, id_user int64) bool {
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
