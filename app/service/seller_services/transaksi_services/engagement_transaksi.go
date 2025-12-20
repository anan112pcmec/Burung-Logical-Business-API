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
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_alamat_ekspedisi "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/alamat_ekspedisi"
	stsk_alamat_gudang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/alamat_gudang"
	stsk_alamat_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/alamat_pengguna"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	stsk_transaksi "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/transaksi"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func ApproveOrderTransaksi(ctx context.Context, data PayloadApproveOrderTransaksi, db *config.InternalDBReadWriteSystem, rds, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ApproveOrderTransaksi"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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
	if err := db.Read.WithContext(ctx).Model(&sot_threshold.ThresholdOrderSeller{}).Select("id").Where(&sot_threshold.ThresholdOrderSeller{
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

		if err := db.Read.WithContext(ctx).Model(&sot_threshold.ThresholdOrderSeller{}).Select("id").Where(&sot_threshold.ThresholdOrderSeller{
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

		if err := tx.Model(&sot_threshold.ThresholdOrderSeller{}).
			Where(&sot_threshold.ThresholdOrderSeller{ID: id_threshold}).
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

	go func(It int64, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataTransaksiUpdated models.Transaksi
		if err := Trh.WithContext(konteks).Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: It,
		}).Limit(1).Take(&dataTransaksiUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data transaksi")
			return
		}

		transaksiUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataTransaksiUpdated).SetTableName(dataTransaksiUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated transaksi ke message broker")
		}

	}(data.IdTransaksi, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func KirimOrderTransaksi(ctx context.Context, data PayloadKirimOrderTransaksi, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "KirimOrderTransaksi"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	if err := db.Read.WithContext(ctx).Model(&sot_threshold.ThresholdOrderSeller{}).Select("id").Where(&sot_threshold.ThresholdOrderSeller{
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

		if err := tx.Model(&sot_threshold.ThresholdOrderSeller{}).Where(&sot_threshold.ThresholdOrderSeller{
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

			go func(Pe models.PengirimanEkspedisi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
				ctx_t := context.Background()
				konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
				defer cancel()

				thresholdSeller := sot_threshold.SellerThreshold{
					IdSeller: Pe.IdSeller,
				}

				thresholdTransaksi := sot_threshold.TransaksiThreshold{
					IdTransaksi: Pe.IdTransaksi,
				}

				thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
					IdAlamatGudang: Pe.IdAlamatGudang,
				}

				thresholdAlamatEkspedisi := sot_threshold.AlamatEkspedisiThreshold{
					IdAlamatEkspedisi: Pe.IdAlamatEkspedisi,
				}

				thresholdPengirimanEkspedisi := sot_threshold.PengirimanEkspedisiThreshold{
					IdPengirimanEkspedisi: Pe.ID,
				}

				if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.PengirimanEkspedisi); err != nil {
					fmt.Println("Gagal incr count pengiriman ekspedisi ke seller threshold")
				}

				if err := thresholdTransaksi.Increment(konteks, Trh, stsk_transaksi.PengirimanEkspedisi); err != nil {
					fmt.Println("Gagal incr count pengiriman ekspedisi ke transaksi threshold")
				}

				if err := thresholdAlamatGudang.Increment(konteks, Trh, stsk_alamat_gudang.PengirimanEkspedisi); err != nil {
					fmt.Println("Gagal incr count pengiriman ekspedisi ke alamat gudang threshold")
				}

				if err := thresholdAlamatEkspedisi.Increment(konteks, Trh, stsk_alamat_ekspedisi.PengirimanEkspedisi); err != nil {
					fmt.Println("Gagal incr count pengiriman ekspedisi ke alamat ekspedisi threshold")
				}

				if err := thresholdPengirimanEkspedisi.Inisialisasi(konteks, Trh); err != nil {
					fmt.Println("gagal membuat threshold pengiriman ekspedisi")
				}

				pengirimanEkspedisiCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Pe).SetTableName(Pe.TableName())
				if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanEkspedisiCreatePublish); err != nil {
					fmt.Println("Gagal publish create pengiriman ekspedisi ke message broker")
				}
			}(pengiriman_ekspedisi, db.Write, cud_publisher)
		} else {
			if err := tx.Create(&pengiriman_biasa).Error; err != nil {
				return err
			}

			go func(P models.Pengiriman, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
				ctx_t := context.Background()
				konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
				defer cancel()

				thresholdSeller := sot_threshold.SellerThreshold{
					IdSeller: P.IdSeller,
				}

				thresholdTransaksi := sot_threshold.TransaksiThreshold{
					IdTransaksi: P.IdTransaksi,
				}

				thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
					IdAlamatGudang: P.IdAlamatGudang,
				}

				thresholdAlamatPengguna := sot_threshold.AlamatPenggunaThreshold{
					IdAlamatPengguna: P.IdAlamatPengguna,
				}

				thresholdPengiriman := sot_threshold.PengirimanNonEkspedisiThreshold{
					IdPengiriman: P.ID,
				}

				if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Pengiriman); err != nil {
					fmt.Println("Gagal incr count pengiriman ke seller threshold")
				}

				if err := thresholdTransaksi.Increment(konteks, Trh, stsk_transaksi.Pengiriman); err != nil {
					fmt.Println("Gagal incr count pengiriman ke transaksi threshold")
				}

				if err := thresholdAlamatGudang.Increment(konteks, Trh, stsk_alamat_gudang.Pengiriman); err != nil {
					fmt.Println("Gagal incr count pengiriman ke alamat gudang threshold")
				}

				if err := thresholdAlamatPengguna.Increment(konteks, Trh, stsk_alamat_pengguna.Pengiriman); err != nil {
					fmt.Println("Gagal incr count pengiriman ke alamat pengguna threshold")
				}

				if err := thresholdPengiriman.Inisialisasi(konteks, Trh); err != nil {
					fmt.Println("Gagal membuat pengiriman threshold")
				}

				pengirimanCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(P).SetTableName(P.TableName())
				if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanCreatePublish); err != nil {
					fmt.Println("Gagal publish create pengiriman ke message broker")
				}

			}(pengiriman_biasa, db.Write, cud_publisher)
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

func UnApproveOrderTransaksi(ctx context.Context, data PayloadUnApproveOrderTransaksi, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UnApproveOrderTransaksi"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	go func(It int64, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataTransaksiUpdated models.Transaksi
		if err := Trh.WithContext(konteks).Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: It,
		}).Limit(1).Take(&dataTransaksiUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data transaksi")
			return
		}

		transaksiUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataTransaksiUpdated).SetTableName(dataTransaksiUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated transaksi ke message broker")
		}

	}(data.IdTransaksi, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
