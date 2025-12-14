package pengguna_wishlist_services

import (
	"context"
	"net/http"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func TambahBarangKeWishlist(ctx context.Context, data PayloadTambahBarangKeWishlist, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahBarangKeWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	var id_data_wishlist = 0
	if err := db.Read.WithContext(ctx).Model(&models.Wishlist{}).Select("id").Where(&models.Wishlist{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&id_data_wishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_wishlist != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah memasukan barang itu ke dalam wishlist",
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.Wishlist{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IdBarangInduk,
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

func HapusBarangDariWishlist(ctx context.Context, data PayloadHapusBarangDariWishlist, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusBarangDariWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	var id_data_wishlist int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Wishlist{}).Select("id").Where(&models.Wishlist{
		ID:         data.IdWishlist,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_data_wishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_wishlist == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data wishlist tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Wishlist{}).Where(&models.Wishlist{
		ID: data.IdWishlist,
	}).Delete(&models.Wishlist{}).Error; err != nil {
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
