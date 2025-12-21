package kurir_pengiriman_services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	payment_out_constanta "github.com/anan112pcmec/Burung-backend-1/app/api/payment_out_flip"
	payment_out_disbursment "github.com/anan112pcmec/Burung-backend-1/app/api/payment_out_flip/disbursment"
	"github.com/anan112pcmec/Burung-backend-1/app/config"
	kurir_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_kota"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_provinsi"
	pengiriman_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/pengiriman"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_bid_kurir_data "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/bid_kurir_data"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	stsk_pengiriman "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengiriman"
	stsk_pengiriman_ekspedisi "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengiriman_ekspedisi"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func AktifkanBidKurir(ctx context.Context, data PayloadAktifkanBidKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "AktifkanBidKurir"

	kurir, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)

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

	newBidKurirData := models.BidKurirData{
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
	}
	var idKurirUpdated int64

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := tx.Create(&newBidKurirData).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
			ID: data.IdentitasKurir.IdKurir,
		}).Update("status_bid", kurir_enums.Idle).Error; err != nil {
			return err
		}

		idKurirUpdated = data.IdentitasKurir.IdKurir
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bkd models.BidKurirData, IdKurir int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Bkd.IdKurir,
		}

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.BidKurirData); err != nil {
			fmt.Println("Gagal incr count bid kurir data ke threshold kurir")
		}

		bidKurirDataCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bkd).SetTableName(Bkd.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataCreatePublish); err != nil {
			fmt.Println("Gagal publish bid kurir data create ke message broker")
		}

		var dataKurirUpdated models.Kurir
		if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
			ID: IdKurir,
		}).Limit(1).Take(&dataKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data kurir")
			return
		}

		kurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKurirUpdated).SetTableName(dataKurirUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update kurir ke message broker")
		}
	}(newBidKurirData, idKurirUpdated, db.Write, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UpdatePosisiBidKurir(ctx context.Context, data PayloadUpdatePosisiBid, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "UpdatePosisiBidKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
			Message:  "Gagal data bid tidak ditemukan",
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

	go func(IdBkd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataBidKurirDataUpdated models.BidKurirData
		if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: IdBkd,
		}).Limit(1).Take(&dataBidKurirDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir data")
			return
		}

		bidKurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataUpdated).SetTableName(dataBidKurirDataUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update bid kurir data ke message broker")
		}
	}(data.IdBidKurir, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func AmbilPengirimanNonEksManualReguler(ctx context.Context, data PayloadAmbilPengirimanNonEksManualReguler, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "AmbilPengirimanManualReguler"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
	var newBidKurirNonEksScheduler models.BidKurirNonEksScheduler
	var idPengirimanUpdated int64
	var idBidKurirDataUpdated int64

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: data.IdPengiriman,
		}).Updates(&models.Pengiriman{
			IdKurir: &data.IdentitasKurir.IdKurir,
		}).Error; err != nil {
			return err
		}

		newBidKurirNonEksScheduler = models.BidKurirNonEksScheduler{
			IdBid:        data.IdBid,
			IdKurir:      data.IdentitasKurir.IdKurir,
			Urutan:       max_slot - int8(bid_data.SlotTersisa) + 1,
			IdPengiriman: data.IdPengiriman,
			Status:       kurir_enums.Wait,
		}

		if err := tx.Create(&newBidKurirNonEksScheduler).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBid,
		}).Updates(map[string]interface{}{
			"slot_tersisa": gorm.Expr("slot_tersisa - 1"),
		}).Error; err != nil {
			return err
		}

		idPengirimanUpdated = data.IdPengiriman
		idBidKurirDataUpdated = data.IdBid
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bknes models.BidKurirNonEksScheduler, IdPengiriman int64, IdBkd int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Bknes.IdKurir,
		}

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.BidKurirNonEksScheduler); err != nil {
			fmt.Println("Gagal incr count bid kurir non eks scheduler ke threshold kurir")
		}

		bidKurirNonEksSchedulerCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bknes).SetTableName(Bknes.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirNonEksSchedulerCreatePublish); err != nil {
			fmt.Println("Gagal publish bid kurir non eks scheduler create ke message broker")
		}

		var dataPengirimanUpdated models.Pengiriman
		if err := Read.WithContext(konteks).Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: IdPengiriman,
		}).Limit(1).Take(&dataPengirimanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman")
		} else {
			pengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanUpdated).SetTableName(dataPengirimanUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ke message broker")
			}
		}

		var dataBidKurirDataUpdated models.BidKurirData
		if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: IdBkd,
		}).Limit(1).Take(&dataBidKurirDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir data")
		} else {
			bidKurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataUpdated).SetTableName(dataBidKurirDataUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir data ke message broker")
			}
		}
	}(newBidKurirNonEksScheduler, idPengirimanUpdated, idBidKurirDataUpdated, db.Write, db.Read, cud_publisher)

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

		go func(IdBkd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBidKurirDataStatusUpdated models.BidKurirData
			if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
				ID: IdBkd,
			}).Limit(1).Take(&dataBidKurirDataStatusUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data bid kurir data untuk update status")
				return
			}

			bidKurirDataStatusUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataStatusUpdated).SetTableName(dataBidKurirDataStatusUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataStatusUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update status bid kurir data ke message broker")
			}
		}(data.IdBid, db.Read, cud_publisher)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func AmbilPengirimanEksManualReguler(ctx context.Context, data PayloadAmbilPengirimanEksManualReguler, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "AmbilPengirimanEksManualReguler"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
	var newBidKurirEksScheduler models.BidKurirEksScheduler
	var idPengirimanEksUpdated int64
	var idBidKurirDataUpdated int64

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: data.IdPengiriman,
		}).Updates(&models.PengirimanEkspedisi{
			IdKurir: &data.IdentitasKurir.IdKurir,
		}).Error; err != nil {
			return err
		}

		newBidKurirEksScheduler = models.BidKurirEksScheduler{
			IdBid:           data.IdBid,
			IdKurir:         data.IdentitasKurir.IdKurir,
			Urutan:          int8(max_slot) - int8(bid_data.SlotTersisa) + 1,
			IdPengirimanEks: data.IdPengiriman,
			Status:          kurir_enums.Wait,
		}

		if err := tx.Create(&newBidKurirEksScheduler).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: data.IdBid,
		}).Update("slot_tersisa", gorm.Expr("slot_tersisa - 1")).Error; err != nil {
			return err
		}

		idPengirimanEksUpdated = data.IdPengiriman
		idBidKurirDataUpdated = data.IdBid
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bkes models.BidKurirEksScheduler, IdPengirimanEks int64, IdBkd int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Bkes.IdKurir,
		}

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.BidKurirEksScheduler); err != nil {
			fmt.Println("Gagal incr count bid kurir eks scheduler ke threshold kurir")
		}

		bidKurirEksSchedulerCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bkes).SetTableName(Bkes.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirEksSchedulerCreatePublish); err != nil {
			fmt.Println("Gagal publish bid kurir eks scheduler create ke message broker")
		}

		var dataPengirimanEksUpdated models.PengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: IdPengirimanEks,
		}).Limit(1).Take(&dataPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman ekspedisi")
		} else {
			pengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanEksUpdated).SetTableName(dataPengirimanEksUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanEksUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ekspedisi ke message broker")
			}
		}

		var dataBidKurirDataUpdated models.BidKurirData
		if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: IdBkd,
		}).Limit(1).Take(&dataBidKurirDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir data")
		} else {
			bidKurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataUpdated).SetTableName(dataBidKurirDataUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir data ke message broker")
			}
		}
	}(newBidKurirEksScheduler, idPengirimanEksUpdated, idBidKurirDataUpdated, db.Write, db.Read, cud_publisher)

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

		go func(IdBkd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBidKurirDataStatusUpdated models.BidKurirData
			if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
				ID: IdBkd,
			}).Limit(1).Take(&dataBidKurirDataStatusUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data bid kurir data untuk update status")
				return
			}

			bidKurirDataStatusUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataStatusUpdated).SetTableName(dataBidKurirDataStatusUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataStatusUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update status bid kurir data ke message broker")
			}
		}(data.IdBid, db.Read, cud_publisher)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func LockSiapAntarBidKurir(ctx context.Context, data PayloadLockSiapAntar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "LockSiapAntarBidKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
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

	var idBidKurirDataUpdated int64
	var idKurirUpdated int64

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

		idBidKurirDataUpdated = data.IdBidKurir
		idKurirUpdated = data.IdentitasKurir.IdKurir
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IsEks bool, IdsScheduler []int64, IdBkd int64, IdKurir int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if IsEks {
			var dataSchedulers []models.BidKurirEksScheduler
			if err := Read.WithContext(konteks).Model(&models.BidKurirEksScheduler{}).Where("id IN ?", IdsScheduler).Find(&dataSchedulers).Error; err != nil {
				fmt.Println("Gagal mengambil data bid kurir eks scheduler")
			} else {
				for _, scheduler := range dataSchedulers {
					schedulerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(scheduler).SetTableName(scheduler.TableName())
					if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, schedulerUpdatedPublish); err != nil {
						fmt.Println("Gagal publish update bid kurir eks scheduler ke message broker")
					}
				}
			}
		} else {
			var dataSchedulers []models.BidKurirNonEksScheduler
			if err := Read.WithContext(konteks).Model(&models.BidKurirNonEksScheduler{}).Where("id IN ?", IdsScheduler).Find(&dataSchedulers).Error; err != nil {
				fmt.Println("Gagal mengambil data bid kurir non eks scheduler")
			} else {
				for _, scheduler := range dataSchedulers {
					schedulerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(scheduler).SetTableName(scheduler.TableName())
					if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, schedulerUpdatedPublish); err != nil {
						fmt.Println("Gagal publish update bid kurir non eks scheduler ke message broker")
					}
				}
			}
		}

		var dataBidKurirDataUpdated models.BidKurirData
		if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: IdBkd,
		}).Limit(1).Take(&dataBidKurirDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir data")
		} else {
			bidKurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataUpdated).SetTableName(dataBidKurirDataUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir data ke message broker")
			}
		}

		var dataKurirUpdated models.Kurir
		if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
			ID: IdKurir,
		}).Limit(1).Take(&dataKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data kurir")
		} else {
			kurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKurirUpdated).SetTableName(dataKurirUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update kurir ke message broker")
			}
		}
	}(data_bid_kurir.IsEkspedisi, ids_data_bid_kurir_scheduler, idBidKurirDataUpdated, idKurirUpdated, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Selamat mengantar Paket",
	}
}

