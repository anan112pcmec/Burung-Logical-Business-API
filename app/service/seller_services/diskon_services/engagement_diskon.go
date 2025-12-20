package seller_diskon_services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	seller_enum "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_diskon_produk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/diskon_produk"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/diskon_services/response_diskon_services_seller"

)

func TambahDiskonProduk(ctx context.Context, data PayloadTambahDiskonProduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahDiskonProduk"

	seller, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
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

	newDiskonProduk := models.DiskonProduk{
		SellerId:      data.IdentitasSeller.IdSeller,
		Nama:          data.Nama,
		Deskripsi:     data.Deskripsi,
		DiskonPersen:  data.DiskonPersen,
		BerlakuMulai:  data.BerlakuMulai,
		BerlakuSampai: data.BerlakuSampai,
		Status:        seller_enum.Draft,
	}

	if err := db.Write.WithContext(ctx).Create(&newDiskonProduk).RowsAffected; err == 0 {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Dp models.DiskonProduk, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Dp.SellerId),
		}

		diskonProdukThreshold := sot_threshold.DiskonProdukThreshold{
			IdDiskonProduk: Dp.ID,
		}

		if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.DiskonProduk); err != nil {
			fmt.Println("Gagal menambah count diskon produk ke threshold seller")
		}

		if err := diskonProdukThreshold.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold diskon produk")
		}

		diskonProdukCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dp).SetTableName(Dp.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, diskonProdukCreatePublish); err != nil {
			fmt.Println("Gagal publish create diskon produk ke message broker")
		}

	}(newDiskonProduk, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseTambahDiskonProduk{
			Message: "Berhasil",
		},
	}
}

func EditDiskonProduk(ctx context.Context, data PayloadEditDiskonProduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditDiskonProduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdDp int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataUpdatedDiskonProduk models.DiskonProduk
		if err := Read.WithContext(konteks).Model(&models.DiskonProduk{}).Where(&models.DiskonProduk{
			ID: IdDp,
		}).Limit(1).Take(&data.IdDiskonProduk).Error; err != nil {
			fmt.Println("Gagal mendapatkan data updated diskon produk")
			return
		}

		diskonProdukUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedDiskonProduk).SetTableName(dataUpdatedDiskonProduk.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, diskonProdukUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated diskon produk ke message broker")
		}
	}(data.IdDiskonProduk, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseEditDiskonProduk{
			Message: "Berhasil",
		},
	}
}

func HapusDiskonProduk(ctx context.Context, data PayloadHapusDiskonProduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDiskonProduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
				Message: "Gagal data seller tidak ditemukan",
			},
		}
	}

	var diskon_produk models.DiskonProduk
	if err := db.Read.WithContext(ctx).Model(&models.DiskonProduk{}).Where(&models.DiskonProduk{
		ID:       data.IdDiskonProduk,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   seller_enum.Draft,
	}).Limit(1).Scan(&diskon_produk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if diskon_produk.ID == 0 {
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
			IdDiskon: diskon_produk.ID,
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

	go func(Dp models.DiskonProduk, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: int64(Dp.SellerId),
		}

		if err := sellerThreshold.Decrement(konteks, Trh, stsk_seller.DiskonProduk); err != nil {
			fmt.Println("Gagal decr count diskon produk ke threshold diskon produk")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.DiskonProdukThreshold{}).Where(&sot_threshold.DiskonProdukThreshold{
			ID: Dp.ID,
		}).Delete(&sot_threshold.DiskonProdukThreshold{}).Error; err != nil {
			fmt.Println("Gagal hapus threshold diskon produk ")
		}

		diskonProdukDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dp).SetTableName(Dp.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, diskonProdukDeletePublish); err != nil {
			fmt.Println("Gagal mempublish delete diskon produk ke message broker")
		}

	}(diskon_produk, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseHapusDiskonProduk{
			Message: "Berhasil",
		},
	}
}

