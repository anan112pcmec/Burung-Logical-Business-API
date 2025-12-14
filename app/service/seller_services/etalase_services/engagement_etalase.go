package seller_etalase_services

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_etalase_services_seller "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/etalase_services/response_etalase_services"
)

func TambahEtalaseSeller(ctx context.Context, data PayloadMenambahEtalase, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseMenambahEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	var id_data_etalase int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Etalase{}).Select("id").Where(&models.Etalase{
		SellerID: int64(data.IdentitasSeller.IdSeller),
		Nama:     data.NamaEtalase,
	}).Limit(1).Scan(&id_data_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseMenambahEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_etalase != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_etalase_services_seller.ResponseMenambahEtalase{
				Message: "Gagal kamu sudah memiliki etalase dengan nama itu",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.Etalase{
		SellerID:     int64(data.IdentitasSeller.IdSeller),
		Nama:         data.NamaEtalase,
		Deskripsi:    data.Deskripsi,
		JumlahBarang: 0,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseMenambahEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseMenambahEtalase{
			Message: "Berhasil",
		},
	}
}

func EditEtalaseSeller(ctx context.Context, data PayloadEditEtalase, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseEditEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	var id_data_etalase int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Etalase{}).Select("id").Where(&models.Etalase{
		ID:       data.IdEtalase,
		SellerID: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&id_data_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseEditEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_etalase == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_etalase_services_seller.ResponseEditEtalase{
				Message: "Gagal etalase tak ditemukan",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Etalase{}).Where(&models.Etalase{
		ID: data.IdEtalase,
	}).Updates(&models.Etalase{
		Nama:      data.NamaEtalase,
		Deskripsi: data.Deskripsi,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseEditEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseEditEtalase{
			Message: "Berhasil",
		},
	}
}

func HapusEtalaseSeller(ctx context.Context, data PayloadHapusEtalase, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	var id_data_etalase int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Etalase{}).Select("id").Where(&models.Etalase{
		ID:       data.IdEtalase,
		SellerID: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&id_data_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_etalase == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusEtalase{
				Message: "Gagal etalase tidak ditemukan",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BarangKeEtalase{}).Where(&models.BarangKeEtalase{
			IdEtalase: data.IdEtalase,
		}).Delete(&models.BarangKeEtalase{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Etalase{}).Where(&models.Etalase{
			ID: data.IdEtalase,
		}).Delete(&models.Etalase{}).Error; err != nil {

		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseHapusEtalase{
			Message: "Berhasil",
		},
	}
}

func TambahkanBarangKeEtalase(ctx context.Context, data PayloadTambahkanBarangKeEtalase, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahkanBarangKeEtalase"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_barang_ke_etalase int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangKeEtalase{}).Select("id").Where(&models.BarangKeEtalase{
		IdEtalase:     data.IdEtalase,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&id_barang_ke_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_barang_ke_etalase != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
				Message: "Gagal barang itu sudah termasuk dalam etalase ini",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.BarangKeEtalase{
		IdBarangInduk: data.IdBarangInduk,
		IdEtalase:     data.IdEtalase,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
			Message: "Berhasil",
		},
	}
}

func HapusBarangDariEtalase(ctx context.Context, data PayloadHapusBarangDiEtalase, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusBarangDariEtalase"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	var id_barang_ke_etalase int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangKeEtalase{}).Select("id").Where(&models.BarangKeEtalase{
		ID:            data.IdBarangKeEtalase,
		IdEtalase:     data.IdEtalase,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&id_barang_ke_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_barang_ke_etalase == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
				Message: "Gagal barang tida termasuk dalam etalase",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BarangKeEtalase{}).Where(&models.BarangKeEtalase{
		ID: data.IdBarangKeEtalase,
	}).Delete(&models.BarangKeEtalase{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
			Message: "Berhasil",
		},
	}
}
