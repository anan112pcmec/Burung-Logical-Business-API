package pengguna_alamat_services

import (
	"context"
	"net/http"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_kota"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_provinsi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Masukan Alamat Pengguna
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanAlamatPengguna(ctx context.Context, data PayloadMasukanAlamatPengguna, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "MasukanAlamatPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data Pengguna Tidak Valid",
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

	var id_data_alamats []int64
	if err := db.Read.WithContext(ctx).Select("id").Model(&models.AlamatPengguna{}).
		Where(models.AlamatPengguna{IDPengguna: data.IdentitasPengguna.ID}).
		Limit(5).Scan(&id_data_alamats).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi waktu",
		}
	}

	if len(id_data_alamats) == 5 {
		return &response.ResponseForm{
			Status:   http.StatusForbidden,
			Services: services,
			Message:  "Gagal batas alamat hanya 5",
		}
	}

	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatPengguna{}).Select("id").Where(&models.AlamatPengguna{
		IDPengguna:      data.IdentitasPengguna.ID,
		PanggilanAlamat: data.PanggilanAlamat,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_alamat != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah memiliki alamat dengan panggilan yang sama",
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longitude)

	if err := db.Write.WithContext(ctx).Create(&models.AlamatPengguna{
		IDPengguna:      data.IdentitasPengguna.ID,
		PanggilanAlamat: data.PanggilanAlamat,
		NamaAlamat:      data.NamaAlamat,
		Deskripsi:       data.Deskripsi,
		NomorTelephone:  data.NomorTelephone,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodePos:         data.KodePos,
		KodeNegara:      data.KodeNegara,
		Longitude:       data.Longitude,
		Latitude:        data.Latitude,
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

func EditAlamatPengguna(ctx context.Context, data PayloadEditAlamatPengguna, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditAlamatPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Pengguna tidak ditemukan",
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

	var id_alamat_pengguna int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatPengguna{}).Select("id").Where(&models.AlamatPengguna{
		ID:         data.IdAlamatPengguna,
		IDPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_alamat_pengguna).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_alamat_pengguna == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data alamat tidak ditemukan",
		}
	}

	var idDataTransaksi int64 = 0

	if err := db.Read.WithContext(ctx).
		Model(&models.Transaksi{}).
		Select("id").
		Where("id_alamat_pengguna = ? AND status != ?", data.IdAlamatPengguna, "Selesai").
		Limit(1).
		Scan(&idDataTransaksi).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	// Jika ada transaksi yang menggunakan alamat ini
	if idDataTransaksi != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, alamat sedang digunakan sebagai acuan transaksi",
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longitude)

	if err := db.Write.WithContext(ctx).Model(&models.AlamatPengguna{}).Where(&models.AlamatPengguna{
		ID: data.IdAlamatPengguna,
	}).Updates(&models.AlamatPengguna{
		PanggilanAlamat: data.PanggilanAlamat,
		NamaAlamat:      data.NamaAlamat,
		Deskripsi:       data.Deskripsi,
		NomorTelephone:  data.NomorTelephone,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodePos:         data.KodePos,
		KodeNegara:      data.KodeNegara,
		Longitude:       data.Longitude,
		Latitude:        data.Latitude,
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

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Hapus Alamat Pengguna
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusAlamatPengguna(ctx context.Context, data PayloadHapusAlamatPengguna, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusAlamatPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusOK,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_alamat_pengguna int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatPengguna{}).Select("id").Where(&models.AlamatPengguna{
		ID:         data.IdAlamatPengguna,
		IDPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_alamat_pengguna).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_alamat_pengguna == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data alamat tidak ditemukan",
		}
	}

	var idDataTransaksi int64 = 0

	if err := db.Read.WithContext(ctx).
		Model(&models.Transaksi{}).
		Select("id").
		Where("id_alamat_pengguna = ? AND status != ?", data.IdAlamatPengguna, "Selesai").
		Limit(1).
		Scan(&idDataTransaksi).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	// Jika ada transaksi yang menggunakan alamat ini
	if idDataTransaksi != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, alamat sedang digunakan sebagai acuan transaksi",
		}
	}

	if err_hapus := db.Write.WithContext(ctx).Where(models.AlamatPengguna{
		ID: data.IdAlamatPengguna,
	}).Delete(&models.AlamatPengguna{}).Error; err_hapus != nil {
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
