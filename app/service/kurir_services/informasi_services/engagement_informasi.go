package kurir_informasi_services

import (
	"context"
	"log"
	"net/http"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_informasi_services_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/informasi_services/response_informasi_services"
)

func AjukanInformasiKendaraan(ctx context.Context, data PayloadInformasiDataKendaraan, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "AjukanInformasiKendaraanKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_pengajuan_data_kendaraan int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_pengajuan_data_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_pengajuan_data_kendaraan != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, Kamu sudah membuat pengajuan",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.InformasiKendaraanKurir{
		IDkurir:        data.DataIdentitasKurir.IdKurir,
		JenisKendaraan: data.JenisKendaraan,
		NamaKendaraan:  data.NamaKendaraan,
		RodaKendaraan:  data.RodaKendaraan,
		STNK:           data.InformasiStnk,
		BPKB:           data.InformasiBpkb,
		NoRangka:       data.NomorRangka,
		NoMesin:        data.NomorMesin,
		Status:         "Pending",
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	log.Printf("[INFO] Pengajuan informasi kendaraan berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
			Message: "Berhasil.",
		},
	}
}

func EditInformasiKendaraan(ctx context.Context, data PayloadEditInformasiDataKendaraan, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditInformasiKendaraanKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_data_informasi_kendaraan int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		ID:      data.IdInformasiKendaraan,
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal, Server Sedang Sibuk Coba Lagi Lain Waktu",
			},
		}
	}

	if id_data_informasi_kendaraan == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal, Data Tidak Valid",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Where(&models.InformasiKendaraanKurir{
		ID: data.IdInformasiKendaraan,
	}).Updates(&models.InformasiKendaraanKurir{
		JenisKendaraan: data.JenisKendaraan,
		NamaKendaraan:  data.NamaKendaraan,
		RodaKendaraan:  data.RodaKendaraan,
		STNK:           data.InformasiStnk,
		BPKB:           data.InformasiBpkb,
		NoRangka:       data.NomorRangka,
		NoMesin:        data.NomorMesin,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	log.Printf("[INFO] Edit informasi kendaraan berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
			Message: "Berhasil.",
		},
	}

}

func AjukanInformasiKurir(ctx context.Context, data PayloadInformasiDataKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "AjukanInformasiKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_data_pengajuan_informasi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKurir{}).Select("id").Where(&models.InformasiKurir{
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_pengajuan_informasi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_pengajuan_informasi != 0 {
		log.Printf("[WARN] Sudah ada pengajuan data kurir yang belum diproses untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
				Message: "Gagal, tunggu pengajuan sebelumnya ditindak kami.",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.InformasiKurir{
		IDkurir:      data.DataIdentitasKurir.IdKurir,
		TanggalLahir: data.TanggalLahir,
		Alasan:       data.Alasan,
		Ktp:          data.InformasiKtp,
		InformasiSim: data.InformasiSim,
		Status:       "Pending",
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
				Message: "Gagal Server Sedang Sibuk Coba Lagi Lain Waktu",
			},
		}
	}

	log.Printf("[INFO] Pengajuan data kurir berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
			Message: "Berhasil.",
		},
	}
}

func EditInformasiKurir(ctx context.Context, data PayloadEditInformasiDataKurir, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditInformasiKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_data_pengajuan_informasi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKurir{}).Select("id").Where(&models.InformasiKurir{
		ID:      data.IdInformasiKurir,
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_pengajuan_informasi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal, server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_pengajuan_informasi == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal, data Tidak valid",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.InformasiKurir{}).Where(&models.InformasiKurir{
		ID: data.IdInformasiKurir,
	}).Updates(&models.InformasiKurir{
		TanggalLahir: data.TanggalLahir,
		Alasan:       data.Alasan,
		Ktp:          data.InformasiKtp,
		InformasiSim: data.InformasiSim,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	log.Printf("[INFO] Edit data kurir berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
			Message: "Berhasil.",
		},
	}
}
