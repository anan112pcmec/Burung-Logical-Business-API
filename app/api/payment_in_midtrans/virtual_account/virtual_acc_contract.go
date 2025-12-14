package payment_in_va

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

func Bayar(r Response) (models.Pembayaran, bool) {
	return r.Pembayaran()
}

// BCA
func (b *BcaVirtualAccountResponse) Pembayaran() (p models.Pembayaran, s bool) {
	s = true

	if b.BcaVaNumber == "" || b.OrderId == "" || b.FraudStatus == "" {
		s = false
	}

	grossFloat, err := strconv.ParseFloat(b.GrossAmount, 64)
	if err != nil {
		s = false
	}
	provider := ""
	if len(b.VaNumbers) > 0 {
		provider = b.VaNumbers[0].Bank
	}

	p = models.Pembayaran{
		KodeTransaksiPG: b.TransactionId,
		KodeOrderSistem: b.OrderId,
		Provider:        provider,
		Total:           int32(grossFloat),
		PaymentType:     b.PaymentType,
		PaidAt:          b.TransactionTime,
	}
	return
}

// BNI
func (b *BniVirtualAccountResponse) Pembayaran() (p models.Pembayaran, s bool) {
	s = true

	if len(b.VaNumbers) == 0 || b.VaNumbers[0].Bank != "bni" {
		s = false
	}
	if b.OrderId == "" || b.FraudStatus == "" {
		s = false
	}

	grossFloat, err := strconv.ParseFloat(b.GrossAmount, 64)
	if err != nil {
		s = false
	}

	provider := ""
	if len(b.VaNumbers) > 0 {
		provider = b.VaNumbers[0].Bank
	}

	p = models.Pembayaran{
		KodeTransaksiPG: b.TransactionId,
		KodeOrderSistem: b.OrderId,
		Provider:        provider,
		Total:           int32(grossFloat),
		PaymentType:     b.PaymentType,
		PaidAt:          b.TransactionTime,
	}
	return
}

func (b *BriVirtualAccountResponse) Pembayaran() (p models.Pembayaran, s bool) {
	s = true

	if len(b.VaNumbers) == 0 || b.VaNumbers[0].Bank != "bri" {
		s = false
	}
	if b.OrderId == "" || b.FraudStatus == "" {
		s = false
	}

	grossFloat, err := strconv.ParseFloat(b.GrossAmount, 64)
	if err != nil {
		s = false
	}

	provider := ""
	if len(b.VaNumbers) > 0 {
		provider = b.VaNumbers[0].Bank
	}

	p = models.Pembayaran{
		KodeTransaksiPG: b.TransactionId,
		KodeOrderSistem: b.OrderId,
		Provider:        provider,
		Total:           int32(grossFloat),
		PaymentType:     b.PaymentType,
		PaidAt:          b.TransactionTime,
	}
	return
}

// PERMATA
func (b *PermataVirtualAccount) Pembayaran() (p models.Pembayaran, s bool) {
	s = true

	if b.PermataVaNumber == "" || b.OrderId == "" || b.FraudStatus == "" {
		s = false
	}

	grossFloat, err := strconv.ParseFloat(b.GrossAmount, 64)
	if err != nil {
		s = false
	}

	p = models.Pembayaran{
		KodeTransaksiPG: b.TransactionId,
		KodeOrderSistem: b.OrderId,
		Provider:        "permata",
		Total:           int32(grossFloat),
		PaymentType:     b.PaymentType,
		PaidAt:          b.TransactionTime,
	}
	return
}

// //////////////////////////////////////////////////////////////////////////////////////////
// Implementasi Pending
// //////////////////////////////////////////////////////////////////////////////////////////

func CachePending(r Response, rds *redis.Client, id_user int64) bool {
	return r.Pending(rds, id_user)
}

const CBPENDING = 4

func (b *BcaVirtualAccountResponse) Pending(rds *redis.Client, id_user int64) bool {
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

func (b *BniVirtualAccountResponse) Pending(rds *redis.Client, id_user int64) bool {
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

func (b *BriVirtualAccountResponse) Pending(rds *redis.Client, id_user int64) bool {
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

func (b *PermataVirtualAccount) Pending(rds *redis.Client, id_user int64) bool {
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

// //////////////////////////////////////////////////////////////////////////////////////////
// Implementasi StandardResponse
// //////////////////////////////////////////////////////////////////////////////////////////
