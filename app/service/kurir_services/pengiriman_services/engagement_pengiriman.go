package kurir_pengiriman_services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	payment_out_constanta "github.com/anan112pcmec/Burung-backend-1/app/api/payment_out_flip"
	payment_out_disbursment "github.com/anan112pcmec/Burung-backend-1/app/api/payment_out_flip/disbursment"
	"github.com/anan112pcmec/Burung-backend-1/app/config"
	kurir_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/entity/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/database/enums/nama_kota"
	"github.com/anan112pcmec/Burung-backend-1/app/database/enums/nama_provinsi"
	pengiriman_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/pengiriman"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func AktifkanBidKurir(ctx context.Context, data PayloadAktifkanBidKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "AktifkanBidKurir"

	kurir, status := data.IdentitasKurir.Validating(ctx, db.Read)

	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	if _, ok := nama_provinsi.JawaProvinsiMap[data.Provinsi]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Nama provinsi tidak valid",
		}
	}

	if _, ok := nama_kota.KotaJawaMap[data.Kota]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Nama kota tidak valid",
		}
	}

	if kurir.StatusBid != kurir_enums.Off {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Kamu sudah melakukan bid",
		}
	}

	if !kurir.VerifiedKurir {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal harap lengkapi data minimum kurir terlebih dahulu",
		}
	}

	var id_data_kurir_bid int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("id").Where(&models.BidKurirData{
		IdKurir: data.IdentitasKurir.IdKurir,
		Selesai: nil,
	}).Limit(1).Scan(&id_data_kurir_bid).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_kurir_bid != 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal bid sebelumnya belum selesai",
		}
	}

	var SlotTersisa int8 = 0

	switch data.JenisPengiriman {
	case pengiriman_enums.Instant:
		SlotTersisa = 1
	case pengiriman_enums.Fast:
		SlotTersisa = 5
	case pengiriman_enums.Reguler:
		SlotTersisa = 8
	default:
		SlotTersisa = 0
	}

	if data.Mode == "manual" && data.JenisPengiriman != pengiriman_enums.Reguler {
		data.Mode = "auto"
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.BidKurirData{
			IdKurir:         data.IdentitasKurir.IdKurir,
			JenisPengiriman: data.JenisPengiriman,
			Mode:            data.Mode,
			Provinsi:        data.Provinsi,
			Kota:            data.Kota,
			Alamat:          data.Alamat,
			Longitude:       data.Longitude,
			Latitude:        data.Latitude,
			MaxKg:           int16(data.MaxKg),
			JenisKendaraan:  kurir.TipeKendaraan,
			IsEkspedisi:     data.IsEkspedisi,
			SlotTersisa:     int32(SlotTersisa),
			Status:          kurir_enums.Mengumpulkan,
			Dimulai:         time.Now(),
			Selesai:         nil,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
			ID: data.IdentitasKurir.IdKurir,
		}).Update("status_bid", kurir_enums.Idle).Error; err != nil {
			return err
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

func UpdatePosisiBidKurir(ctx context.Context, data PayloadUpdatePosisiBid, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "UpdatePosisiBidKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_bid_kurir_data int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("id").Where(&models.BidKurirData{
		ID:      data.IdBidKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_bid_kurir_data).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_bid_kurir_data == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidka ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
		ID: data.IdBidKurir,
	}).Updates(&models.BidKurirData{
		Longitude: data.Longitude,
		Latitude:  data.Latitude,
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

func AmbilPengirimanNonEksManualReguler(ctx context.Context, data PayloadAmbilPengirimanNonEksManualReguler, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "AmbilPengirimanManualReguler"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	// Memastikan data bid ada
	var bid_data models.BidKurirData = models.BidKurirData{ID: 0}
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("id", "jenis_kendaraan", "slot_tersisa", "jenis_pengiriman").Where(&models.BidKurirData{
		ID:          data.IdBid,
		IdKurir:     data.IdentitasKurir.IdKurir,
		Mode:        kurir_enums.Manual,
		IsEkspedisi: false,
	}).Limit(1).Scan(&bid_data).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if bid_data.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidak ditemukan",
		}
	}

	if bid_data.SlotTersisa == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal Slot mencapai batas",
		}
	}

	var id_same_bid_scheduler int64 = 0
	if err := db.Write.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Select("id").Where(&models.BidKurirNonEksScheduler{
		IdPengiriman: data.IdPengiriman,
	}).Limit(1).Scan(&id_same_bid_scheduler).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_same_bid_scheduler != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah ambil barang itu",
		}
	}

	// Memastikan data pengiriman ada
	var jenis_kendaraan string = ""
	if err := db.Read.WithContext(ctx).Model(&models.Pengiriman{}).Select("kendaraan_required").Where(&models.Pengiriman{
		ID:      data.IdPengiriman,
		Status:  pengiriman_enums.Waiting,
		IdKurir: nil,
	}).Limit(1).Scan(&jenis_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if jenis_kendaraan == "" {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengiriman tidak ditemukan",
		}
	}

	if jenis_kendaraan != bid_data.JenisKendaraan {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal jenis kendaraan tidak sesuai",
		}
	}

	var max_slot int8 = 8

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: data.IdPengiriman,
		}).Updates(&models.Pengiriman{
			IdKurir: &data.IdentitasKurir.IdKurir,
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.BidKurirNonEksScheduler{
			IdBid:        data.IdBid,
			IdKurir:      data.IdentitasKurir.IdKurir,
			Urutan:       max_slot - int8(bid_data.SlotTersisa) + 1,
			IdPengiriman: data.IdPengiriman,
			Status:       kurir_enums.Wait,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBid,
		}).Updates(map[string]interface{}{
			"slot_tersisa": gorm.Expr("slot_tersisa - 1"),
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if bid_data.SlotTersisa == 1 {
		if err := db.Write.WithContext(ctx).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBid,
		}).Update("status", kurir_enums.SiapAntar).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusProcessing,
				Services: services,
				Message:  "Tunggu sebentar",
			}
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func AmbilPengirimanEksManualReguler(ctx context.Context, data PayloadAmbilPengirimanEksManualReguler, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "AmbilPengirimanEksManualReguler"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	// Memastikan data bid ada
	var bid_data models.BidKurirData = models.BidKurirData{ID: 0}
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("id", "jenis_kendaraan", "slot_tersisa", "jenis_pengiriman").Where(&models.BidKurirData{
		ID:          data.IdBid,
		IdKurir:     data.IdentitasKurir.IdKurir,
		Mode:        kurir_enums.Manual,
		IsEkspedisi: true,
	}).Limit(1).Scan(&bid_data).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if bid_data.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidak ditemukan",
		}
	}

	if bid_data.SlotTersisa == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal Slot mencapai batas",
		}
	}

	// Memastikan data pengiriman ada
	var jenis_kendaraan string = ""
	if err := db.Read.WithContext(ctx).Model(&models.PengirimanEkspedisi{}).Select("kendaraan_required").Where(&models.PengirimanEkspedisi{
		ID:     data.IdPengiriman,
		Status: pengiriman_enums.Waiting,
	}).Limit(1).Scan(&jenis_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if jenis_kendaraan == "" {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengiriman tidak ditemukan",
		}
	}

	if jenis_kendaraan != bid_data.JenisKendaraan {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal jenis kendaraan tidak sesuai",
		}
	}

	var id_bid_scheduler int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
		IdPengirimanEks: data.IdPengiriman,
	}).Limit(1).Scan(&id_bid_scheduler).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_bid_scheduler != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah mengambil barang itu pada bid mu",
		}
	}

	var max_slot int64 = 8

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: data.IdPengiriman,
		}).Updates(&models.PengirimanEkspedisi{
			IdKurir: &data.IdentitasKurir.IdKurir,
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.BidKurirEksScheduler{
			IdBid:           data.IdBid,
			IdKurir:         data.IdentitasKurir.IdKurir,
			Urutan:          int8(max_slot) - int8(bid_data.SlotTersisa) + 1,
			IdPengirimanEks: data.IdPengiriman,
			Status:          kurir_enums.Wait,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBid,
		}).Update("slot_tersisa", gorm.Expr("slot_tersisa - 1")).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if bid_data.SlotTersisa == 1 {
		if err := db.Write.WithContext(ctx).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBid,
		}).Update("status", kurir_enums.SiapAntar).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusProcessing,
				Services: services,
				Message:  "Tunggu sebentar",
			}
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func LockSiapAntarBidKurir(ctx context.Context, data PayloadLockSiapAntar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "LockSiapAntarBidKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var data_bid_kurir models.BidKurirData = models.BidKurirData{ID: 0}
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
		ID:      data.IdBidKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&data_bid_kurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_bid_kurir.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidak ditemukan",
		}
	}

	// Oke data bid sudah ditemukan

	var ids_data_bid_kurir_scheduler []int64

	if data_bid_kurir.IsEkspedisi {
		if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
			IdBid:   data.IdBidKurir,
			IdKurir: data.IdentitasKurir.IdKurir,
			Status:  kurir_enums.Wait,
		}).Limit(8 - int(data_bid_kurir.SlotTersisa)).Scan(&ids_data_bid_kurir_scheduler).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
	} else {
		if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Select("id").Where(&models.BidKurirNonEksScheduler{
			IdBid:   data.IdBidKurir,
			IdKurir: data.IdentitasKurir.IdKurir,
			Status:  kurir_enums.Wait,
		}).Limit(8 - int(data_bid_kurir.SlotTersisa)).Scan(&ids_data_bid_kurir_scheduler).Error; err != nil {
			return &response.ResponseForm{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
	}

	if len(ids_data_bid_kurir_scheduler) < 1 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal minimal 1 barang untuk di lock",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if data_bid_kurir.IsEkspedisi {
			if err := tx.Model(&models.BidKurirEksScheduler{}).Where("id IN ?", ids_data_bid_kurir_scheduler).Update("status", kurir_enums.Ambil).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&models.BidKurirNonEksScheduler{}).Where("id IN ?", ids_data_bid_kurir_scheduler).Update("status", kurir_enums.Ambil).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBidKurir,
		}).Update("status", kurir_enums.SiapAntar).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
			ID: data.IdentitasKurir.IdKurir,
		}).Update("status_bid", "OnDelivery").Error; err != nil {
			return err
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
		Message:  "Selamat mengantar Paket",
	}
}

