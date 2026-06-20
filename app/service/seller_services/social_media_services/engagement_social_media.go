package seller_social_media_services

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	settings "github.com/anan112pcmec/Burung-backend-1/app/app_settings"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_social_media_seller "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/social_media_services/response_social_media_services"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Engage Media Social Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EngageSocialMediaSeller(ctx context.Context, data PayloadEngageSocialMedia, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EngagementSocialMediaSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial seller tidak valid untuk ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_social_media_seller.ResponseEngageSocialMedia{
				Message: "Kredensial seller tidak valid.",
			},
		}
	}

	var id_sosmed_table int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.EntitySocialMedia{}).Select("id").Where(&models.EntitySocialMedia{
		EntityId:   int64(data.IdentitasSeller.IdSeller),
		EntityType: entity_enums.Seller,
	}).Scan(&id_sosmed_table).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_sosmed_table == 0 {
		if err_buat_kolom := db.Write.WithContext(ctx).Create(&models.EntitySocialMedia{
			EntityId:   int64(data.IdentitasSeller.IdSeller),
			Whatsapp:   data.Data.Whatsapp,
			Facebook:   data.Data.Facebook,
			TikTok:     data.Data.TikTok,
			Instagram:  data.Data.Instagram,
			EntityType: entity_enums.Seller,
		}).Error; err_buat_kolom != nil {
			log.Printf("[ERROR] Gagal menambah data social media untuk seller ID %d: %v", data.IdentitasSeller.IdSeller, err_buat_kolom)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Payload: response_social_media_seller.ResponseEngageSocialMedia{
					Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
				},
			}
		} else {
			go func(IdEsm int64, Read gorm.DB, publisher *mb_cud_publisher.Publisher) {
				ctx := context.Background()
				konteks, cancel := context.WithCancel(ctx)
				defer cancel()

				var Udesm models.EntitySocialMedia
				if err := Read.WithContext(konteks).Model(models.EntitySocialMedia{}).Where(models.EntitySocialMedia{
					ID: IdEsm,
				}).Limit(1).Take(&Udesm).Error; err != nil {
					fmt.Println("Gagal mendapatkan data perubahan seller pembaruan sesi dibatalkan")
					return
				}

				SellerCreatedSocialMediaEntityPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Udesm).SetTableName(Udesm.TableName()).SetRole(mb_cud_seeders.Seller)
				if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, SellerCreatedSocialMediaEntityPublish); err != nil {
					fmt.Println("Gagal publish update data seller ke message broker")
				}

			}(id_sosmed_table, *db.Read, cud_publisher)
		}

		log.Printf("[INFO] Data social media berhasil ditambahkan untuk seller ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusOK,
			Services: services,
			Payload: response_social_media_seller.ResponseEngageSocialMedia{
				Message: "Data social media berhasil ditambahkan.",
			},
		}
	}

	if err_update := db.Write.WithContext(ctx).Model(models.EntitySocialMedia{}).Where(models.EntitySocialMedia{
		ID: id_sosmed_table,
	}).Updates(&models.EntitySocialMedia{
		Whatsapp:  data.Data.Whatsapp,
		Facebook:  data.Data.Facebook,
		TikTok:    data.Data.TikTok,
		Instagram: data.Data.Instagram,
	}).Error; err_update != nil {
		log.Printf("[ERROR] Gagal memperbarui data social media untuk seller ID %d: %v", data.IdentitasSeller.IdSeller, err_update)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_social_media_seller.ResponseEngageSocialMedia{
				Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
			},
		}
	} else {
		go func(IdEsm int64, Read gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx := context.Background()
			konteks, cancel := context.WithTimeout(ctx, settings.TimeoutContext)
			defer cancel()

			var Udesm models.EntitySocialMedia
			if err := Read.WithContext(konteks).Model(models.EntitySocialMedia{}).Where(models.EntitySocialMedia{
				ID: IdEsm,
			}).Limit(1).Take(&Udesm).Error; err != nil {
				fmt.Println("Gagal mendapatkan data perubahan seller pembaruan sesi dibatalkan")
				return
			}

			SellerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Udesm).SetTableName("EngageSocialMediaSeller").SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, SellerUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update data seller ke message broker")
			}

		}(id_sosmed_table, *db.Read, cud_publisher)
	}

	log.Printf("[INFO] Data social media berhasil diperbarui untuk seller ID %d", data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_social_media_seller.ResponseEngageSocialMedia{
			Message: "Data social media berhasil diperbarui.",
		},
	}
}
