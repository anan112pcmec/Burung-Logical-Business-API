package kurir_social_media_services

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"

	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_social_media_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/social_media_services/response_social_media_services"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

func EngagementSocialMediaKurir(ctx context.Context, data PayloadEngageSocialMedia, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EngagementSocialMediaKurir"

	if _, status := data.DataIdentitas.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitas.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_social_media_kurir.ResponseEngageSocialMedia{
				Message: "Kredensial kurir tidak valid.",
			},
		}
	}

	var id_sosmed_table int64 = 0
	if err := db.Read.Model(sot_models.EntitySocialMedia{}).Select("id").Where(sot_models.EntitySocialMedia{
		EntityId:   data.DataIdentitas.IdKurir,
		EntityType: entity_enums.Kurir,
	}).Limit(1).Scan(&id_sosmed_table).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_sosmed_table == 0 {
		if err_buat_kolom := db.Write.WithContext(ctx).Create(&sot_models.EntitySocialMedia{
			EntityId:   data.DataIdentitas.IdKurir,
			Whatsapp:   data.Data.Whatsapp,
			Facebook:   data.Data.Facebook,
			TikTok:     data.Data.TikTok,
			Instagram:  data.Data.Instagram,
			EntityType: entity_enums.Kurir,
		}).Error; err_buat_kolom != nil {
			log.Printf("[ERROR] Gagal menambah data social media untuk kurir ID %d: %v", data.DataIdentitas.IdKurir, err_buat_kolom)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Payload: response_social_media_kurir.ResponseEngageSocialMedia{
					Message: "Gagal, server sedang sibuk. Coba lagi lain waktu.",
				},
			}
		}

		go func(mesm sot_models.EntitySocialMedia, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()
			newCreatedEngagementSocialMediaKurir := mb_cud_serializer.NewJsonPayload().SetPayload(mesm).SetTableName(mesm.TableName()).SetRole(mb_cud_seeders.Kurir)
			if err := mb_cud_publisher.CreatePublish(konteks, publisher, newCreatedEngagementSocialMediaKurir); err != nil {
				fmt.Println("Gagal publish create social media kurir")
			}
		}(data.Data, cud_publisher)

		log.Printf("[INFO] Data social media berhasil ditambahkan untuk kurir ID %d", data.DataIdentitas.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusOK,
			Services: services,
			Payload: response_social_media_kurir.ResponseEngageSocialMedia{
				Message: "Data social media berhasil ditambahkan.",
			},
		}
	}

	if err_update := db.Write.WithContext(ctx).Model(sot_models.EntitySocialMedia{}).Where(sot_models.EntitySocialMedia{
		ID: id_sosmed_table,
	}).Updates(&sot_models.EntitySocialMedia{
		Whatsapp:  data.Data.Whatsapp,
		Facebook:  data.Data.Facebook,
		TikTok:    data.Data.TikTok,
		Instagram: data.Data.Instagram,
	}).Error; err_update != nil {
		log.Printf("[ERROR] Gagal memperbarui data social media untuk kurir ID %d: %v", data.DataIdentitas.IdKurir, err_update)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_social_media_kurir.ResponseEngageSocialMedia{
				Message: "Gagal, server sedang sibuk. Coba lagi lain waktu.",
			},
		}
	}

	go func(mesm sot_models.EntitySocialMedia, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()
		newUpdatedEngagementSocialMediaKurir := mb_cud_serializer.NewJsonPayload().SetPayload(mesm).SetTableName(mesm.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.UpdatePublish(konteks, publisher, newUpdatedEngagementSocialMediaKurir); err != nil {
			fmt.Println("Gagal publish update social media kurir")
		}
	}(data.Data, cud_publisher)

	log.Printf("[INFO] Data social media berhasil diperbarui untuk kurir ID %d", data.DataIdentitas.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_social_media_kurir.ResponseEngageSocialMedia{
			Message: "Data social media berhasil diperbarui.",
		},
	}
}
