package pengguna_media_services

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/media_ekstension"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
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
		media_storage_database_seeders.BucketFotoName,
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

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}
}

func HapusFotoProfilPengguna(ctx context.Context, data PayloadHapusFotoProfilPengguna, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusFotoProfilPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var data_media_foto models.MediaPenggunaProfilFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaPenggunaProfilFoto{}).Where(&models.MediaPenggunaProfilFoto{
		ID:         data.IdMediaDataPengguna,
		IdPengguna: data.IdentitasPengguna.ID,
		Key:        data.KeyFoto,
	}).Limit(1).Scan(&data_media_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if data_media_foto.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaPenggunaProfilFoto{}).Where(&models.MediaPenggunaProfilFoto{
		ID: data.IdMediaDataPengguna,
	}).Delete(&models.MediaPenggunaProfilFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	// nanti push ke cassandra hystorical db

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
	}
}

func TambahMediaReviewFoto(ctx context.Context, data PayloadTambahMediaReviewFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaReviewFoto"
	const LimitPhoto = 5

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if len(data.Ekstensi) > LimitPhoto {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_review_produk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Review{}).Select("id").Where(&models.Review{
		ID:         data.IdReviewData,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_data_review_produk).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_review_produk == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_review_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaReviewFoto{}).Select("id").Where(&models.MediaReviewFoto{
		IdReview: id_data_review_produk,
	}).Limit(1).Scan(&id_media_review_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_review_foto != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	totalData := len(data.Ekstensi)
	var simpanDataFotoReview []models.MediaReviewFoto = make([]models.MediaReviewFoto, 0, totalData)
	var keyzAndUrl []response.UrlAndKey = make([]response.UrlAndKey, 0, totalData)

	for i := 0; i < totalData; i++ {
		if !media_ekstension.PhotoValidExt[data.Ekstensi[i]] {
			return &response.ResponseMediaUpload{
				Status:   http.StatusUnauthorized,
				Services: services,
			}
		}

		keyz := models.MediaReviewFoto{}.PathName() + strconv.Itoa(int(id_data_review_produk)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi[i]

		if url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2); err_url != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		} else {
			simpanDataFotoReview = append(simpanDataFotoReview, models.MediaReviewFoto{
				IdReview: id_data_review_produk,
				Key:      keyz,
				Format:   data.Ekstensi[i],
			})

			keyzAndUrl = append(keyzAndUrl, response.UrlAndKey{
				UrlUpload: url.String(),
				Key:       keyz,
			})
		}
	}

	if err := db.Write.WithContext(ctx).CreateInBatches(&simpanDataFotoReview, totalData).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlAndKey: keyzAndUrl,
	}
}

func TambahMediaReviewVideo(ctx context.Context, data PayloadTambahMediaReviewVideo, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaReviewVideo"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.VideoValistExt[data.Ekstensi] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_review_produk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Review{}).Select("id").Where(&models.Review{
		ID:         data.IdReviewData,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_data_review_produk).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_review_produk == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_review_video int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaReviewVideo{}).Select("id").Where(&models.MediaReviewVideo{
		IdReview: id_data_review_produk,
	}).Limit(1).Scan(&id_media_review_video).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_review_video != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	keyz := models.MediaReviewVideo{}.PathName() + strconv.Itoa(int(id_data_review_produk)) + "/" + helper.GenerateMediaKeyVideo() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketVideoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.MediaReviewVideo{
		IdReview: data.IdReviewData,
		Key:      keyz,
		Format:   data.Ekstensi,
	}).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}
