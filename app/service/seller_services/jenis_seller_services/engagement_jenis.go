package jenis_seller_services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/jenis_seller_services/response_jenis_seller"
)

func MasukanDataDistributor(ctx context.Context, data PayloadMasukanDataDistributor, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanDataDistributor"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataDistributor{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_distributor).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataDistributor{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_distributor != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataDistributor{
				Message: "Gagal kamu sudah mengajukan data!.",
			},
		}
	}

	newDistributorData := models.DistributorData{
		SellerId:                  data.IdentitasSeller.IdSeller,
		NamaPerusahaan:            data.NamaPerusahaan,
		NIB:                       data.NIB,
		NPWP:                      data.NPWP,
		Alasan:                    data.Alasan,
		DokumenIzinDistributorUrl: data.DokumenIzinDistributorUrl,
		Status:                    "Pending",
	}

	if err := db.Write.WithContext(ctx).Create(&newDistributorData).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataDistributor{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Ddata models.DistributorData, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: int64(Ddata.SellerId),
		}

		distributorDataThreshold := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Ddata.ID,
		}

		if err := sellerThreshold.Increment(konteks, Trh, stsk_seller.DistributorData); err != nil {
			fmt.Println("Gagal incr count distributor data ke seller threshold")
		}

		if err := distributorDataThreshold.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat distributor data threshold")
		}

		distributorDataCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ddata).SetTableName(Ddata.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataCreatePublish); err != nil {
			fmt.Println("Gagal publish create distributor data ke message broker")
		}

	}(newDistributorData, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_jenis_seller.ResponseMasukanDataDistributor{
			Message: "Berhasil",
		},
	}
}

func EditDataDistributor(ctx context.Context, data PayloadEditDataDistributor, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditDataDistributor"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataDistributor{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   "Pending",
	}).Limit(1).Scan(&id_data_distributor).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataDistributor{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataDistributor{
				Message: "Gagal data yang dituju tidak ada",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.DistributorData{}).Where(&models.DistributorData{
		ID: data.IdDistributorData,
	}).Updates(&models.DistributorData{
		NamaPerusahaan:            data.NamaPerusahaan,
		NIB:                       data.NIB,
		NPWP:                      data.NPWP,
		Alasan:                    data.Alasan,
		DokumenIzinDistributorUrl: data.DokumenIzinDistributorUrl,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataDistributor{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(IdD int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var distributorDataUpdated models.DistributorData
		if err := Read.WithContext(konteks).Model(&models.DistributorData{}).Where(&models.DistributorData{
			ID: IdD,
		}).Limit(1).Take(&distributorDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil distributor data updated")
			return
		}

		distributorDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(distributorDataUpdated).SetTableName(distributorDataUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated distributor data ke message broker")
		}
	}(data.IdDistributorData, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_jenis_seller.ResponseEditDataDistributor{
			Message: "Berhasil",
		},
	}
}

func HapusDataDistributor(ctx context.Context, data PayloadHapusDataDistributor, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDataDistributor"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseHapusDataDistributor{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var data_distributor models.DistributorData
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   "Pending",
	}).Limit(1).Scan(&data_distributor).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseHapusDataDistributor{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if data_distributor.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseHapusDataDistributor{
				Message: "Gagal data yang dituju tidak ada",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.DistributorData{}).Where(&models.DistributorData{
		ID: data.IdDistributorData,
	}).Delete(&models.DistributorData{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseHapusDataDistributor{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Ddata models.DistributorData, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: int64(Ddata.SellerId),
		}

		if err := sellerThreshold.Decrement(konteks, Trh, stsk_seller.DistributorData); err != nil {
			fmt.Println("Gagal decr count distributor data ke seller threshold")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.DistributorDataThreshold{}).Where(&sot_threshold.DistributorDataThreshold{
			IdDistributorData: Ddata.ID,
		}).Delete(&sot_threshold.DistributorDataThreshold{}).Error; err != nil {
			fmt.Println("Gagal menghapus threshold data distributor")
		}

		distributorDataCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ddata).SetTableName(Ddata.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataCreatePublish); err != nil {
			fmt.Println("Gagal publish delete distributor data ke message broker")
		}

	}(data_distributor, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_jenis_seller.ResponseHapusDataDistributor{
			Message: "Berhasil",
		},
	}
}

func MasukanDataBrand(ctx context.Context, data PayloadMasukanDataBrand, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanDataBrand"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataBrand{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataBrand{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_brand != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataBrand{
				Message: "Gagal kamu sudah mengajukan data brand!.",
			},
		}
	}

	newBrandData := models.BrandData{
		SellerId:              data.IdentitasSeller.IdSeller,
		NamaPerusahaan:        data.NamaPerusahaan,
		NegaraAsal:            data.NegaraAsal,
		LembagaPendaftaran:    data.LembagaPendaftaran,
		NomorPendaftaranMerek: data.NomorPendaftaranMerek,
		SertifikatMerekUrl:    data.SertifikatMerekUrl,
		DokumenPerwakilanUrl:  data.DokumenPerwakilanUrl,
		NIB:                   data.NIB,
		NPWP:                  data.NPWP,
		Alasan:                data.Alasan,
		Status:                "Pending",
	}

	if err := db.Write.WithContext(ctx).Create(&newBrandData).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataBrand{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Db models.BrandData, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: int64(Db.SellerId),
		}

		brandDataThreshold := sot_threshold.BrandDataThreshold{
			IdBrandData: Db.ID,
		}

		if err := sellerThreshold.Increment(konteks, Trh, stsk_seller.BrandData); err != nil {
			fmt.Println("Gagal incr count brand data ke seller threshold")
		}

		if err := brandDataThreshold.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold brand data ")
		}

		brandDataCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Db).SetTableName(Db.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataCreatePublish); err != nil {
			fmt.Println("Gagal publish create brand data ke message broker")
		}
	}(newBrandData, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_jenis_seller.ResponseMasukanDataBrand{
			Message: "Berhasil",
		},
	}
}

