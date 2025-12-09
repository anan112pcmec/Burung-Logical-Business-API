package pengguna_media_services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"

	data_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/data"
	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func UbahFotoProfilPengguna(ctx context.Context, data PayloadUbahFotoProfilPengguna, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "UbahFotoProfilPengguna"

	// Validasi identitas pengguna
	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	// Cek apakah data foto profil sudah ada
	var id_data_photo_exist int64 = 0
	if err := db.Read.WithContext(ctx).
		Model(&models.MediaPenggunaProfilFoto{}).
		Select("id").
		Where(&models.MediaPenggunaProfilFoto{
			IdPengguna: data.IdentitasPengguna.ID,
		}).
		Limit(1).
		Scan(&id_data_photo_exist).Error; err != nil {

		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	// Generate key baru
	keyz := models.MediaPenggunaProfilFoto{}.PathName() +
		strconv.Itoa(int(data.IdentitasPengguna.ID)) + "/" +
		helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	// Generate presigned URL
	url, err := ms.PresignedPutObject(
		ctx,
		data_cache.BucketFotoName,
		keyz,
		time.Minute*10,
	)
	if err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUploadUrl := url.String()

	// Kalau belum ada → insert
	if id_data_photo_exist == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaPenggunaProfilFoto{
			IdPengguna: data.IdentitasPengguna.ID,
			Key:        keyz,
			Format:     data.Ekstensi,
		}).Error; err != nil {

			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

	} else {

		var id_data_key_sama int64 = 0
		if err := db.Read.WithContext(ctx).
			Model(&models.MediaPenggunaProfilFoto{}).
			Select("id").
			Where(&models.MediaPenggunaProfilFoto{
				ID:  id_data_photo_exist,
				Key: keyz,
			}).
			Limit(1).
			Scan(&id_data_key_sama).Error; err != nil {

			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		if id_data_key_sama != 0 {
			return &response.ResponseMediaUpload{
				Status:   http.StatusConflict,
				Services: services,
			}
		}

		if err := db.Write.WithContext(ctx).
			Model(&models.MediaPenggunaProfilFoto{}).
			Where(&models.MediaPenggunaProfilFoto{
				ID: id_data_photo_exist,
			}).
			Update("key", keyz).Error; err != nil {

			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	}

	fmt.Println("keynyo", keyz)
	fmt.Println("urlnyo:", minIOUploadUrl)
	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}
}
