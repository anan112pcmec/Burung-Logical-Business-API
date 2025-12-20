package seller_etalase_services

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
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_etalase "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/etalase"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_etalase_services_seller "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/etalase_services/response_etalase_services"
)

func TambahEtalaseSeller(ctx context.Context, data PayloadMenambahEtalase, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	newEtalase := models.Etalase{
		SellerID:     int64(data.IdentitasSeller.IdSeller),
		Nama:         data.NamaEtalase,
		Deskripsi:    data.Deskripsi,
		JumlahBarang: 0,
	}

	if err := db.Write.WithContext(ctx).Create(&newEtalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseMenambahEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Es models.Etalase, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: Es.SellerID,
		}

		thresholdEtalase := sot_threshold.EtalaseThreshold{
			IdEtalase: Es.ID,
		}

		if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Etalase); err != nil {
			fmt.Println("Gagal incr count etalase ke threshold seller")
		}

		if err := thresholdEtalase.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold etalase")
		}

		etalaseCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Es).SetTableName(Es.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseCreatePublish); err != nil {
			fmt.Println("Gagal publish create etalase ke message broker")
		}
	}(newEtalase, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseMenambahEtalase{
			Message: "Berhasil",
		},
	}
}

func EditEtalaseSeller(ctx context.Context, data PayloadEditEtalase, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdE int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataUpdatedEtalase models.Etalase
		if err := Read.WithContext(konteks).Model(&models.Etalase{}).Where(&models.Etalase{
			ID: IdE,
		}).Limit(1).Take(&dataUpdatedEtalase).Error; err != nil {
			fmt.Println("Gagal mendapatkan data etalase updated")
			return
		}

		etalaseUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedEtalase).SetTableName(dataUpdatedEtalase.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseUpdatedPublish); err != nil {
			fmt.Println("Gagal mempublish update etalase ke message broker")
		}
	}(data.IdEtalase, db.Read, cud_publisher)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseEditEtalase{
			Message: "Berhasil",
		},
	}
}

func HapusEtalaseSeller(ctx context.Context, data PayloadHapusEtalase, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	var data_etalase models.Etalase
	if err := db.Read.WithContext(ctx).Model(&models.Etalase{}).Where(&models.Etalase{
		ID:       data.IdEtalase,
		SellerID: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&data_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if data_etalase.ID == 0 {
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
			return err
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

	go func(De models.Etalase, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		sellerThreshold := sot_threshold.SellerThreshold{
			IdSeller: De.SellerID,
		}

		if err := sellerThreshold.Decrement(konteks, Trh, stsk_seller.Etalase); err != nil {
			fmt.Println("Gagal decr count etalase ke threshold seller")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.EtalaseThreshold{}).Where(&sot_threshold.EtalaseThreshold{
			IdEtalase: De.ID,
		}).Delete(&sot_threshold.EtalaseThreshold{}).Error; err != nil {
			fmt.Println("Gagal menghapus threshold etalase")
		}

		etalaseDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(De).SetTableName(De.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseDeletePublish); err != nil {
			fmt.Println("Gagal publish delete etalase ke message broker")
		}
	}(data_etalase, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseHapusEtalase{
			Message: "Berhasil",
		},
	}
}

func TambahkanBarangKeEtalase(ctx context.Context, data PayloadTambahkanBarangKeEtalase, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahkanBarangKeEtalase"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	newBarangKeEtalase := models.BarangKeEtalase{
		IdBarangInduk: data.IdBarangInduk,
		IdEtalase:     data.IdEtalase,
	}

	if err := db.Write.WithContext(ctx).Create(&newBarangKeEtalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Bke models.BarangKeEtalase, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Bke.IdBarangInduk,
		}

		etalaseThreshold := sot_threshold.EtalaseThreshold{
			IdEtalase: Bke.IdEtalase,
		}

		if err := barangIndukThreshold.Increment(konteks, Trh, stsk_baranginduk.BarangKeEtalase); err != nil {
			fmt.Println("Gagal incr count barang ke etalase ke threshold barang induk")
		}

		if err := etalaseThreshold.Increment(konteks, Trh, stsk_etalase.BarangKeEtalase); err != nil {
			fmt.Println("Gagal incr count barang ke etalase ke threshold etalase")
		}

		barangKeEtalaseCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bke).SetTableName(Bke.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangKeEtalaseCreatePublish); err != nil {
			fmt.Println("Gagal publish create barang ke etalase ke message broker")
		}

	}(newBarangKeEtalase, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseTambahBarangKeEtalase{
			Message: "Berhasil",
		},
	}
}

func HapusBarangDariEtalase(ctx context.Context, data PayloadHapusBarangDiEtalase, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusBarangDariEtalase"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	var barang_ke_etalase models.BarangKeEtalase
	if err := db.Read.WithContext(ctx).Model(&models.BarangKeEtalase{}).Where(&models.BarangKeEtalase{
		ID:            data.IdBarangKeEtalase,
		IdEtalase:     data.IdEtalase,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&barang_ke_etalase).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
				Message: "Gagal server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if barang_ke_etalase.ID == 0 {
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

	go func(Bke models.BarangKeEtalase, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Bke.IdBarangInduk,
		}

		etalaseThreshold := sot_threshold.EtalaseThreshold{
			IdEtalase: Bke.IdEtalase,
		}

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.BarangKeEtalase); err != nil {
			fmt.Println("Gagal decr count barang ke etalase ke threshold barang induk")
		}

		if err := etalaseThreshold.Decrement(konteks, Trh, stsk_etalase.BarangKeEtalase); err != nil {
			fmt.Println("Gagal decr count barang ke etalase ke threshold etalase")
		}

		barangKeEtalaseCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bke).SetTableName(Bke.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangKeEtalaseCreatePublish); err != nil {
			fmt.Println("Gagal publish create barang ke etalase ke message broker")
		}

	}(barang_ke_etalase, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_etalase_services_seller.ResponseHapusBarangKeEtalase{
			Message: "Berhasil",
		},
	}
}