func PickedUpPengirimanNonEks(ctx context.Context, data PayloadPickedUpPengirimanNonEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "PickedUpPengiriman"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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

	var newJejakPengiriman models.JejakPengiriman
	var idSchedulerUpdated int64
	var idPengirimanUpdated int64
	var idTransaksiUpdated int64

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			ID: check_exist_bid_schedul,
		}).Update("status", kurir_enums.Kirim).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: data.IdPengiriman,
		}).Update("status", pengiriman_enums.PickedUp).Error; err != nil {
			return err
		}

		newJejakPengiriman = models.JejakPengiriman{
			IdPengiriman: data.IdPengiriman,
			Lokasi:       data.Lokasi,
			Keterangan:   data.Keterangan,
			Latitude:     data.Latitude,
			Longtitude:   data.Longitude,
		}

		if err := tx.Create(&newJejakPengiriman).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksiPengiriman,
		}).Update("status", transaksi_enums.Dikirim).Error; err != nil {
			return err
		}

		idSchedulerUpdated = check_exist_bid_schedul
		idPengirimanUpdated = data.IdPengiriman
		idTransaksiUpdated = IdTransaksiPengiriman
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Jp models.JejakPengiriman, IdScheduler int64, IdPengiriman int64, IdTransaksi int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		jejakPengirimanCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Jp).SetTableName(Jp.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanCreatePublish); err != nil {
			fmt.Println("Gagal publish jejak pengiriman create ke message broker")
		}

		var dataSchedulerUpdated models.BidKurirNonEksScheduler
		if err := Read.WithContext(konteks).Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			ID: IdScheduler,
		}).Limit(1).Take(&dataSchedulerUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir non eks scheduler")
		} else {
			schedulerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataSchedulerUpdated).SetTableName(dataSchedulerUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, schedulerUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir non eks scheduler ke message broker")
			}
		}

		var dataPengirimanUpdated models.Pengiriman
		if err := Read.WithContext(konteks).Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: IdPengiriman,
		}).Limit(1).Take(&dataPengirimanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman")
		} else {
			pengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanUpdated).SetTableName(dataPengirimanUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ke message broker")
			}
		}

		var dataTransaksiUpdated models.Transaksi
		if err := Read.WithContext(konteks).Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksi,
		}).Limit(1).Take(&dataTransaksiUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data transaksi")
		} else {
			transaksiUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataTransaksiUpdated).SetTableName(dataTransaksiUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update transaksi ke message broker")
			}
		}
	}(newJejakPengiriman, idSchedulerUpdated, idPengirimanUpdated, idTransaksiUpdated, db.Write, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func KirimPengirimanNonEks(ctx context.Context, data PayloadKirimPengirimanNonEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "KirimPengirimanNonEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdBks, IdP, IdJp int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataBidKurirSchedulerUpdated models.BidKurirNonEksScheduler
		if err := Read.WithContext(konteks).Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
			ID: IdBks,
		}).Limit(1).Take(&dataBidKurirSchedulerUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir scheduler")
		} else {
			bidKurirSchedulerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirSchedulerUpdated).SetTableName(dataBidKurirSchedulerUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirSchedulerUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir scheduler ke message broker")
			}
		}

		var dataPengirimanUpdated models.Pengiriman
		if err := Read.WithContext(konteks).Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: IdP,
		}).Limit(1).Take(&dataPengirimanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman")
		} else {
			pengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanUpdated).SetTableName(dataPengirimanUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ke message broker")
			}
		}

		var dataJejakPengirimanUpdated models.JejakPengiriman
		if err := Read.WithContext(konteks).Model(&models.JejakPengiriman{}).Where(&models.JejakPengiriman{
			ID: IdJp,
		}).Limit(1).Take(&dataJejakPengirimanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data jejak pengiriman")
		} else {
			jejakPengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataJejakPengirimanUpdated).SetTableName(dataJejakPengirimanUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update jejak pengiriman ke message broker")
			}
		}
	}(exist_bid_data_schedul, data.IdPengiriman, id_jejak_pengiriman, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UpdateInformasiPerjalananPengirimanNonEks(ctx context.Context, data PayloadUpdateInformasiPerjalananPengiriman, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "UpdateInformasiPengirimanNonEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdJp int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataJejakPengirimanUpdated models.JejakPengiriman
		if err := Read.WithContext(konteks).Model(&models.JejakPengiriman{}).Where(&models.JejakPengiriman{
			ID: IdJp,
		}).Limit(1).Take(&dataJejakPengirimanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data jejak pengiriman")
			return
		}

		jejakPengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataJejakPengirimanUpdated).SetTableName(dataJejakPengirimanUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update jejak pengiriman ke message broker")
		}
	}(id_jejak_pengiriman, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func SampaiPengirimanNonEks(ctx context.Context, data PayloadSampaiPengirimanNonEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "SampaiPengirimanNonEks"
	var wg sync.WaitGroup
	var final bool = false

	kurirData, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
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

	var deletedBidKurirNonEksScheduler models.BidKurirNonEksScheduler
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirNonEksScheduler{}).Where(&models.BidKurirNonEksScheduler{
		IdBid:        data.IdBidKurir,
		IdPengiriman: data.IdPengiriman,
		Status:       kurir_enums.Finish,
	}).Limit(1).Take(&deletedBidKurirNonEksScheduler).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal menemukan data bid kurir scheduler",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirNonEksScheduler{}).Where(&deletedBidKurirNonEksScheduler).Delete(&models.BidKurirNonEksScheduler{}).Error; err != nil {
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

	go func(Bknes models.BidKurirNonEksScheduler, Ipengiriman, IbidKurir, IjejakPengiriman, ITransaksi int64, Sds models.PayOutSeller, Sdk models.PayOutKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher, f bool) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: Sds.IdSeller,
		}

		kurirThreshold := sot_threshold.KurirThreshold{
			IdKurir: Bknes.IdKurir,
		}

		bidKurirDataThreshold := sot_threshold.BidKurirDataThreshold{
			IdBidKurirData: Bknes.IdBid,
		}

		pengirimanThreshold := sot_threshold.PengirimanNonEkspedisiThreshold{
			IdPengiriman: Bknes.IdPengiriman,
		}

		if err := kurirThreshold.Decrement(konteks, Trh, stsk_kurir.BidKurirNonEksScheduler); err != nil {
			fmt.Println("Gagal decr count bid kurir non eks scheduler ke kurir threshold")
		}

		if err := bidKurirDataThreshold.Decrement(konteks, Trh, stsk_bid_kurir_data.BidKurirNonEksScheduler); err != nil {
			fmt.Println("Gagal decr count bid kurir non eks scheduler ke bid kurir data threshold")
		}

		if err := pengirimanThreshold.Decrement(konteks, Trh, stsk_pengiriman.BidKurirNonEksScheduler); err != nil {
			fmt.Println("Gagal decr count bid kurir non eks scheduler ke pengiriman threshold")
		}

		bidKurirNonEksDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bknes).SetTableName(Bknes.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirNonEksDeletePublish); err != nil {
			fmt.Println("Gagal publish delete bid kurir non eks ke message broker")
		}

		var dataUpdatedPengiriman models.Pengiriman
		if err := Trh.WithContext(konteks).Model(&models.Pengiriman{}).Where(&models.Pengiriman{
			ID: Ipengiriman,
		}).Limit(1).Take(&dataUpdatedPengiriman).Error; err != nil {
			fmt.Println("Gagal mengambil data updated pengiriman")
		} else {
			pengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedPengiriman).SetTableName(dataUpdatedPengiriman.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanUpdatedPublish); err != nil {
				fmt.Println("Gagal publish updated pengiriman ke message broker")
			}
		}

		var dataUpdatedBidKurir models.BidKurirData
		if err := Trh.WithContext(konteks).Where(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: IbidKurir,
		}).Limit(1).Take(&dataUpdatedBidKurir).Error; err != nil {
			fmt.Println("Gagal menngambil data updated bid kurir")
		} else {
			bidKurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedBidKurir).SetTableName(dataUpdatedBidKurir.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataUpdatedPublish); err != nil {
				fmt.Println("Gagal publish updated bid kurir data ke message broker")
			}
		}

		var dataUpdatedJejakPengiriman models.JejakPengiriman
		if err := Trh.WithContext(konteks).Model(&models.JejakPengiriman{}).Where(&models.JejakPengiriman{
			ID: IjejakPengiriman,
		}).Limit(1).Take(&dataUpdatedJejakPengiriman).Error; err != nil {
			fmt.Println("Gagal mengambil updated data jejak pengiriman")
		} else {
			jejakPengirimanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedJejakPengiriman).SetTableName(dataUpdatedJejakPengiriman.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update jejak pengiriman ke message broker")
			}
		}

		var dataUpdatedTransaksi models.Transaksi
		if err := Trh.WithContext(konteks).Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: ITransaksi,
		}).Limit(1).Take(&dataUpdatedTransaksi).Error; err != nil {
			fmt.Println("Gagal mendapatkan data updated transaksi")
		} else {
			transaksiUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedTransaksi).SetTableName(dataUpdatedTransaksi.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiUpdatedPublish); err != nil {
				fmt.Println("Gagal publish updated transaksi ke message broker")
			}
		}

		if err := sellerThreshold.Increment(konteks, Trh, stsk_seller.PayoutSeller); err != nil {
			fmt.Println("Gagal incr count payout seller ke threshold seller")
		}

		payOutSellerCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Sds).SetTableName(Sds.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, payOutSellerCreatePublish); err != nil {
			fmt.Println("Gagal publish create payout seller ke message broker")
		}

		if err := kurirThreshold.Increment(konteks, Trh, stsk_kurir.PayoutKurir); err != nil {
			fmt.Println("Gagal incr count payout kurir ke threshold kurir")
		}

		payOutKurirCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Sdk).SetTableName(Sdk.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, payOutKurirCreatePublish); err != nil {
			fmt.Println("Gagal publish create payout kurir ke message broker")
		}

	}(deletedBidKurirNonEksScheduler, data.IdPengiriman, data.IdBidKurir, id_jejak_pengiriman, IdTransaksiPengiriman, saveDisbursmentSeller, saveDisbursmentKurir, db.Write, cud_publisher, final)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func PickedUpPengirimanEks(ctx context.Context, data PayloadPickedUpPengirimanEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "PickedUpPengirimanEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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

	var newJejakPengirimanEkspedisi models.JejakPengirimanEkspedisi
	var idSchedulerUpdated int64
	var idPengirimanEksUpdated int64
	var idTransaksiUpdated int64

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

		newJejakPengirimanEkspedisi = models.JejakPengirimanEkspedisi{
			IdPengirimanEkspedisi: data.IdPengirimanEks,
			Lokasi:                data.Lokasi,
			Keterangan:            data.Keterangan,
			Latitude:              data.Latitude,
			Longitude:             data.Longitude,
		}

		if err := tx.Create(&newJejakPengirimanEkspedisi).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksiPengiriman,
		}).Update("status", transaksi_enums.Dikirim).Error; err != nil {
			return err
		}

		idSchedulerUpdated = id_data_bid_schedul
		idPengirimanEksUpdated = data.IdPengirimanEks
		idTransaksiUpdated = IdTransaksiPengiriman
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Jpe models.JejakPengirimanEkspedisi, IdScheduler int64, IdPengirimanEks int64, IdTransaksi int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		jejakPengirimanEksCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Jpe).SetTableName(Jpe.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanEksCreatePublish); err != nil {
			fmt.Println("Gagal publish jejak pengiriman ekspedisi create ke message broker")
		}

		var dataSchedulerUpdated models.BidKurirEksScheduler
		if err := Read.WithContext(konteks).Model(&models.BidKurirEksScheduler{}).Where(&models.BidKurirEksScheduler{
			ID: IdScheduler,
		}).Limit(1).Take(&dataSchedulerUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir eks scheduler")
		} else {
			schedulerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataSchedulerUpdated).SetTableName(dataSchedulerUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, schedulerUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir eks scheduler ke message broker")
			}
		}

		var dataPengirimanEksUpdated models.PengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: IdPengirimanEks,
		}).Limit(1).Take(&dataPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman ekspedisi")
		} else {
			pengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanEksUpdated).SetTableName(dataPengirimanEksUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanEksUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ekspedisi ke message broker")
			}
		}

		var dataTransaksiUpdated models.Transaksi
		if err := Read.WithContext(konteks).Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksi,
		}).Limit(1).Take(&dataTransaksiUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data transaksi")
		} else {
			transaksiUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataTransaksiUpdated).SetTableName(dataTransaksiUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update transaksi ke message broker")
			}
		}
	}(newJejakPengirimanEkspedisi, idSchedulerUpdated, idPengirimanEksUpdated, idTransaksiUpdated, db.Write, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func KirimPengirimanEks(ctx context.Context, data PayloadKirimPengirimanEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "KirimPengirimanEks"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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

	var idSchedulerUpdated int64
	var idPengirimanEksUpdated int64
	var idJejakPengirimanEksUpdated int64

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

		idSchedulerUpdated = id_data_schedul
		idPengirimanEksUpdated = data.IdPengirimanEks
		idJejakPengirimanEksUpdated = id_jejak_pengiriman_eks
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdScheduler, IdPengirimanEks, IdJejakPengirimanEks int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataSchedulerUpdated models.BidKurirEksScheduler
		if err := Read.WithContext(konteks).Model(&models.BidKurirEksScheduler{}).Where(&models.BidKurirEksScheduler{
			ID: IdScheduler,
		}).Limit(1).Take(&dataSchedulerUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir eks scheduler")
		} else {
			schedulerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataSchedulerUpdated).SetTableName(dataSchedulerUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, schedulerUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir eks scheduler ke message broker")
			}
		}

		var dataPengirimanEksUpdated models.PengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: IdPengirimanEks,
		}).Limit(1).Take(&dataPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman ekspedisi")
		} else {
			pengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanEksUpdated).SetTableName(dataPengirimanEksUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanEksUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ekspedisi ke message broker")
			}
		}

		var dataJejakPengirimanEksUpdated models.JejakPengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.JejakPengirimanEkspedisi{}).Where(&models.JejakPengirimanEkspedisi{
			ID: IdJejakPengirimanEks,
		}).Limit(1).Take(&dataJejakPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data jejak pengiriman ekspedisi")
		} else {
			jejakPengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataJejakPengirimanEksUpdated).SetTableName(dataJejakPengirimanEksUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanEksUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update jejak pengiriman ekspedisi ke message broker")
			}
		}
	}(idSchedulerUpdated, idPengirimanEksUpdated, idJejakPengirimanEksUpdated, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UpdateInformasiPerjalananPengirimanEks(ctx context.Context, data PayloadUpdateInformasiPerjalananPengirimanEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "UpdateInformasiPerjalananPengirimanEks"

	if _, status := data.IdenititasKurir.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdJejakPengirimanEks int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataJejakPengirimanEksUpdated models.JejakPengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.JejakPengirimanEkspedisi{}).Where(&models.JejakPengirimanEkspedisi{
			ID: IdJejakPengirimanEks,
		}).Limit(1).Take(&dataJejakPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data jejak pengiriman ekspedisi")
			return
		}

		jejakPengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataJejakPengirimanEksUpdated).SetTableName(dataJejakPengirimanEksUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanEksUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update jejak pengiriman ekspedisi ke message broker")
		}
	}(id_data_jejak_pengiriman_eks, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func SampaiPengirimanEks(ctx context.Context, data PayloadSampaiPengirimanEks, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "SampaiPengirimanEks"
	var wg sync.WaitGroup
	var final bool = false

	DataKurir, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)

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
		IdKurir:         data.IdentitasKurir.IdKurir,
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
		IdKurir:          data.IdentitasKurir.IdKurir,
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

	var deletedBidKurirEksScheduler models.BidKurirEksScheduler
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirEksScheduler{}).Where(&models.BidKurirEksScheduler{
		IdBid:           data.IdBidKurir,
		IdPengirimanEks: data.IdPengirimanEks,
		Status:          kurir_enums.Finish,
	}).Limit(1).Take(&deletedBidKurirEksScheduler).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal menemukan data bid kurir eks scheduler",
		}
	}

	var idPengirimanEksUpdated int64
	var idBidKurirDataUpdated int64
	var idJejakPengirimanEksUpdated int64
	var idTransaksiUpdated int64
	var idKurirUpdated int64

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := tx.Model(&models.BidKurirEksScheduler{}).Where(&deletedBidKurirEksScheduler).Delete(&models.BidKurirEksScheduler{}).Error; err != nil {
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

		idPengirimanEksUpdated = data.IdPengirimanEks
		idBidKurirDataUpdated = data.IdBidKurir
		idJejakPengirimanEksUpdated = id_jejak_pengiriman_eks
		idTransaksiUpdated = IdTransaksiPengiriman
		idKurirUpdated = data.IdentitasKurir.IdKurir
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bkes models.BidKurirEksScheduler, IdPengirimanEks, IdBidKurir, IdJejakPengirimanEks, IdTransaksi, IdKurir int64, Sdk models.PayOutKurir, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher, f bool) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		kurirThreshold := sot_threshold.KurirThreshold{
			IdKurir: Bkes.IdKurir,
		}

		bidKurirDataThreshold := sot_threshold.BidKurirDataThreshold{
			IdBidKurirData: Bkes.IdBid,
		}

		pengirimanEksThreshold := sot_threshold.PengirimanEkspedisiThreshold{
			IdPengirimanEkspedisi: Bkes.IdPengirimanEks,
		}

		if err := kurirThreshold.Decrement(konteks, Trh, stsk_kurir.BidKurirEksScheduler); err != nil {
			fmt.Println("Gagal decr count bid kurir eks scheduler ke kurir threshold")
		}

		if err := bidKurirDataThreshold.Decrement(konteks, Trh, stsk_bid_kurir_data.BidKurirEksScheduler); err != nil {
			fmt.Println("Gagal decr count bid kurir eks scheduler ke bid kurir data threshold")
		}

		if err := pengirimanEksThreshold.Decrement(konteks, Trh, stsk_pengiriman_ekspedisi.BidKurirEksScheduler); err != nil {
			fmt.Println("Gagal decr count bid kurir eks scheduler ke pengiriman ekspedisi threshold")
		}

		bidKurirEksDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bkes).SetTableName(Bkes.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirEksDeletePublish); err != nil {
			fmt.Println("Gagal publish delete bid kurir eks scheduler ke message broker")
		}

		var dataPengirimanEksUpdated models.PengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.PengirimanEkspedisi{}).Where(&models.PengirimanEkspedisi{
			ID: IdPengirimanEks,
		}).Limit(1).Take(&dataPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data pengiriman ekspedisi")
		} else {
			pengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengirimanEksUpdated).SetTableName(dataPengirimanEksUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, pengirimanEksUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update pengiriman ekspedisi ke message broker")
			}
		}

		var dataBidKurirDataUpdated models.BidKurirData
		if err := Read.WithContext(konteks).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
			ID: IdBidKurir,
		}).Limit(1).Take(&dataBidKurirDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data bid kurir data")
		} else {
			bidKurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBidKurirDataUpdated).SetTableName(dataBidKurirDataUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update bid kurir data ke message broker")
			}
		}

		var dataJejakPengirimanEksUpdated models.JejakPengirimanEkspedisi
		if err := Read.WithContext(konteks).Model(&models.JejakPengirimanEkspedisi{}).Where(&models.JejakPengirimanEkspedisi{
			ID: IdJejakPengirimanEks,
		}).Limit(1).Take(&dataJejakPengirimanEksUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data jejak pengiriman ekspedisi")
		} else {
			jejakPengirimanEksUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataJejakPengirimanEksUpdated).SetTableName(dataJejakPengirimanEksUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, jejakPengirimanEksUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update jejak pengiriman ekspedisi ke message broker")
			}
		}

		var dataTransaksiUpdated models.Transaksi
		if err := Read.WithContext(konteks).Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: IdTransaksi,
		}).Limit(1).Take(&dataTransaksiUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data transaksi")
		} else {
			transaksiUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataTransaksiUpdated).SetTableName(dataTransaksiUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update transaksi ke message broker")
			}
		}

		if err := kurirThreshold.Increment(konteks, Trh, stsk_kurir.PayoutKurir); err != nil {
			fmt.Println("Gagal incr count payout kurir ke threshold kurir")
		}

		payOutKurirCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Sdk).SetTableName(Sdk.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, payOutKurirCreatePublish); err != nil {
			fmt.Println("Gagal publish create payout kurir ke message broker")
		}

		if f {
			var dataKurirUpdated models.Kurir
			if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
				ID: IdKurir,
			}).Limit(1).Take(&dataKurirUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data kurir")
			} else {
				kurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKurirUpdated).SetTableName(dataKurirUpdated.TableName())
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatedPublish); err != nil {
					fmt.Println("Gagal publish update kurir ke message broker")
				}
			}
		}
	}(deletedBidKurirEksScheduler, idPengirimanEksUpdated, idBidKurirDataUpdated, idJejakPengirimanEksUpdated, idTransaksiUpdated, idKurirUpdated, saveDisbursmentKurir, db.Write, db.Read, cud_publisher, final)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func NonaktifkanBidKurir(ctx context.Context, data PayloadNonaktifkanBidKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "NonaktifkanBidKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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

		if id_data_bid_schedul_non_eks != 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal selesaikan dulu pengiriman di bid sampai selesai",
			}
		}
	}

	var deletedBidKurirData models.BidKurirData
	if err := db.Read.WithContext(ctx).Model(&models.BidKurirData{}).Where(&models.BidKurirData{
		ID: data.IdBidKurir,
	}).Limit(1).Take(&deletedBidKurirData).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal menemukan data bid kurir data",
		}
	}

	var idKurirUpdated int64

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BidKurirData{}).Where(&deletedBidKurirData).Delete(&models.BidKurirData{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Kurir{}).Where(&models.Kurir{
			ID: data.IdentitasKurir.IdKurir,
		}).Update("status_bid", kurir_enums.Off).Error; err != nil {
			return err
		}

		idKurirUpdated = data.IdentitasKurir.IdKurir
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bkd models.BidKurirData, IdKurir int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Bkd.IdKurir,
		}

		if err := thresholdKurir.Decrement(konteks, Trh, stsk_kurir.BidKurirData); err != nil {
			fmt.Println("Gagal decr count bid kurir data ke threshold kurir")
		}

		bidKurirDataDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bkd).SetTableName(Bkd.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bidKurirDataDeletePublish); err != nil {
			fmt.Println("Gagal publish delete bid kurir data ke message broker")
		}

		var dataKurirUpdated models.Kurir
		if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
			ID: IdKurir,
		}).Limit(1).Take(&dataKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data kurir")
		} else {
			kurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKurirUpdated).SetTableName(dataKurirUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update kurir ke message broker")
			}
		}
	}(deletedBidKurirData, idKurirUpdated, db.Write, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
