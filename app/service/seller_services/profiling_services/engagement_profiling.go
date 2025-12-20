package seller_profiling_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	cache_db_entity_sessioning_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/cache_database/entity_sessioning/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	seller_particular_profiling "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/profiling_services/particular_profiling"
	seller_response_profiling "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/profiling_services/response_profiling"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Update Personal Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UpdatePersonalSeller(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadUpdateProfilePersonalSeller, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UpdatePersonalSeller"
	var hasil_update_nama seller_particular_profiling.ResponseUbahNama
	var hasil_update_username seller_particular_profiling.ResponseUbahUsername
	var hasil_update_gmail seller_particular_profiling.ResponseUbahEmail

	seller, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if data.Username == "not" {
		log.Println("[INFO] Username kosong, tidak diupdate.")
	} else {
		hasil_update_username = seller_particular_profiling.UbahUsernameSeller(ctx, data.IdentitasSeller.IdSeller, data.Username, db)
	}

	if data.Nama == "not" {
		log.Println("[INFO] Nama kosong, tidak diupdate.")
	} else {
		hasil_update_nama = seller_particular_profiling.UbahNamaSeller(ctx, data.IdentitasSeller.IdSeller, data.Nama, db)
	}

	if data.Email == "not" {
		log.Println("[INFO] Email kosong, tidak diupdate.")
	} else {
		hasil_update_gmail = seller_particular_profiling.UbahEmailSeller(ctx, data.IdentitasSeller.IdSeller, data.Email, db)
	}

	go func(BSeller models.Seller, Read *gorm.DB, rds_sesi *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		KeyLama := cache_db_entity_sessioning_seeders.SetSessionKey[*models.Seller](&BSeller)
		var uptodateSellerData models.Seller
		if err := Read.WithContext(konteks).Model(&models.Seller{}).Where(&models.Seller{
			ID: BSeller.ID,
		}).Limit(1).Take(&uptodateSellerData); err != nil {
			fmt.Println("Gagal mengambil data seller terbaru")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionKey[*models.Seller](konteks, &uptodateSellerData, KeyLama, rds_sesi); err != nil {
			fmt.Println("Gagal update cache key dan data")
			return
		}

		updatedSellerDataPublish := mb_cud_serializer.NewJsonPayload().SetPayload(uptodateSellerData).SetTableName(uptodateSellerData.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedSellerDataPublish); err != nil {
			fmt.Println("Gagal publish update data seller ke message broker")
		}
	}(seller, db.Read, rds_session, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: seller_response_profiling.ResponseUpdateProfileSeller{
			UpdateNama:     hasil_update_nama,
			UpdateUsername: hasil_update_username,
			UpdateGmail:    hasil_update_gmail,
		},
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Update Info General Public
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UpdateInfoGeneralPublic(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadUpdateInfoGeneralSeller, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UpdatePersonalSeller"
	var hasil_update_punchline seller_particular_profiling.ResponseUbahPunchline
	var hasil_update_deskripsi seller_particular_profiling.ResponseUbahDeskripsi
	var hasil_update_jam_operasional seller_particular_profiling.ResponseUbahJamOperasional
	var hasil_update_dedication seller_particular_profiling.ResponseUbahDedication

	seller, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if data.Deskripsi != "not" && data.Deskripsi != "" && data.Deskripsi != seller.Deskripsi {
		hasil_update_deskripsi = seller_particular_profiling.UbahDeskripsiSeller(ctx, data.IdentitasSeller.IdSeller, data.Deskripsi, db)
	} else {
		log.Println("[INFO] Deskripsi kosong atau tidak berubah, tidak diupdate.")
	}

	if data.Punchline != "not" && data.Punchline != "" && data.Punchline != seller.Punchline {
		hasil_update_punchline = seller_particular_profiling.UbahPunchlineSeller(ctx, data.IdentitasSeller.IdSeller, data.Punchline, db)
	} else {
		log.Println("[INFO] Punchline kosong atau tidak berubah, tidak diupdate.")
	}

	if data.JamOperasional != "not" && data.JamOperasional != "" && data.JamOperasional != seller.JamOperasional {
		hasil_update_jam_operasional = seller_particular_profiling.UbahJamOperasionalSeller(ctx, data.IdentitasSeller.IdSeller, data.JamOperasional, db)
	} else {
		log.Println("[INFO] Jam operasional kosong atau tidak berubah, tidak diupdate.")
	}

	if data.Dedication != "not" && data.Dedication != "" && data.Dedication != seller.SellerDedication {
		hasil_update_dedication = seller_particular_profiling.UbahSellerDedication(ctx, data.IdentitasSeller.IdSeller, data.Dedication, db)
	} else {
		log.Println("[INFO] Dedication seller kosong atau tidak berubah, tidak diupdate.")
	}

	go func(IdS int32, Read *gorm.DB, rds_sesi *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataUpdatedSeller models.Seller
		if err := Read.WithContext(konteks).Model(&models.Seller{}).Where(&models.Seller{
			ID: IdS,
		}).Limit(1).Take(&dataUpdatedSeller).Error; err != nil {
			fmt.Println("Gagal mendapatkan data perubahan seller pembaruan sesi dibatalkan")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Seller](konteks, &dataUpdatedSeller, rds_sesi); err != nil {
			fmt.Println("Gagal memperbarui session seller data")
		}

		sellerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(data.IdentitasSeller).SetTableName(dataUpdatedSeller.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, sellerUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update data seller ke message broker")
		}
	}(seller.ID, db.Read, rds_session, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: seller_response_profiling.ResponseUpdateInfoGeneralSeller{
			UpdatePunchline:      hasil_update_punchline,
			UpdateDeskripsi:      hasil_update_deskripsi,
			UpdateJamOperasional: hasil_update_jam_operasional,
			UpdateDedication:     hasil_update_dedication,
		},
	}
}
