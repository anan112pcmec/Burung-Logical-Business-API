package kurir_rekening_services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_bank"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/rekening_services/response_rekening_services_kurir"
)

func MasukanRekeningKurir(ctx context.Context, data PayloadMasukanRekeningKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanRekeningKurir"

	_, validasi := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !validasi {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Gagal Data Kurir Tidak Valid",
			},
		}
	}

	if _, ok := nama_bank.BankMap[data.NamaBank]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal, nama bank tidak diterima",
		}
	}

	var id_alamat int64 = 0
	if err := db.Read.WithContext(ctx).
		Model(&models.RekeningKurir{}).
		Select("id").
		Where(&models.RekeningKurir{IdKurir: data.IdentitasKurir.IdKurir}).
		Limit(1).
		Scan(&id_alamat).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Gagal, server sedang sibuk coba lagi nanti",
			},
		}
	}

	if id_alamat != 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Maksimal hanya memasukan 1 rekening",
			},
		}
	}

	newRekeningKurir := models.RekeningKurir{
		IdKurir:         data.IdentitasKurir.IdKurir,
		NamaBank:        data.NamaBank,
		NomorRekening:   data.NomorRekening,
		PemilikRekening: data.PemilikRekening,
	}

	if err := db.Write.WithContext(ctx).Create(&newRekeningKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Gagal, server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Rk models.RekeningKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		kurirThreshold := sot_threshold.KurirThreshold{
			IdKurir: Rk.IdKurir,
		}

		if err := kurirThreshold.Increment(konteks, Trh, stsk_kurir.RekeningKurir); err != nil {
			fmt.Println("Gagal incr count rekening kurir ke threshold seller")
		}

		rekeningKurirCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Rk).SetTableName(Rk.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningKurirCreatePublish); err != nil {
			fmt.Println("Gagal publish create rekening kurir ke message broker")
		}
	}(newRekeningKurir, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
			Message: "Berhasil",
		},
	}
}

func EditRekeningKurir(ctx context.Context, data PayloadEditRekeningKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditRekeningKurir"

	_, validasi := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !validasi {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseEditRekeningKurir{
				Message: "Gagal menemukan data kurir",
			},
		}
	}

	if _, ok := nama_bank.BankMap[data.NamaBank]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal, nama bank tidak diterima",
		}
	}

	var id_alamat int64 = 0
	if err := db.Read.WithContext(ctx).
		Model(&models.RekeningKurir{}).
		Select("id").
		Where(&models.RekeningKurir{ID: data.IdRekening, IdKurir: data.IdentitasKurir.IdKurir}).
		Limit(1).
		Scan(&id_alamat).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Gagal, server sedang sibuk coba lagi nanti",
			},
		}
	}

	if id_alamat == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Maksimal hanya memasukan 1 rekening",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.RekeningKurir{}).Where(&models.RekeningKurir{
		ID: data.IdRekening,
	}).Updates(&models.RekeningKurir{
		NamaBank:        data.NamaBank,
		NomorRekening:   data.NomorRekening,
		PemilikRekening: data.PemilikRekening,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseEditRekeningKurir{
				Message: "Gagal, server sedang sibuk, coba lagi lain waktu",
			},
		}
	}

	go func(IdRk int64, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var rekeningKurirUpdated models.RekeningKurir
		if err := Trh.WithContext(konteks).Model(&models.RekeningKurir{}).Where(&models.RekeningKurir{
			ID: IdRk,
		}).Limit(1).Take(&rekeningKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kurir")
			return
		}

		rekeningKurirUpdatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(rekeningKurirUpdated).SetTableName(rekeningKurirUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningKurirUpdatePublish); err != nil {
			fmt.Println("Gagal publish create rekening kurir ke message broker")
		}
	}(id_alamat, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_rekening_services_kurir.ResponseEditRekeningKurir{
			Message: "Berhasil",
		},
	}
}

func HapusRekeningKurir(ctx context.Context, data PayloadHapusRekeningKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusRekeningKurir"

	_, validasi := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !validasi {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseHapusRekeningKurir{
				Message: "Gagal menemukan data kurir",
			},
		}
	}

	var data_rekening models.RekeningKurir
	if err := db.Read.WithContext(ctx).
		Model(&models.RekeningKurir{}).
		Select("id").
		Where(&models.RekeningKurir{ID: data.IdRekening, IdKurir: data.IdentitasKurir.IdKurir}).
		Limit(1).
		Scan(&data_rekening).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Gagal, server sedang sibuk coba lagi nanti",
			},
		}
	}

	if data_rekening.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseMasukanRekeningKurir{
				Message: "Data Rekening Tidak Ditemukan",
			},
		}
	}

	if err := db.Write.Model(&models.RekeningKurir{}).Where(&models.RekeningKurir{
		ID:      data.IdRekening,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Delete(&models.RekeningKurir{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_rekening_services_kurir.ResponseHapusRekeningKurir{
				Message: "Gagal, server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Rk models.RekeningKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		kurirThreshold := sot_threshold.KurirThreshold{
			IdKurir: Rk.IdKurir,
		}

		if err := kurirThreshold.Decrement(konteks, Trh, stsk_kurir.RekeningKurir); err != nil {
			fmt.Println("Gagal decr count rekening kurir ke threshold seller")
		}

		rekeningKurirDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Rk).SetTableName(Rk.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, rekeningKurirDeletePublish); err != nil {
			fmt.Println("Gagal publish delete rekening kurir ke message broker")
		}
	}(data_rekening, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_rekening_services_kurir.ResponseHapusRekeningKurir{
			Message: "Berhasil",
		},
	}
}
