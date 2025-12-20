package seller_credential_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_bank"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Tambahkan Rekening Seller
// Berfungsi untuk menambahkan rekening seller ke database
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TambahRekeningSeller(ctx context.Context, data PayloadTambahkanNorekSeller, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahRekeningSeller"

	// validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	if _, ok := nama_bank.BankMap[data.NamaBank]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal, nama bank tidak diterima",
		}
	}

	// cek rekening sudah ada
	var id_rekening int64 = 0
	if err_check_rekening := db.Read.WithContext(ctx).
		Model(&models.RekeningSeller{}).
		Select("id").
		Where(models.RekeningSeller{
			IDSeller:      data.IdentitasSeller.IdSeller,
			NamaBank:      data.NamaBank,
			NomorRekening: data.NomorRekening,
		}).
		Limit(1).
		Scan(&id_rekening).Error; err_check_rekening == nil && id_rekening != 0 {
		log.Printf("[WARN] Rekening sudah ada untuk seller ID %d: %s - %s",
			data.IdentitasSeller.IdSeller, data.NamaBank, data.NomorRekening)
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	if id_rekening != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	// cek apakah seller sudah punya rekening lain (buat tentuin default)
	var id_data_rekening int64 = 0
	if err := db.Read.WithContext(ctx).
		Model(&models.RekeningSeller{}).
		Select("id").
		Where(&models.RekeningSeller{
			IDSeller: data.IdentitasSeller.IdSeller,
		}).
		Limit(1).
		Scan(&id_data_rekening).Error; err != nil {
		log.Printf("[ERROR] Gagal cek rekening seller ID %d: %v", data.IdentitasSeller.IdSeller, err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	// set default jika belum ada rekening
	var IsDefault bool = false
	if id_data_rekening == 0 {
		IsDefault = true
	} else {
		IsDefault = false
	}

	newRekening := models.RekeningSeller{
		IDSeller:        data.IdentitasSeller.IdSeller,
		NamaBank:        data.NamaBank,
		NomorRekening:   data.NomorRekening,
		PemilikRekening: data.PemilikiRekening,
		IsDefault:       IsDefault,
	}

	// insert rekening baru
	if err_masukan := db.Write.WithContext(ctx).Create(&newRekening).Error; err_masukan != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	go func(Dr models.RekeningSeller, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Dr.IDSeller),
		}

		if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Rekening); err != nil {
			fmt.Println("Gagal increment rekening seller ke threshold seller")
		}

		rekeningSellerThreshold := sot_threshold.RekeningSellerThreshold{
			IdRekeningSeller: Dr.ID,
		}

		if err := rekeningSellerThreshold.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal inisialisasi threshold rekening seller")
		}

		rekeningSellerCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dr).SetTableName(Dr.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningSellerCreatePublish); err != nil {
			fmt.Println("Gagal publish create new rekening kurir ke message broker")
		}
	}(newRekening, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Edit Rekening Seller
// Berfungsi untuk mengedit rekening seller di database
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditRekeningSeller(ctx context.Context, data PayloadEditNorekSeler, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditRekeningSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data seller tidak valid",
		}
	}

	if _, ok := nama_bank.BankMap[data.NamaBank]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal, nama bank tidak diterima",
		}
	}

	var id_data_rekening int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.RekeningSeller{}).Select("id").Where(&models.RekeningSeller{
		ID:       data.IdRekening,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_rekening).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Data seller tidak valid",
		}
	}

	if id_data_rekening == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data seller tidak valid",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.RekeningSeller{}).Where(&models.RekeningSeller{
		ID: data.IdRekening,
	}).Updates(&models.RekeningSeller{
		NamaBank:        data.NamaBank,
		NomorRekening:   data.NomorRekening,
		PemilikRekening: data.PemilikiRekening,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Data seller tidak valid",
		}
	}

	go func(Ir int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var updatedRekeningSeller models.RekeningSeller
		if err := Read.WithContext(konteks).Model(&models.RekeningSeller{}).Where(&models.RekeningSeller{
			ID: Ir,
		}).Limit(1).Take(&updatedRekeningSeller).Error; err != nil {
			fmt.Println("Gagal mengambil data rekening seller")
			return
		}

		rekeningSellerUpdate := mb_cud_serializer.NewJsonPayload().SetPayload(updatedRekeningSeller).SetTableName(updatedRekeningSeller.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningSellerUpdate); err != nil {
			fmt.Println("Gagal mempublish update rekening seller ke messag broker")
		}
	}(data.IdRekening, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Set default rekening seller
// Berfungsi untuk mengubah rekening default
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func SetDefaultRekeningSeller(ctx context.Context, data PayloadSetDefaultRekeningSeller, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "SetDefaultRekeningSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data seller tidak valid",
		}
	}

	var id_data_rekening int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.RekeningSeller{}).Select("id").Where(&models.RekeningSeller{
		ID:       data.IdRekening,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_rekening).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_rekening == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data rekening tidak valid",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.RekeningSeller{}).Where(&models.RekeningSeller{
			IDSeller:  data.IdentitasSeller.IdSeller,
			IsDefault: true,
		}).Update("is_default", false).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.RekeningSeller{}).Where(&models.RekeningSeller{
			ID:       data.IdRekening,
			IDSeller: data.IdentitasSeller.IdSeller,
		}).Update("is_default", true).Error; err != nil {
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

	go func(Ir int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var updatedRekeningSeller models.RekeningSeller
		if err := Read.WithContext(konteks).Model(&models.RekeningSeller{}).Where(&models.RekeningSeller{
			ID: Ir,
		}).Limit(1).Take(&updatedRekeningSeller).Error; err != nil {
			fmt.Println("Gagal mengambil data rekening seller")
			return
		}

		rekeningSellerUpdate := mb_cud_serializer.NewJsonPayload().SetPayload(updatedRekeningSeller).SetTableName(updatedRekeningSeller.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningSellerUpdate); err != nil {
			fmt.Println("Gagal mempublish update set default rekening seller ke messag broker")
		}
	}(data.IdRekening, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Rekening Seller
// Berfungsi untuk Menghapus Data Rekening Seller Yang sudah ada di db
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusRekeningSeller(ctx context.Context, data PayloadHapusNorekSeller, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusRekeningSeller"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, kredensial seller tidak valid",
		}
	}

	// Validasi apakah rekening ada dan milik seller ini
	var data_rekening models.RekeningSeller
	if err := db.Read.WithContext(ctx).
		Model(&models.RekeningSeller{}).
		Select("id").
		Where(&models.RekeningSeller{
			ID:       data.IdRekening,
			IDSeller: data.IdentitasSeller.IdSeller,
		}).
		Limit(1).
		Scan(&data_rekening).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_rekening.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, data rekening tidak valid",
		}
	}

	// Hapus rekening
	if err := db.Write.WithContext(ctx).
		Where(&models.RekeningSeller{
			ID:            data_rekening.ID,
			NomorRekening: data.NomorRekening,
		}).
		Delete(&models.RekeningSeller{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dr models.RekeningSeller, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Dr.IDSeller),
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.Rekening); err != nil {
			fmt.Println("Gagal decr count rekening rekening ke threshold seller")
		}

		if err := Trh.WithContext(konteks).Model(sot_threshold.RekeningSellerThreshold{}).Where(&sot_threshold.RekeningSellerThreshold{
			ID: Dr.ID,
		}).Delete(&sot_threshold.RekeningSellerThreshold{}).Error; err != nil {
			fmt.Println("Gagal menghapus threshold rekening seller")
		}

		rekeningSellerDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dr).SetTableName(Dr.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningSellerDeletePublish); err != nil {
			fmt.Println("Gagal publish delete rekening seller ke message broker")
		}

	}(data_rekening, db.Write, cud_publisher)

	log.Printf("[INFO] Rekening ID %d milik seller ID %d berhasil dihapus", data_rekening.ID, data.IdentitasSeller.IdSeller)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Rekening berhasil dihapus.",
	}
}
