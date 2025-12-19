package pengguna_media_services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/media_ekstension"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_review "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/review"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func UbahFotoProfilPengguna(ctx context.Context, data PayloadUbahFotoProfilPengguna, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahFotoProfilPengguna"

	// Validasi identitas pengguna
	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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
		newPhoto := models.MediaPenggunaProfilFoto{
			IdPengguna: data.IdentitasPengguna.ID,
			Key:        keyz,
			Format:     data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newPhoto).Error; err != nil {

			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mppf models.MediaPenggunaProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			thresholdPengguna := sot_threshold.PenggunaThreshold{
				IdPengguna: Mppf.IdPengguna,
			}

			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.MediaPenggunaProfilFoto); err != nil {
				fmt.Println("Gagal incr count foto profil pengguna ke threshold pengguna")
			}

			createPhotoPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mppf).SetTableName(Mppf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createPhotoPublish); err != nil {
				fmt.Println("Gagal publish create photo profil pengguna ke message broker")
			}
		}(newPhoto, db.Write, cud_publisher)

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

		go func(idMppf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataPhotoProfil models.MediaPenggunaProfilFoto
			if err := Read.WithContext(konteks).Model(&models.MediaPenggunaProfilFoto{}).Where(&models.MediaPenggunaProfilFoto{
				ID: idMppf,
			}).Limit(1).Take(&dataPhotoProfil).Error; err != nil {
				fmt.Println("Gagal mendapatkan data photo profil")
				return
			}

			updatePhotoProfilPenggunaPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPhotoProfil).SetTableName(dataPhotoProfil.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatePhotoProfilPenggunaPublish); err != nil {
				fmt.Println("Gagal publish update photo profil pengguna ke message broker")
			}
		}(id_data_photo_exist, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}
}

func HapusFotoProfilPengguna(ctx context.Context, data PayloadHapusFotoProfilPengguna, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusFotoProfilPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	go func(Mppf models.MediaPenggunaProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: Mppf.IdPengguna,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdPengguna.Decrement(konteks, Trh, stsk_pengguna.MediaPenggunaProfilFoto); err != nil {
			fmt.Println("Gagal decrement count photo profil pengguna ke pengguna threshold")
		}

		deletePhotoProfilPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mppf).SetTableName(Mppf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deletePhotoProfilPublish); err != nil {
			fmt.Println("Gagal publish delete foto profil pengguna ke message broker")
		}
	}(data_media_foto, db.Write, cud_publisher)

	// nanti push ke cassandra hystorical db

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
	}
}

func TambahMediaReviewFoto(ctx context.Context, data PayloadTambahMediaReviewFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaReviewFoto"
	const LimitPhoto = 5

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	go func(mediaPhotos []models.MediaReviewFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdReview := sot_threshold.ReviewThreshold{
			IdReview: mediaPhotos[0].IdReview,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		customIncr := sot_threshold.CustomCounter{
			FieldName: stsk_review.MediaReviewFoto,
			Count:     len(mediaPhotos),
		}

		if err := thresholdReview.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{customIncr}); err != nil {
			fmt.Println("gagal increment review foto ke threhsold review")
		}

		for _, mP := range mediaPhotos {
			go func(Photo models.MediaReviewFoto, publisherr *mb_cud_publisher.Publisher) {
				ctx_tt := context.Background()
				kontekss, cancell := context.WithTimeout(ctx_tt, time.Second*5)
				defer cancell()

				createReviewFotoPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Photo).SetTableName(Photo.TableName())
				if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](kontekss, publisherr, createReviewFotoPublish); err != nil {
					fmt.Println("Gagal publish create photo ke message broker")
				}
			}(mP, publisher)
		}

	}(simpanDataFotoReview, db.Write, cud_publisher)

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlAndKey: keyzAndUrl,
	}
}

func TambahMediaReviewVideo(ctx context.Context, data PayloadTambahMediaReviewVideo, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaReviewVideo"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	createVideoReview := models.MediaReviewVideo{
		IdReview: data.IdReviewData,
		Key:      keyz,
		Format:   data.Ekstensi,
	}

	if err := db.Write.WithContext(ctx).Create(&createVideoReview).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	go func(Vr models.MediaReviewVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		reviewThreshold := sot_threshold.ReviewThreshold{
			IdReview: Vr.IdReview,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := reviewThreshold.Increment(konteks, Trh, stsk_review.MediaReviewVideo); err != nil {
			fmt.Println("Gagal increment review video counter ke threshold review")
		}

		createVideoReviewPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Vr).SetTableName(Vr.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createVideoReviewPublish); err != nil {
			fmt.Println("Gagal publish create video review ke message broker")
		}
	}(createVideoReview, db.Write, cud_publisher)

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}
