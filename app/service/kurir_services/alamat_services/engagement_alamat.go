package kurir_alamat_services

import (
	"context"
	"net/http"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_kota"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_provinsi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/alamat_services/response_alamat_service_kurir"
)

func MasukanAlamatKurir(ctx context.Context, data PayloadMasukanAlamatKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "MasukanAlamatKurir"

	// Validasi identitas kurir
	_, valid := data.IdentitasKurir.Validating(ctx, db.Read)
	if !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Data kurir tidak valid",
			},
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

	// Cek apakah alamat sudah ada
	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatKurir{}).Select("id").
		Where(&models.AlamatKurir{IdKurir: data.IdentitasKurir.IdKurir}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Server sedang sibuk, coba lagi lain waktu",
			},
		}
	}

	if id_data_alamat != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Kamu sudah memasukan data alamat",
			},
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longtitude)

	// Simpan alamat baru
	if err := db.Write.WithContext(ctx).Create(&models.AlamatKurir{
		IdKurir:         data.IdentitasKurir.IdKurir,
		PanggilanAlamat: data.PanggilanAlamat,
		NomorTelephone:  data.NomorTelephone,
		NamaAlamat:      data.NamaAlamat,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodeNegara:      data.KodeNegara,
		KodePos:         data.KodePos,
		Deskripsi:       data.Deskripsi,
		Longitude:       data.Longtitude,
		Latitude:        data.Latitude,
	}).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Tidak dapat menyimpan data alamat, coba lagi lain waktu",
			},
		}
	}

	// Berhasil
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
			Message: "Berhasil",
		},
	}
}

func EditAlamatKurir(ctx context.Context, data PayloadEditAlamatKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditAlamatKurir"

	// Validasi identitas kurir
	_, valid := data.IdentitasKurir.Validating(ctx, db.Read)
	if !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal: Data kurir tidak valid",
			},
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

	// Cek apakah alamat dengan ID dan kurir terkait benar-benar ada
	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatKurir{}).Select("id").Where(&models.AlamatKurir{
		ID:      data.IDAlamatKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal Data Alamat Tidak Valid",
			},
		}
	}

	if id_data_alamat == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal: Data alamat tidak ditemukan",
			},
		}
	}

	// Update data alamat
	if err := db.Write.WithContext(ctx).Model(&models.AlamatKurir{}).
		Where(&models.AlamatKurir{ID: data.IDAlamatKurir}).
		Updates(&models.AlamatKurir{
			PanggilanAlamat: data.PanggilanAlamat,
			NomorTelephone:  data.NomorTelephone,
			NamaAlamat:      data.NamaAlamat,
			Kota:            data.Kota,
			Provinsi:        data.Provinsi,
			KodeNegara:      data.KodeNegara,
			KodePos:         data.KodePos,
			Deskripsi:       data.Deskripsi,
			Longitude:       data.Longtitude,
			Latitude:        data.Latitude,
		}).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal: Mengubah data alamat, coba lagi nanti",
			},
		}
	}

	// Berhasil
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
			Message: "Berhasil",
		},
	}
}

func HapusAlamatKurir(ctx context.Context, data PayloadHapusAlamatKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusAlamatKurir"

	// Validasi identitas kurir
	_, valid := data.IdentitasKurir.Validating(ctx, db.Read)
	if !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal: Data kurir tidak valid",
			},
		}
	}

	// Cek apakah alamat milik kurir tersebut ada
	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatKurir{}).Select("id").Where(&models.AlamatKurir{
		ID:      data.IdAlamatKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Take(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal Data Alamat Tidak Valid",
			},
		}
	}

	if id_data_alamat == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal: Data alamat tidak ditemukan",
			},
		}
	}

	// Hapus alamat
	if err := db.Write.Model(&models.AlamatKurir{}).
		Where(&models.AlamatKurir{
			ID:      data.IdAlamatKurir,
			IdKurir: data.IdentitasKurir.IdKurir,
		}).
		Delete(&models.AlamatKurir{}).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal: Server sedang sibuk, coba lagi lain waktu",
			},
		}
	}

	// Berhasil
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
			Message: "Berhasil",
		},
	}
}