func PickedUpPengirimanNonEks(ctx context.Context, data PayloadPickedUpPengirimanNonEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "PickedUpPengiriman"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var check_exist_bid_schedul int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Select("id").Where(&models.BidKurirNonEksScheduler{
		IdBid:        data.IdBidKurir,
		IdKurir:      data.IdentitasKurir.IdKurir,
		IdPengiriman: data.IdPengiriman,
		Status:       kurir_enums.Ambil,
	}).Limit(1).Scan(&check_exist_bid_schedul).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if check_exist_bid_schedul == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var IdTransaksiPengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Pengiriman{}).Select("id_transaksi").Where(&models.Pengiriman{
		ID: data.IdPengiriman,
	}).Limit(1).Scan(&IdTransaksiPengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if IdTransaksiPengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data Id Transaksi Tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			ID: check_exist_bid_schedul,
		}).Update("status", kurir_enums.Kirim).Error; err != nil {
			return err
		}

		fmt.Println("Update Status")

		if err := tx.Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: data.IdPengiriman,
		}).Update("status", pengiriman_enums.PickedUp).Error; err != nil {
			return err
		}

		fmt.Println("update pengiriman")

		if err := tx.Create(&models.JejakPengiriman{
			IdPengiriman: data.IdPengiriman,
			Lokasi:       data.Lokasi,
			Keterangan:   data.Keterangan,
			Latitude:     data.Latitude,
			Longtitude:   data.Longitude,
		}).Error; err != nil {
			fmt.Println("jeder")
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksiPengiriman,
		}).Update("status", transaksi_enums.Dikirim).Error; err != nil {
			return err
		}

		fmt.Println("Kelar")
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

