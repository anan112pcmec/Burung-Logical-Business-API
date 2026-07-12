package pengguna_social_media_service

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"

)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Tambahkan Social Media
// Berfungsi Untuk menautkan atau melampirkan akun / social media mereka ke sistem burung
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EngageTautkanSocialMediaPengguna(ctx context.Context, data PayloadEngageTautkanSocialMedia, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahkanSocialMediaPenguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial pengguna tidak valid untuk ID %d", data.IdentitasPengguna.ID)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Kredensial pengguna tidak valid.",
		}
	}

	var id_sosmed_table int64 = 0
	_ = db.Read.WithContext(ctx).Model(&sot_models.EntitySocialMedia{}).
		Select("id").
		Where(&sot_models.EntitySocialMedia{
			EntityId:   data.IdentitasPengguna.ID,
			EntityType: entity_enums.Pengguna,
		}).Take(&id_sosmed_table)

	if id_sosmed_table == 0 {
		newTautkanSocialMedia := sot_models.EntitySocialMedia{
			EntityId:   data.IdentitasPengguna.ID,
			Whatsapp:   data.Data.Whatsapp,
			Facebook:   data.Data.Facebook,
			TikTok:     data.Data.TikTok,
			Instagram:  data.Data.Instagram,
			EntityType: entity_enums.Pengguna,
		}
		if err_buat_kolom := db.Write.WithContext(ctx).Create(&newTautkanSocialMedia).Error; err_buat_kolom != nil {
			log.Printf("[ERROR] Gagal membuat data social media untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_buat_kolom)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
			}
		}

		go func(Esm sot_models.EntitySocialMedia, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			newTautkanSocialMediaPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Esm).SetTableName(Esm.TableName()).SetRole(mb_cud_seeders.Pengguna)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newTautkanSocialMediaPublish); err != nil {
				fmt.Println("Gagal publish penautan social media baru ke message broker")
			}
		}(newTautkanSocialMedia, cud_publisher)

		log.Printf("[INFO] Data social media berhasil ditambahkan untuk pengguna ID %d", data.IdentitasPengguna.ID)
	} else {
		if data.Data.Whatsapp != "" && data.Data.Whatsapp != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&sot_models.EntitySocialMedia{}).
				Where(&sot_models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&sot_models.EntitySocialMedia{
					Whatsapp: data.Data.Whatsapp,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui Whatsapp untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
				}
			}
		}

		if data.Data.TikTok != "" && data.Data.TikTok != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&sot_models.EntitySocialMedia{}).
				Where(&sot_models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&sot_models.EntitySocialMedia{
					TikTok: data.Data.TikTok,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui TikTok untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
				}
			}
		}

		if data.Data.Facebook != "" && data.Data.Facebook != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&sot_models.EntitySocialMedia{}).
				Where(&sot_models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&sot_models.EntitySocialMedia{
					Facebook: data.Data.Facebook,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui Facebook untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
				}
			}
		}

		if data.Data.Instagram != "" && data.Data.Instagram != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&sot_models.EntitySocialMedia{}).
				Where(&sot_models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&sot_models.EntitySocialMedia{
					Instagram: data.Data.Instagram,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui Instagram untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
				}
			}
		}

		go func(idEntitySosmed int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var Esm sot_models.EntitySocialMedia
			if err := Read.WithContext(konteks).Model(&sot_models.EntitySocialMedia{}).Where(&sot_models.EntitySocialMedia{
				ID: idEntitySosmed,
			}).Limit(1).Take(&Esm).Error; err != nil {
				fmt.Println("Gagal mendapatkan data update entity social media")
				return
			}

			newTautkanSocialMediaPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Esm).SetTableName("EngageTautkanSocialMediaPengguna").SetRole(mb_cud_seeders.Pengguna)
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newTautkanSocialMediaPublish); err != nil {
				fmt.Println("Gagal publish update penautan social media ke message broker")
			}
		}(id_sosmed_table, db.Read, cud_publisher)
	}

	log.Printf("[INFO] Data social media berhasil diperbarui untuk pengguna ID %d", data.IdentitasPengguna.ID)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Data social media berhasil diperbarui.",
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Social Media
// Berfungsi Untuk hapus social media mereka yang terhubung ke sistem burung
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EngageHapusSocialMedia(ctx context.Context, data PayloadEngageHapusSocialMedia, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EngageHapusSocialMedia"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial pengguna tidak valid untuk ID %d", data.IdentitasPengguna.ID)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, kredensial pengguna tidak valid.",
		}
	}

	var kolom_update map[string]interface{}

	switch data.HapusSocialMedia {
	case "whatsapp":
		kolom_update = map[string]interface{}{"whatsapp": nil}
	case "facebook":
		kolom_update = map[string]interface{}{"facebook": nil}
	case "tiktok":
		kolom_update = map[string]interface{}{"tik_tok": nil}
	case "instagram":
		kolom_update = map[string]interface{}{"instagram": nil}
	default:
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Jenis social media tidak dikenal.",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&sot_models.EntitySocialMedia{}).
		Where(&sot_models.EntitySocialMedia{ID: data.IdSocialMedia}).
		Updates(kolom_update).Error; err != nil {
		log.Printf("[ERROR] Gagal menghapus data %s untuk pengguna ID %d: %v", data.HapusSocialMedia, data.IdentitasPengguna.ID, err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
		}
	}

	go func(idEntitySosmed int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var Esm sot_models.EntitySocialMedia
		if err := Read.WithContext(konteks).Model(&sot_models.EntitySocialMedia{}).Where(&sot_models.EntitySocialMedia{
			ID: idEntitySosmed,
		}).Limit(1).Take(&Esm).Error; err != nil {
			fmt.Println("Gagal mendapatkan data update entity social media")
			return
		}

		newTautkanSocialMediaPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Esm).SetTableName("EngageHapusSocialMedia").SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newTautkanSocialMediaPublish); err != nil {
			fmt.Println("Gagal publish update penautan social media ke message broker")
		}
	}(data.IdSocialMedia, db.Read, cud_publisher)

	log.Printf("[INFO] Data %s berhasil dihapus untuk pengguna ID %d", data.HapusSocialMedia, data.IdentitasPengguna.ID)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Data social media berhasil dihapus.",
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Follow Seler
// Berfungsi Untuk Memfollow sebuah seller
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func FollowSeller(ctx context.Context, data PayloadFollowOrUnfollowSeller, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "FollowSeller"

	_, status := data.IdentitasUser.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, kredensial user tidak valid.",
		}
	}

	var id_data_follower int64 = 0

	if err := db.Read.WithContext(ctx).Model(&sot_models.Follower{}).Select("id").
		Where(&sot_models.Follower{IdFollower: data.IdentitasUser.ID, IdFollowed: int64(data.IdSeller)}).
		Limit(1).Scan(&id_data_follower).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_follower == 0 {
		newFollow := sot_models.Follower{
			IdFollower: data.IdentitasUser.ID,
			IdFollowed: int64(data.IdSeller),
		}
		if err := db.Write.WithContext(ctx).Create(&newFollow).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal follow, silakan coba lagi lain waktu.",
			}
		}

		go func(Nf sot_models.Follower, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			thresholdPengguna := sot_threshold.PenggunaThreshold{
				IdPengguna: Nf.IdFollower,
			}

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: Nf.IdFollowed,
			}

			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.Following); err != nil {
				fmt.Println("Gagal incr count following ke threshold Pengguna")
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Follower); err != nil {
				fmt.Println("Gagal Incr count follower ke threshold Seller")
			}

			newFollowPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Nf).SetTableName(Nf.TableName()).SetRole(mb_cud_seeders.Pengguna)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newFollowPublish); err != nil {
				fmt.Println("Gagal publish new follow ke message broker")
			}

		}(newFollow, db.Write, cud_publisher)
	} else {
		// sudah follow
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal, kamu sudah follow seller tersebut.",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Unfollow seller
// Berfungsi Untuk unfollowe seller
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UnfollowSeller(ctx context.Context, data PayloadFollowOrUnfollowSeller, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "UnfollowSeller"

	_, status := data.IdentitasUser.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, kredensial user tidak valid.",
		}
	}

	var follower sot_models.Follower
	if err := db.Read.WithContext(ctx).Model(&sot_models.Follower{}).Where(&sot_models.Follower{
		IdFollower: data.IdentitasUser.ID,
		IdFollowed: int64(data.IdSeller),
	}).Limit(1).Scan(&follower).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal,server sedang sibuk coba lagi lain waktu",
		}
	}

	if follower.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	if result := db.Write.WithContext(ctx).Where(&sot_models.Follower{
		ID: follower.ID,
	}).Delete(&sot_models.Follower{}).Error; result != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal unfollow seller, coba lagi lain waktu.",
		}
	}

	go func(Nf sot_models.Follower, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: Nf.IdFollower,
		}

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: Nf.IdFollowed,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := thresholdPengguna.Decrement(konteks, Trh, stsk_pengguna.Following); err != nil {
			fmt.Println("Gagal decr count following ke threshold Pengguna")
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.Follower); err != nil {
			fmt.Println("Gagal decr count follower ke threshold Seller")
		}

		newFollowPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Nf).SetTableName(Nf.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newFollowPublish); err != nil {
			fmt.Println("Gagal publish delete follow ke message broker")
		}

	}(follower, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
