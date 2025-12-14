package seller_diskon_services

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	seller_enum "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/diskon_services/response_diskon_services_seller"
)

func TambahDiskonProduk(ctx context.Context, data PayloadTambahDiskonProduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahDiskonProduk"

	seller, status := data.IdentitasSeller.Validating(ctx, db.Read)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal data seller tidak ditemukan",
			},
		}
	}

	var id_diskon_produk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DiskonProduk{}).Select("id").Where(&models.DiskonProduk{
		SellerId:     data.IdentitasSeller.IdSeller,
		Nama:         data.Nama,
		DiskonPersen: data.DiskonPersen,
	}).Limit(1).Scan(&id_diskon_produk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_diskon_produk != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal kamu sudah memiliki diskon serupa",
			},
		}
	}

	var limit int = 0
	switch seller.Jenis {
	case "Personal":
		limit = 5
	case "Distributor":
		limit = 10
	case "Brand":
		limit = 15
	}

	var id_diskon_produks []int64
	if err := db.Read.WithContext(ctx).Model(&models.DiskonProduk{}).Select("id").Where(&models.DiskonProduk{
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(limit).Scan(&id_diskon_produks).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if len(id_diskon_produks) >= limit {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Kamu telah mencapai batasan Diskon",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.DiskonProduk{
		SellerId:      data.IdentitasSeller.IdSeller,
		Nama:          data.Nama,
		Deskripsi:     data.Deskripsi,
		DiskonPersen:  data.DiskonPersen,
		BerlakuMulai:  data.BerlakuMulai,
		BerlakuSampai: data.BerlakuSampai,
		Status:        seller_enum.Draft,
	}).RowsAffected; err == 0 {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
			Message: "Berhasil",
		},
	}
}

func EditDiskonProduk(ctx context.Context, data PayloadEditDiskonProduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditDiskonProduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal data seller tidak ditemukan",
			},
		}
	}

	var id_diskon_produk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DiskonProduk{}).Select("id").Where(&models.DiskonProduk{
		ID:       data.IdDiskonProduk,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   seller_enum.Draft,
	}).Limit(1).Scan(&id_diskon_produk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_diskon_produk == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal data diskon tidak ditemukan",
			},
		}
	}

	if err := db.Read.WithContext(ctx).Model(&models.DiskonProduk{}).Where(&models.DiskonProduk{
		ID: data.IdDiskonProduk,
	}).Updates(&models.DiskonProduk{
		Nama:          data.Nama,
		Deskripsi:     data.Deskripsi,
		DiskonPersen:  data.DiskonPersen,
		BerlakuMulai:  data.BerlakuMulai,
		BerlakuSampai: data.BerlakuSampai,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseEditDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseEditDiskonProduk{
			Message: "Berhasil",
		},
	}
}

func HapusDiskonProduk(ctx context.Context, data PayloadHapusDiskonProduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusDiskonProduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
				Message: "Gagal data seller tidak ditemukan",
			},
		}
	}

	var id_diskon_produk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DiskonProduk{}).Select("id").Where(&models.DiskonProduk{
		ID:       data.IdDiskonProduk,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   seller_enum.Draft,
	}).Limit(1).Scan(&id_diskon_produk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_diskon_produk == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
				Message: "Gagal data diskon tidak ditemukan",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BarangDiDiskon{}).Where(&models.BarangDiDiskon{
			IdDiskon: data.IdDiskonProduk,
		}).Delete(&models.BarangDiDiskon{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.DiskonProduk{}).Where(&models.DiskonProduk{
			ID: data.IdDiskonProduk,
		}).Delete(&models.DiskonProduk{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
			Message: "Berhasil",
		},
	}
}

func TetapKanDiskonPadaBarang(ctx context.Context, data PayloadTetapkanDiskonPadaBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TetapkanDiskonPadaBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_kategori_barang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
		ID:            data.IdKategoriBarang,
		IdBarangInduk: data.IdBarangInduk,
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_kategori_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_kategori_barang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal Barang tidak ditemukan",
			},
		}
	}

	var id_barang_di_diskon int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangDiDiskon{}).Select("id").Where(&models.BarangDiDiskon{
		SellerId:         data.IdentitasSeller.IdSeller,
		IdBarangInduk:    data.IdBarangInduk,
		IdKategoriBarang: data.IdKategoriBarang,
	}).Limit(1).Scan(&id_barang_di_diskon).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_barang_di_diskon != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal kamu sudah menetapkan barang itu kedalam diskon",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.BarangDiDiskon{
		SellerId:         data.IdentitasSeller.IdSeller,
		IdDiskon:         data.IdDiskonProduk,
		IdBarangInduk:    data.IdBarangInduk,
		IdKategoriBarang: data.IdKategoriBarang,
		Status:           seller_enum.Waiting,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
			Message: "Berhasil",
		},
	}
}

func HapusDiskonPadaBarang(ctx context.Context, data PayloadHapusDiskonPadaBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusDiskonPadaBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_barang_di_diskon int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangDiDiskon{}).Select("id").Where(&models.BarangDiDiskon{
		ID:       data.IdBarangDiDiskon,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_barang_di_diskon).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_barang_di_diskon == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal barang itu tidak di diskon",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BarangDiDiskon{}).Where(&models.BarangDiDiskon{
		ID: id_barang_di_diskon,
	}).Delete(&models.BarangDiDiskon{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
			Message: "Berhasil",
		},
	}
}