func KirimPengirimanNonEks(ctx context.Context, data PayloadKirimPengirimanNonEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "KirimPengirimanNonEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var exist_bid_data_schedul int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Select("id").Where(&models.BidKurirNonEksScheduler{
		IdBid:        data.IdBidKurir,
		IdKurir:      data.IdentitasKurir.IdKurir,
		IdPengiriman: data.IdPengiriman,
		Status:       kurir_enums.Kirim,
	}).Limit(1).Scan(&exist_bid_data_schedul).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if exist_bid_data_schedul == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var id_jejak_pengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.JejakPengiriman{}).Select("id").Where(&models.JejakPengiriman{
		IdPengiriman: data.IdPengiriman,
	}).Limit(1).Scan(&id_jejak_pengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_jejak_pengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			ID: exist_bid_data_schedul,
		}).Update("status", kurir_enums.Finish).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: data.IdPengiriman,
		}).Update("status", pengiriman_enums.Diperjalanan).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.JejakPengiriman{}).Where(&models.JejakPengiriman{
			ID: id_jejak_pengiriman,
		}).Updates(&models.JejakPengiriman{
			Lokasi:     data.Lokasi,
			Keterangan: data.Keterangan,
			Latitude:   data.Latitude,
			Longtitude: data.Longitude,
		}).Error; err != nil {
			return err
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

func UpdateInformasiPerjalananPengirimanNonEks(ctx context.Context, data PayloadUpdateInformasiPerjalananPengiriman, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "UpdateInformasiPengirimanNonEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	var id_jejak_pengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.JejakPengiriman{}).Select("id").Where(&models.JejakPengiriman{
		IdPengiriman: data.IdPengiriman,
	}).Limit(1).Scan(&id_jejak_pengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_jejak_pengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengiriman tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.JejakPengiriman{}).Where(&models.JejakPengiriman{
		ID: id_jejak_pengiriman,
	}).Updates(&models.JejakPengiriman{
		Lokasi:     data.Lokasi,
		Keterangan: data.Keterangan,
		Latitude:   data.Latitude,
		Longtitude: data.Longitude,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal memperbarui informasi pengiriman",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func SampaiPengirimanNonEks(ctx context.Context, data PayloadSampaiPengirimanNonEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "SampaiPengirimanNonEks"
	var wg sync.WaitGroup
	var final bool = false

	kurirData, status := data.IdentitasKurir.Validating(ctx, db.Read)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	wg.Add(1)
	go func(idBid int64) {
		defer wg.Done()
		var ids_data_bid_kurir_scheduler []int64 = make([]int64, 0, 8)
		if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			IdBid: idBid,
		}).Limit(8).Scan(&ids_data_bid_kurir_scheduler).Error; err != nil {
			return
		}

		if len(ids_data_bid_kurir_scheduler) == 1 {
			final = true
		}
	}(data.IdBidKurir)
	var exist_bid_data_schedul int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Select("id").Where(&models.BidKurirNonEksScheduler{
		IdBid:        data.IdBidKurir,
		IdKurir:      data.IdentitasKurir.IdKurir,
		IdPengiriman: data.IdPengiriman,
		Status:       kurir_enums.Finish,
	}).Limit(1).Scan(&exist_bid_data_schedul).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if exist_bid_data_schedul == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var id_jejak_pengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.JejakPengiriman{}).Select("id").Where(&models.JejakPengiriman{
		IdPengiriman: data.IdPengiriman,
	}).Limit(1).Scan(&id_jejak_pengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_jejak_pengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal",
		}
	}

	var IdTransaksiPengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Pengiriman{}).Select("id_transaksi").Where(&models.Pengiriman{
		ID: data.IdPengiriman,
	}).Limit(1).Scan(&IdTransaksiPengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if IdTransaksiPengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data Id Transaksi Tidak ditemukan",
		}
	}

	wg.Wait()

	var id_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Pengiriman{}).Select("id_transaksi").Where(&models.Pengiriman{
		ID: data.IdPengiriman,
	}).Limit(1).Take(&id_transaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data pengiriman tidak ditemukan",
		}
	}

	var dataTransaksi models.Transaksi
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Where(&models.Transaksi{
		ID: id_transaksi,
	}).Limit(1).Take(&dataTransaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data transaksi tidak ditemukan",
		}
	}

	var (
		dataRekeningSeller models.RekeningSeller
		dataRekeningKurir  models.RekeningKurir
		NamaKotaSeller     string
		NamaKotaKurir      string
		EmailSeller        string
	)

	var id_rekening_barang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id_rekening").Where(&models.KategoriBarang{
		ID: dataTransaksi.IdKategoriBarang,
	}).Limit(1).Take(&id_rekening_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kategori barang tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.RekeningSeller{}).Where(&models.RekeningSeller{
		ID: id_rekening_barang,
	}).Limit(1).Take(&dataRekeningSeller).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data rekening seller tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.RekeningKurir{}).Where(&models.RekeningKurir{
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Take(&dataRekeningKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal menemukan rekening kurir",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("kota").Where(&models.BidKurirData{
		ID: data.IdBidKurir,
	}).Limit(1).Take(&NamaKotaKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data kota kurir tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Select("kota").Where(&models.AlamatGudang{
		ID: dataTransaksi.IdAlamatGudang,
	}).Limit(1).Take(&NamaKotaSeller).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data alamat seller tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Select("email").Where(&models.Seller{
		ID: dataTransaksi.IdSeller,
	}).Limit(1).Take(&EmailSeller).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal email seller tidak ditemukan",
		}
	}

	dataDisbursmentSeller, SellerSuccess := payment_out_disbursment.ReqCreateDisbursment(payment_out_disbursment.PayloadCreateDisbursment{
		AccountNumber:    dataRekeningSeller.NomorRekening,
		BankCode:         dataRekeningKurir.NamaBank,
		Amount:           strconv.Itoa(int(dataTransaksi.SellerPaid)),
		Remark:           "Bayaran",
		ReciepentCity:    payment_out_constanta.CityFlipJawaCode[NamaKotaSeller],
		BeneficiaryEmail: EmailSeller,
	})

	if !SellerSuccess {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang gangguan mohon bersabar dan coba ulang",
		}
	}

	dataDisbursmentKurir, KurirSucess := payment_out_disbursment.ReqCreateDisbursment(payment_out_disbursment.PayloadCreateDisbursment{
		AccountNumber:    dataRekeningKurir.NomorRekening,
		BankCode:         dataRekeningKurir.NamaBank,
		Amount:           strconv.Itoa(int(dataTransaksi.KurirPaid)),
		Remark:           "Pengiriman",
		ReciepentCity:    payment_out_constanta.CityFlipJawaCode[NamaKotaKurir],
		BeneficiaryEmail: kurirData.Email,
	})

	if !KurirSucess {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang gangguan mohon bersabar dan coba ulang",
		}
	}

	if !dataDisbursmentSeller.Validating() {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang gangguan mohon bersabar dan coba ulang",
		}
	}

	if !dataDisbursmentKurir.Validating() {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang gangguan mohon bersabar dan coba ulang",
		}
	}

	DisbursmentSeller := dataDisbursmentSeller.ReturnDisburstment()
	saveDisbursmentSeller := models.PayOutSeller{
		IdSeller:         int64(dataTransaksi.IdSeller),
		IdDisbursment:    DisbursmentSeller.ID,
		UserId:           int(DisbursmentSeller.UserID),
		Amount:           int(DisbursmentSeller.Amount),
		Status:           DisbursmentSeller.Status,
		Reason:           DisbursmentSeller.Reason,
		Timestamp:        DisbursmentSeller.Timestamp,
		BankCode:         DisbursmentSeller.BankCode,
		AccountNumber:    DisbursmentSeller.AccountNumber,
		RecipientName:    DisbursmentSeller.RecipientName,
		SenderBank:       DisbursmentSeller.SenderBank,
		Remark:           DisbursmentSeller.Remark,
		Receipt:          DisbursmentSeller.Receipt,
		TimeServed:       DisbursmentSeller.TimeServed,
		BundleId:         DisbursmentSeller.BundleID,
		CompanyId:        DisbursmentSeller.CompanyID,
		RecipientCity:    DisbursmentSeller.RecipientCity,
		CreatedFrom:      DisbursmentSeller.CreatedFrom,
		Direction:        DisbursmentSeller.Direction,
		Sender:           DisbursmentSeller.Sender,
		Fee:              DisbursmentSeller.Fee,
		BeneficiaryEmail: DisbursmentSeller.BeneficiaryEmail,
		IdempotencyKey:   DisbursmentSeller.IdempotencyKey,
		IsVirtualAccount: DisbursmentSeller.IsVirtualAccount,
	}

	DisbursmentKurir := dataDisbursmentKurir.ReturnDisburstment()
	saveDisbursmentKurir := models.PayOutKurir{
		IdKurir:          data.IdentitasKurir.IdKurir, // Pastikan field ini ada
		IdDisbursment:    DisbursmentKurir.ID,
		UserId:           int(DisbursmentKurir.UserID),
		Amount:           int(DisbursmentKurir.Amount),
		Status:           DisbursmentKurir.Status,
		Reason:           DisbursmentKurir.Reason,
		Timestamp:        DisbursmentKurir.Timestamp,
		BankCode:         DisbursmentKurir.BankCode,
		AccountNumber:    DisbursmentKurir.AccountNumber,
		RecipientName:    DisbursmentKurir.RecipientName,
		SenderBank:       DisbursmentKurir.SenderBank,
		Remark:           DisbursmentKurir.Remark,
		Receipt:          DisbursmentKurir.Receipt,
		TimeServed:       DisbursmentKurir.TimeServed,
		BundleId:         DisbursmentKurir.BundleID,
		CompanyId:        DisbursmentKurir.CompanyID,
		RecipientCity:    DisbursmentKurir.RecipientCity,
		CreatedFrom:      DisbursmentKurir.CreatedFrom,
		Direction:        DisbursmentKurir.Direction,
		Sender:           DisbursmentKurir.Sender,
		Fee:              DisbursmentKurir.Fee,
		BeneficiaryEmail: DisbursmentKurir.BeneficiaryEmail,
		IdempotencyKey:   DisbursmentKurir.IdempotencyKey,
		IsVirtualAccount: DisbursmentKurir.IsVirtualAccount,
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			IdBid:        data.IdBidKurir,
			IdPengiriman: data.IdPengiriman,
			Status:       kurir_enums.Finish,
		}).Delete(&models.BidKurirNonEksScheduler{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: data.IdPengiriman,
		}).Update("status", pengiriman_enums.Sampai).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBidKurir,
		}).Update("slot_tersisa", gorm.Expr("slot_tersisa + ?", 1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.JejakPengiriman{}).Where(&models.JejakPengiriman{
			ID: id_jejak_pengiriman,
		}).Updates(&models.JejakPengiriman{
			Lokasi:     data.Lokasi,
			Keterangan: data.Keterangan,
			Latitude:   data.Latitude,
			Longtitude: data.Longitude,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksiPengiriman,
		}).Update("status", transaksi_enums.Selesai).Error; err != nil {
			return err
		}

		if err := tx.Create(&saveDisbursmentSeller).Error; err != nil {
			return err
		}

		if err := tx.Create(&saveDisbursmentKurir).Error; err != nil {
			return err
		}

		if final {
			if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
				ID: data.IdBidKurir,
			}).Update("status", kurir_enums.Mengumpulkan).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
				ID: data.IdentitasKurir.IdKurir,
			}).Update("status_bid", kurir_enums.Idle).Error; err != nil {
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

func PickedUpPengirimanEks(ctx context.Context, data PayloadPickedUpPengirimanEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "PickedUpPengirimanEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_data_bid_schedul int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
		IdBid:           data.IdBidKurir,
		IdKurir:         data.IdentitasKurir.IdKurir,
		IdPengirimanEks: data.IdPengirimanEks,
		Status:          kurir_enums.Ambil,
	}).Limit(1).Scan(&id_data_bid_schedul).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_bid_schedul == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidak ditemukan",
		}
	}

	var IdTransaksiPengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.PengirimanEkspedisi{}).Select("id_transaksi").Where(&models.PengirimanEkspedisi{
		ID: data.IdPengirimanEks,
	}).Limit(1).Scan(&IdTransaksiPengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if IdTransaksiPengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data Id Transaksi Tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirEksScheduler{}).Where(&models.BidKurirEksScheduler{
			ID: id_data_bid_schedul,
		}).Update("status", kurir_enums.Kirim).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: data.IdPengirimanEks,
		}).Update("status", pengiriman_enums.PickedUp).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.JejakPengirimanEkspedisi{
			IdPengirimanEkspedisi: data.IdPengirimanEks,
			Lokasi:                data.Lokasi,
			Keterangan:            data.Keterangan,
			Latitude:              data.Latitude,
			Longitude:             data.Longitude,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksiPengiriman,
		}).Update("status", transaksi_enums.Dikirim).Error; err != nil {
			return err
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

func KirimPengirimanEks(ctx context.Context, data PayloadKirimPengirimanEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "KirimPengirimanEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_data_schedul int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
		IdBid:           data.IdBidKurir,
		IdKurir:         data.IdentitasKurir.IdKurir,
		IdPengirimanEks: data.IdPengirimanEks,
		Status:          kurir_enums.Kirim,
	}).Limit(1).Scan(&id_data_schedul).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_schedul == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidak ditemukan",
		}
	}

	var id_jejak_pengiriman_eks int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.JejakPengirimanEkspedisi{}).Select("id").Where(&models.JejakPengirimanEkspedisi{
		IdPengirimanEkspedisi: data.IdPengirimanEks,
	}).Limit(1).Scan(&id_jejak_pengiriman_eks).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_jejak_pengiriman_eks == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnavailableForLegalReasons,
			Services: services,
			Message:  "Gagal",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirEksScheduler{}).Where(&models.BidKurirEksScheduler{
			ID: id_data_schedul,
		}).Update("status", kurir_enums.Finish).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: data.IdPengirimanEks,
		}).Update("status", pengiriman_enums.DikirimEkspedisi).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.JejakPengirimanEkspedisi{}).Where(&models.JejakPengirimanEkspedisi{
			ID: id_jejak_pengiriman_eks,
		}).Updates(&models.JejakPengirimanEkspedisi{
			IdPengirimanEkspedisi: data.IdPengirimanEks,
			Lokasi:                data.Lokasi,
			Keterangan:            data.Keterangan,
			Latitude:              data.Latitude,
			Longitude:             data.Longitude,
		}).Error; err != nil {
			return err
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

func UpdateInformasiPerjalananPengirimanEks(ctx context.Context, data PayloadUpdateInformasiPerjalananPengirimanEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "UpdateInformasiPerjalananPengirimanEks"

	if _, status := data.IdenititasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_data_jejak_pengiriman_eks int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.JejakPengirimanEkspedisi{}).Select("id").Where(&models.JejakPengirimanEkspedisi{
		IdPengirimanEkspedisi: data.IdPengirimanEks,
	}).Limit(1).Scan(&id_data_jejak_pengiriman_eks).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_jejak_pengiriman_eks == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data jejak pengiriman tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.JejakPengirimanEkspedisi{}).Where(&models.JejakPengirimanEkspedisi{
		ID: id_data_jejak_pengiriman_eks,
	}).Updates(&models.JejakPengirimanEkspedisi{
		Lokasi:     data.Lokasi,
		Keterangan: data.Keterangan,
		Latitude:   data.Latitude,
		Longitude:  data.Longitude,
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

func SampaiPengirimanEks(ctx context.Context, data PayloadSampaiPengirimanEks, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	const services = "SampaiPengirimanEks"
	var wg sync.WaitGroup
	var final bool = false

	DataKurir, status := data.IdentitasKurir.Validating(ctx, db.Read)

	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	wg.Add(1)
	go func(idBid int64) {
		defer wg.Done()

		var ids_data_bid_kurir_scheduler []int64
		if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
			IdBid: idBid,
		}).Limit(8).Scan(&ids_data_bid_kurir_scheduler).Error; err != nil {
			return
		}

		if len(ids_data_bid_kurir_scheduler) == 1 {
			final = true
		}
	}(data.IdBidKurir)

	var id_bid_schedul int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
		IdBid:           data.IdBidKurir,
		IdPengirimanEks: data.IdPengirimanEks,
		IdKurir:         data.IdBidKurir,
		Status:          kurir_enums.Finish,
	}).Limit(1).Scan(&id_bid_schedul).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_bid_schedul == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data bid tidak ditemukan",
		}
	}

	var id_jejak_pengiriman_eks int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.JejakPengirimanEkspedisi{}).Select("id").Where(&models.JejakPengirimanEkspedisi{
		IdPengirimanEkspedisi: data.IdPengirimanEks,
	}).Limit(1).Scan(&id_jejak_pengiriman_eks).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_jejak_pengiriman_eks == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnavailableForLegalReasons,
			Services: services,
			Message:  "Gagal",
		}
	}

	var IdTransaksiPengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.PengirimanEkspedisi{}).Select("id_transaksi").Where(&models.PengirimanEkspedisi{
		ID: data.IdPengirimanEks,
	}).Limit(1).Scan(&IdTransaksiPengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if IdTransaksiPengiriman == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data Id Transaksi Tidak ditemukan",
		}
	}

	wg.Wait()

	var (
		DataTransaksi  models.Transaksi
		DataPengiriman models.PengirimanEkspedisi
		RekeningKurir  models.RekeningKurir
		NamaKotaKurir  string
	)

	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Where(&models.Transaksi{
		ID: IdTransaksiPengiriman,
	}).Limit(1).Scan(&DataTransaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data transaksi tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
		ID: data.IdPengirimanEks,
	}).Limit(1).Take(&DataPengiriman).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengiriman tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.RekeningKurir{}).Where(&models.RekeningKurir{
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Take(&RekeningKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data rekening kurir tidak ditemukan",
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("kota").Where(&models.BidKurirData{
		ID: data.IdBidKurir,
	}).Limit(1).Take(&NamaKotaKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kota kurir tidak didapatkan",
		}
	}

	dataDisbursmentKurir, StatusDisbursmentKurir := payment_out_disbursment.ReqCreateDisbursment(payment_out_disbursment.PayloadCreateDisbursment{
		AccountNumber:    RekeningKurir.NomorRekening,
		BankCode:         RekeningKurir.NamaBank,
		Amount:           strconv.Itoa(int(DataPengiriman.KurirPaid)),
		Remark:           "Pengiriman",
		ReciepentCity:    payment_out_constanta.CityFlipJawaCode[NamaKotaKurir],
		BeneficiaryEmail: DataKurir.Email,
	})

	if !StatusDisbursmentKurir {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server mengalami gangguan mohon bersabar dan coba lagi",
		}
	}

	if !dataDisbursmentKurir.Validating() {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang mengalami gangguan coba lagi lain waktu",
		}
	}

	DisbursmentKurir := dataDisbursmentKurir.ReturnDisburstment()
	saveDisbursmentKurir := models.PayOutKurir{
		IdKurir:          data.IdentitasKurir.IdKurir, // Pastikan field ini ada
		IdDisbursment:    DisbursmentKurir.ID,
		UserId:           int(DisbursmentKurir.UserID),
		Amount:           int(DisbursmentKurir.Amount),
		Status:           DisbursmentKurir.Status,
		Reason:           DisbursmentKurir.Reason,
		Timestamp:        DisbursmentKurir.Timestamp,
		BankCode:         DisbursmentKurir.BankCode,
		AccountNumber:    DisbursmentKurir.AccountNumber,
		RecipientName:    DisbursmentKurir.RecipientName,
		SenderBank:       DisbursmentKurir.SenderBank,
		Remark:           DisbursmentKurir.Remark,
		Receipt:          DisbursmentKurir.Receipt,
		TimeServed:       DisbursmentKurir.TimeServed,
		BundleId:         DisbursmentKurir.BundleID,
		CompanyId:        DisbursmentKurir.CompanyID,
		RecipientCity:    DisbursmentKurir.RecipientCity,
		CreatedFrom:      DisbursmentKurir.CreatedFrom,
		Direction:        DisbursmentKurir.Direction,
		Sender:           DisbursmentKurir.Sender,
		Fee:              DisbursmentKurir.Fee,
		BeneficiaryEmail: DisbursmentKurir.BeneficiaryEmail,
		IdempotencyKey:   DisbursmentKurir.IdempotencyKey,
		IsVirtualAccount: DisbursmentKurir.IsVirtualAccount,
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := tx.Model(&models.BidKurirEksScheduler{}).Where(&models.BidKurirEksScheduler{
			IdBid:           data.IdBidKurir,
			IdPengirimanEks: data.IdPengirimanEks,
			Status:          kurir_enums.Finish,
		}).Delete(&models.BidKurirNonEksScheduler{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: data.IdPengirimanEks,
		}).Update("status", pengiriman_enums.SampaiAgentEkspedisi).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBidKurir,
		}).Update("slot_tersisa", gorm.Expr("slot_tersisa + ?", 1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.JejakPengirimanEkspedisi{}).Where(&models.JejakPengirimanEkspedisi{
			ID: id_jejak_pengiriman_eks,
		}).Updates(&models.JejakPengirimanEkspedisi{
			Lokasi:     data.Lokasi,
			Keterangan: data.Keterangan,
			Latitude:   data.Latitude,
			Longitude:  data.Longitude,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksiPengiriman,
		}).Updates(&models.Transaksi{
			KodeResiEkspedisi: &data.NoResiEkspedisi,
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&saveDisbursmentKurir).Error; err != nil {
			return err
		}

		if final {
			if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
				ID: data.IdBidKurir,
			}).Update("status", kurir_enums.Mengumpulkan).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
				ID: data.IdentitasKurir.IdKurir,
			}).Update("status_bid", kurir_enums.Idle).Error; err != nil {
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

func NonaktifkanBidKurir(ctx context.Context, data PayloadNonaktifkanBidKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "NonaktifkanBidKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var data_bid models.BidKurirData = models.BidKurirData{ID: 0}
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Select("id", "is_ekspedisi").Where(&models.BidKurirData{
		ID:      data.IdBidKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
		Status:  kurir_enums.Idle,
	}).Limit(1).Scan(&data_bid).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_bid.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal menemukan data bid",
		}
	}

	// Mengecek bid kurir scheduler
	if data_bid.IsEkspedisi {
		var id_data_bid_schedul_eks int64 = 0
		if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Select("id").Where(&models.BidKurirEksScheduler{
			IdBid:   data.IdBidKurir,
			IdKurir: data.IdentitasKurir.IdKurir,
		}).Limit(1).Scan(&id_data_bid_schedul_eks).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

		if id_data_bid_schedul_eks != 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal lanjutkan terlebih dahulu pengiriman di bid sampai selesai",
			}
		}
	} else {
		var id_data_bid_schedul_non_eks int64 = 0
		if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Select("id").Where(&models.BidKurirNonEksScheduler{
			IdBid:   data.IdBidKurir,
			IdKurir: data.IdentitasKurir.IdKurir,
		}).Limit(1).Scan(&id_data_bid_schedul_non_eks).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

		if id_data_bid_schedul_non_eks == 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal selesaikan dulu pengiriman di bid sampai selesai",
			}
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBidKurir,
		}).Delete(&models.BidKurirData{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
			ID: data.IdentitasKurir.IdKurir,
		}).Update("status_bid", kurir_enums.Off).Error; err != nil {
			return err
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