func TetapKanDiskonPadaBarang(ctx context.Context, data PayloadTetapkanDiskonPadaBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TetapkanDiskonPadaBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	newBarangDiDiskon := models.BarangDiDiskon{
		SellerId:         data.IdentitasSeller.IdSeller,
		IdDiskon:         data.IdDiskonProduk,
		IdBarangInduk:    data.IdBarangInduk,
		IdKategoriBarang: data.IdKategoriBarang,
		Status:           seller_enum.Waiting,
	}

	if err := db.Write.WithContext(ctx).Create(&newBarangDiDiskon).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Bdd models.BarangDiDiskon, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Bdd.SellerId),
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Bdd.IdBarangInduk),
		}

		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Bdd.IdKategoriBarang,
		}

		thresholdDiskonProduk := sot_threshold.DiskonProdukThreshold{
			IdDiskonProduk: Bdd.IdDiskon,
		}

		if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.BarangDiDiskon); err != nil {
			fmt.Println("Gagal incr count barang di diskon ke threshold seller")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.BarangDiDiskon); err != nil {
			fmt.Println("Gagal incr count barang di diskon ke threshold barang induk")
		}

		if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.BarangDiDiskon); err != nil {
			fmt.Println("Gagal incr count barang di diskon ke treshold kategori barang")
		}

		if err := thresholdDiskonProduk.Increment(konteks, Trh, stsk_diskon_produk.BarangDiDiskon); err != nil {
			fmt.Println("Gagal incr count barang di diskon ke threshold diskon produk")
		}

		barangDiDiskonCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bdd).SetTableName(Bdd.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangDiDiskonCreatePublish); err != nil {
			fmt.Println("Gagal publish create barang di diskon ke message broker")
		}

	}(newBarangDiDiskon, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseTetapkanDiskonPadaBarang{
			Message: "Berhasil",
		},
	}
}

func HapusDiskonPadaBarang(ctx context.Context, data PayloadHapusDiskonPadaBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDiskonPadaBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var barang_di_diskon models.BarangDiDiskon
	if err := db.Read.WithContext(ctx).Model(&models.BarangDiDiskon{}).Where(&models.BarangDiDiskon{
		ID:       data.IdBarangDiDiskon,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&barang_di_diskon).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if barang_di_diskon.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal barang itu tidak di diskon",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BarangDiDiskon{}).Where(&models.BarangDiDiskon{
		ID: barang_di_diskon.ID,
	}).Delete(&models.BarangDiDiskon{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Bdd models.BarangDiDiskon, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Bdd.SellerId),
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Bdd.IdBarangInduk),
		}

		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Bdd.IdKategoriBarang,
		}

		thresholdDiskonProduk := sot_threshold.DiskonProdukThreshold{
			IdDiskonProduk: Bdd.IdDiskon,
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.BarangDiDiskon); err != nil {
			fmt.Println("Gagal decr count barang di diskon ke threshold seller")
		}

		if err := thresholdBarangInduk.Decrement(konteks, Trh, stsk_baranginduk.BarangDiDiskon); err != nil {
			fmt.Println("Gagal decr count barang di diskon ke threshold barang induk")
		}

		if err := thresholdKategoriBarang.Decrement(konteks, Trh, stsk_kategori_barang.BarangDiDiskon); err != nil {
			fmt.Println("Gagal decr count barang di diskon ke treshold kategori barang")
		}

		if err := thresholdDiskonProduk.Decrement(konteks, Trh, stsk_diskon_produk.BarangDiDiskon); err != nil {
			fmt.Println("Gagal decr count barang di diskon ke threshold diskon produk")
		}

		barangDiDiskonDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bdd).SetTableName(Bdd.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangDiDiskonDeletePublish); err != nil {
			fmt.Println("Gagal publish create barang di diskon ke message broker")
		}

	}(barang_di_diskon, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_diskon_services_seller.ResponseHapusDiskonPadaBarang{
			Message: "Berhasil",
		},
	}
}
