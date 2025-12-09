package seller_credential_services

import (
	"context"
	"log"
	"net/http"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/enums/nama_bank"
	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Tambahkan Rekening Seller
// Berfungsi untuk menambahkan rekening seller ke database
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TambahRekeningSeller(ctx context.Context, data PayloadTambahkanNorekSeller, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahRekeningSeller"

	// validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

	// insert rekening baru
	if err_masukan := db.Write.WithContext(ctx).Create(&models.RekeningSeller{
		IDSeller:        data.IdentitasSeller.IdSeller,
		NamaBank:        data.NamaBank,
		NomorRekening:   data.NomorRekening,
		PemilikRekening: data.PemilikiRekening,
		IsDefault:       IsDefault,
	}).Error; err_masukan != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

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

func EditRekeningSeller(ctx context.Context, data PayloadEditNorekSeler, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditRekeningSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

func SetDefaultRekeningSeller(ctx context.Context, data PayloadSetDefaultRekeningSeller, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "SetDefaultRekeningSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

func HapusRekeningSeller(ctx context.Context, data PayloadHapusNorekSeller, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusRekeningSeller"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, kredensial seller tidak valid",
		}
	}

	// Validasi apakah rekening ada dan milik seller ini
	var id_data_rekening int64
	if err := db.Read.WithContext(ctx).
		Model(&models.RekeningSeller{}).
		Select("id").
		Where(&models.RekeningSeller{
			ID:       data.IdRekening,
			IDSeller: data.IdentitasSeller.IdSeller,
		}).
		Limit(1).
		Scan(&id_data_rekening).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_rekening == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, data rekening tidak valid",
		}
	}

	// Hapus rekening
	if err := db.Write.WithContext(ctx).
		Where(&models.RekeningSeller{
			ID:            id_data_rekening,
			NomorRekening: data.NomorRekening,
		}).
		Delete(&models.RekeningSeller{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	log.Printf("[INFO] Rekening ID %d milik seller ID %d berhasil dihapus", id_data_rekening, data.IdentitasSeller.IdSeller)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Rekening berhasil dihapus.",
	}
}
