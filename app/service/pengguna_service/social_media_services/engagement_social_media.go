package pengguna_social_media_service

import (
	"context"
	"log"
	"net/http"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_social_media_pengguna "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/social_media_services/response_social_media_services"
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Tambahkan Social Media
// Berfungsi Untuk menautkan atau melampirkan akun / social media mereka ke sistem burung
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EngageTautkanSocialMediaPengguna(ctx context.Context, data PayloadEngageTautkanSocialMedia, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahkanSocialMediaPenguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		log.Printf("[WARN] Kredensial pengguna tidak valid untuk ID %d", data.IdentitasPengguna.ID)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
				Message: "Kredensial pengguna tidak valid.",
			},
		}
	}

	var id_sosmed_table int64 = 0
	_ = db.Read.WithContext(ctx).Model(&models.EntitySocialMedia{}).
		Select("id").
		Where(&models.EntitySocialMedia{
			EntityId:   data.IdentitasPengguna.ID,
			EntityType: entity_enums.Pengguna,
		}).Take(&id_sosmed_table)

	if id_sosmed_table == 0 {
		if err_buat_kolom := db.Write.WithContext(ctx).Create(&models.EntitySocialMedia{
			EntityId:   data.IdentitasPengguna.ID,
			Whatsapp:   data.Data.Whatsapp,
			Facebook:   data.Data.Facebook,
			TikTok:     data.Data.TikTok,
			Instagram:  data.Data.Instagram,
			EntityType: entity_enums.Pengguna,
		}).Error; err_buat_kolom != nil {
			log.Printf("[ERROR] Gagal membuat data social media untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_buat_kolom)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
					Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
				},
			}
		}

		log.Printf("[INFO] Data social media berhasil ditambahkan untuk pengguna ID %d", data.IdentitasPengguna.ID)
	} else {
		if data.Data.Whatsapp != "" && data.Data.Whatsapp != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&models.EntitySocialMedia{}).
				Where(&models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&models.EntitySocialMedia{
					Whatsapp: data.Data.Whatsapp,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui Whatsapp untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
						Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
					},
				}
			}
		}

		if data.Data.TikTok != "" && data.Data.TikTok != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&models.EntitySocialMedia{}).
				Where(&models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&models.EntitySocialMedia{
					TikTok: data.Data.TikTok,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui TikTok untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
						Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
					},
				}
			}
		}

		if data.Data.Facebook != "" && data.Data.Facebook != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&models.EntitySocialMedia{}).
				Where(&models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&models.EntitySocialMedia{
					Facebook: data.Data.Facebook,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui Facebook untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
						Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
					},
				}
			}
		}

		if data.Data.Instagram != "" && data.Data.Instagram != "not" {
			if err_update := db.Write.WithContext(ctx).Model(&models.EntitySocialMedia{}).
				Where(&models.EntitySocialMedia{ID: id_sosmed_table}).
				Updates(&models.EntitySocialMedia{
					Instagram: data.Data.Instagram,
				}).Error; err_update != nil {
				log.Printf("[ERROR] Gagal memperbarui Instagram untuk pengguna ID %d: %v", data.IdentitasPengguna.ID, err_update)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
						Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
					},
				}
			}
		}
	}

	log.Printf("[INFO] Data social media berhasil diperbarui untuk pengguna ID %d", data.IdentitasPengguna.ID)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_social_media_pengguna.ResponseEngageSocialMedia{
			Message: "Data social media berhasil diperbarui.",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Social Media
// Berfungsi Untuk hapus social media mereka yang terhubung ke sistem burung
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EngageHapusSocialMedia(ctx context.Context, data PayloadEngageHapusSocialMedia, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EngageHapusSocialMedia"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		log.Printf("[WARN] Kredensial pengguna tidak valid untuk ID %d", data.IdentitasPengguna.ID)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_social_media_pengguna.ResponseEngageHapusSocialMedia{
				Message: "Gagal, kredensial pengguna tidak valid.",
			},
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
			Payload: response_social_media_pengguna.ResponseEngageHapusSocialMedia{
				Message: "Jenis social media tidak dikenal.",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.EntitySocialMedia{}).
		Where(&models.EntitySocialMedia{ID: data.IdSocialMedia}).
		Updates(kolom_update).Error; err != nil {
		log.Printf("[ERROR] Gagal menghapus data %s untuk pengguna ID %d: %v", data.HapusSocialMedia, data.IdentitasPengguna.ID, err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_social_media_pengguna.ResponseEngageHapusSocialMedia{
				Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
			},
		}
	}

	log.Printf("[INFO] Data %s berhasil dihapus untuk pengguna ID %d", data.HapusSocialMedia, data.IdentitasPengguna.ID)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_social_media_pengguna.ResponseEngageHapusSocialMedia{
			Message: "Data social media berhasil dihapus.",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Follow Seler
// Berfungsi Untuk Memfollow sebuah seller
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func FollowSeller(ctx context.Context, data PayloadFollowOrUnfollowSeller, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "FollowSeller"

	_, status := data.IdentitasUser.Validating(ctx, db.Read)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_social_media_pengguna.ResponseFollowSeller{
				Message: "Gagal, kredensial user tidak valid.",
			},
		}
	}

	var id_data_follower int64 = 0

	if err := db.Read.WithContext(ctx).Model(&models.Follower{}).Select("id").
		Where(&models.Follower{IdFollower: data.IdentitasUser.ID, IdFollowed: int64(data.IdSeller)}).
		Limit(1).Scan(&id_data_follower).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_social_media_pengguna.ResponseFollowSeller{
				Message: "Gagal Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_follower == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.Follower{
			IdFollower: data.IdentitasUser.ID,
			IdFollowed: int64(data.IdSeller),
		}).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Payload: response_social_media_pengguna.ResponseFollowSeller{
					Message: "Gagal follow, silakan coba lagi lain waktu.",
				},
			}
		}
	} else {
		// sudah follow
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Payload: response_social_media_pengguna.ResponseFollowSeller{
				Message: "Gagal, kamu sudah follow seller tersebut.",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_social_media_pengguna.ResponseFollowSeller{
			Message: "Berhasil",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Unfollow seller
// Berfungsi Untuk unfollowe seller
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UnfollowSeller(ctx context.Context, data PayloadFollowOrUnfollowSeller, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "UnfollowSeller"

	_, status := data.IdentitasUser.Validating(ctx, db.Read)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_social_media_pengguna.ResponseUnfollowSeller{
				Message: "Gagal, kredensial user tidak valid.",
			},
		}
	}

	var id_follower int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Follower{}).Select("id").Where(&models.Follower{
		IdFollower: data.IdentitasUser.ID,
		IdFollowed: int64(data.IdSeller),
	}).Limit(1).Scan(&id_follower).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_social_media_pengguna.ResponseUnfollowSeller{
				Message: "Gagal,server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_follower == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_social_media_pengguna.ResponseUnfollowSeller{
				Message: "Gagal data tidak ditemukan",
			},
		}
	}

	if result := db.Write.WithContext(ctx).Where(&models.Follower{
		ID: id_follower,
	}).Delete(&models.Follower{}).Error; result != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_social_media_pengguna.ResponseUnfollowSeller{
				Message: "Gagal unfollow seller, coba lagi lain waktu.",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_social_media_pengguna.ResponseUnfollowSeller{
			Message: "Berhasil",
		},
	}
}