func EditDataBrand(ctx context.Context, data PayloadEditDataBrand, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditDataBrand"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataBrand{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdDataBrand,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   "Pending",
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataBrand{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_brand == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataBrand{
				Message: "Gagal kamu sudah mengajukan data brand!.",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BrandData{}).Where(&models.BrandData{
		ID: data.IdDataBrand,
	}).Updates(&models.BrandData{
		NamaPerusahaan:        data.NamaPerusahaan,
		NegaraAsal:            data.NegaraAsal,
		LembagaPendaftaran:    data.LembagaPendaftaran,
		NomorPendaftaranMerek: data.NomorPendaftaranMerek,
		SertifikatMerekUrl:    data.SertifikatMerekUrl,
		DokumenPerwakilanUrl:  data.DokumenPerwakilanUrl,
		NIB:                   data.NIB,
		NPWP:                  data.NPWP,
		Alasan:                data.Alasan,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataBrand{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(IdB int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var brandDataUpdated models.BrandData
		if err := Read.WithContext(konteks).Model(&models.BrandData{}).Where(&models.BrandData{
			ID: IdB,
		}).Limit(1).Take(&brandDataUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data brand data")
			return
		}

		brandDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(brandDataUpdated).SetTableName(brandDataUpdated.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataUpdatedPublish); err != nil {
			fmt.Println("Gagal publish brand data create ke message broker")
		}
	}(id_data_brand, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_jenis_seller.ResponseEditDataBrand{
			Message: "Berhasil",
		},
	}
}

func HapusDataBrand(ctx context.Context, data PayloadHapusDataBrand, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDataBrand"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseMasukanDataDistributor{
				Message: "Gagal data seller tidak valid",
			},
		}
	}

	var data_brand models.BrandData
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Where(&models.BrandData{
		ID:       data.IdDataBrand,
		SellerId: data.IdentitasSeller.IdSeller,
		Status:   "Pending",
	}).Limit(1).Scan(&data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataBrand{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if data_brand.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_jenis_seller.ResponseEditDataBrand{
				Message: "Gagal data tidak ditemukan!.",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BrandData{}).Where(&models.BrandData{
		ID: data.IdDataBrand,
	}).Delete(&models.BrandData{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_jenis_seller.ResponseHapusDataBrand{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Db models.BrandData, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: int64(Db.SellerId),
		}

		if err := sellerThreshold.Decrement(konteks, Trh, stsk_seller.BrandData); err != nil {
			fmt.Println("Gagal decr count brand data ke seller threshold")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.BrandDataThreshold{}).Where(&sot_threshold.BrandDataThreshold{
			IdBrandData: Db.ID,
		}).Delete(&sot_threshold.BrandDataThreshold{}).Error; err != nil {
			fmt.Println("Gagal delete threshold brand data")
		}

		brandDataDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Db).SetTableName(Db.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataDeletePublish); err != nil {
			fmt.Println("Gagal publish create brand data ke message broker")
		}
	}(data_brand, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_jenis_seller.ResponseHapusDataBrand{
			Message: "Berhasil",
		},
	}
}
