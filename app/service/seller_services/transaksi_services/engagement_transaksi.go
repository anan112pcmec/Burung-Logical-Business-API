package seller_transaksi_services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	pengiriman_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/pengiriman"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func ApproveOrderTransaksi(ctx context.Context, data PayloadApproveOrderTransaksi, db *config.InternalDBReadWriteSystem, rds *redis.Client) *response.ResponseForm {
	services := "ApproveOrderTransaksi"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Select("id").Where(&models.Transaksi{
		ID:       data.IdTransaksi,
		IdSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_transaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_transaksi == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data transaksi tidak ditemukan",
		}
	}

	var id_threshold int64 = 0
	if err := db.Read.WithContext(ctx).Model(&threshold.ThresholdOrderSeller{}).Select("id").Where(&threshold.ThresholdOrderSeller{
		IdSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_threshold).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal serveer sedang sibuk coba lagi lain waktu",
		}
	}

	if id_threshold == 0 {
		if tStat := data.IdentitasSeller.UpThreshold(ctx, db.Write); !tStat {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

		if err := db.Read.WithContext(ctx).Model(&threshold.ThresholdOrderSeller{}).Select("id").Where(&threshold.ThresholdOrderSeller{
			IdSeller: data.IdentitasSeller.IdSeller,
		}).Limit(1).Scan(&id_threshold).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal serveer sedang sibuk coba lagi lain waktu",
			}
		}

	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: data.IdTransaksi,
		}).Updates(&models.Transaksi{
			Status: transaksi_enums.Diproses,
		}).Error; err != nil {
			fmt.Println("gagal di updates", err)
			return err
		}

		if err := tx.Model(&threshold.ThresholdOrderSeller{}).
			Where(&threshold.ThresholdOrderSeller{ID: id_threshold}).
			Updates(map[string]interface{}{
				"total":    gorm.Expr("total + ?", 1),
				"diproses": gorm.Expr("diproses + ?", 1),
			}).Error; err != nil {
			fmt.Println("gagal di incr", err)
			return err
		}

		if data.IsAuto {
			if caching_rds := func() error {
				keyMembersAuto := "auto_pengiriman"
				keyDetailAuto := fmt.Sprintf("auto_pengiriman:%d", data.IdTransaksi)

				exists := false
				members, err := rds.SMembers(ctx, keyMembersAuto).Result()
				if err != nil {
					return err
				} else {
					for _, m := range members {
						if m == fmt.Sprintf("%d", data.IdTransaksi) {
							exists = true
							break
						}
					}
				}

				if !exists {
					if err := rds.SAdd(ctx, keyMembersAuto, data.IdTransaksi).Err(); err != nil {
						return err
					}
				}

				if err := rds.HSet(ctx, keyDetailAuto, map[string]interface{}{
					"id_transaksi": data.IdTransaksi,
					"id_seller":    data.IdentitasSeller.IdSeller,
					"waktu_commit": data.AutoPengiriman.Format(time.RFC3339),
				}).Err(); err != nil {
					return err
				}

				expiredUnix := data.AutoPengiriman.Unix()

				finalExpireUnix := expiredUnix + (5 * 60)

				if err := rds.ExpireAt(ctx, keyDetailAuto, time.Unix(finalExpireUnix, 0)).Err(); err != nil {
					return err
				}

				return nil
			}(); caching_rds != nil {
				fmt.Println("gagal di redis", caching_rds)
				return caching_rds
			}
		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func KirimOrderTransaksi(ctx context.Context, data PayloadKirimOrderTransaksi, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "KirimOrderTransaksi"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_transaksi models.Transaksi = models.Transaksi{ID: 0}
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Where(&models.Transaksi{
		ID:       data.IdTransaksi,
		IdSeller: data.IdentitasSeller.IdSeller,
		Status:   transaksi_enums.Diproses,
	}).Limit(1).Scan(&data_transaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_transaksi.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data transaksi tidak ditemukan",
		}
	}

	var pengiriman_biasa models.Pengiriman
	var pengiriman_ekspedisi models.PengirimanEkspedisi

	pengiriman_biasa = models.Pengiriman{
		IdTransaksi:       data_transaksi.ID,
		IdSeller:          int64(data.IdentitasSeller.IdSeller),
		IdAlamatGudang:    data_transaksi.IdAlamatGudang,
		IdAlamatPengguna:  data_transaksi.IdAlamatPengguna,
		IdKurir:           nil,
		BeratBarang:       data_transaksi.BeratTotalKg,
		KendaraanRequired: data_transaksi.KendaraanPengiriman,
		JenisPengiriman:   data_transaksi.JenisPengiriman,
		JarakTempuh:       data_transaksi.JarakTempuh,
		KurirPaid:         data_transaksi.KurirPaid,
		Status:            pengiriman_enums.Waiting,
	}

	pengiriman_ekspedisi = models.PengirimanEkspedisi{
		IdTransaksi:       data_transaksi.ID,
		IdSeller:          int64(data.IdentitasSeller.IdSeller),
		IdAlamatGudang:    data_transaksi.IdAlamatGudang,
		IdAlamatEkspedisi: data_transaksi.IdAlamatEkspedisi,
		IdKurir:           nil,
		BeratBarang:       data_transaksi.BeratTotalKg,
		KendaraanRequired: data_transaksi.KendaraanPengiriman,
		JenisPengiriman:   data_transaksi.JenisPengiriman,
		JarakTempuh:       data_transaksi.JarakTempuh,
		KurirPaid:         data_transaksi.KurirPaid,
		Status:            pengiriman_enums.WaitingEkspedisi,
	}

	var id_data_threshold int64 = 0

	if err := db.Read.WithContext(ctx).Model(&threshold.ThresholdOrderSeller{}).Select("id").Where(&threshold.ThresholdOrderSeller{
		IdSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_threshold).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_threshold == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: data_transaksi.ID,
		}).Update("status", transaksi_enums.Waiting).Error; err != nil {
			return err
		}

		if err := tx.Model(&threshold.ThresholdOrderSeller{}).Where(&threshold.ThresholdOrderSeller{
			ID: id_data_threshold,
		}).Updates(map[string]interface{}{
			"diproses": gorm.Expr("diproses - ?", 1),
			"waiting":  gorm.Expr("waiting + ?", 1),
		}).Error; err != nil {
			return err
		}

		if data_transaksi.IsEkspedisi {
			if err := tx.Create(&pengiriman_ekspedisi).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(&pengiriman_biasa).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UnApproveOrderTransaksi(ctx context.Context, data PayloadUnApproveOrderTransaksi, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "UnApproveOrderTransaksi"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Select("id").Where(&models.Transaksi{
		ID:       data.IdTransaksi,
		IdSeller: data.IdentitasSeller.IdSeller,
		Status:   transaksi_enums.Dibayar,
	}).Limit(1).Scan(&id_data_transaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_transaksi == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data transaksi tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Transaksi{}).Where(&models.Transaksi{
		ID: data.IdTransaksi,
	}).Updates(&models.Transaksi{
		Status:         transaksi_enums.Dibatalkan,
		DibatalkanOleh: &entity_enums.Seller,
		Catatan:        data.Catatan,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
