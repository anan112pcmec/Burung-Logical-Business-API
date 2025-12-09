package seller_media_services

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

func UbahFotoProfilSeller(ctx context.Context, data PayloadUbahFotoProfilSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "UbahFotoProfilSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_profil_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaSellerProfilFoto{}).Select("id").Where(&models.MediaSellerProfilFoto{
		IdSeller: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&id_data_profil_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaSellerProfilFoto{}.PathName() +
		strconv.Itoa(int(data.IdentitasSeller.IdSeller)) + "/" +
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

	if id_data_profil_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaSellerProfilFoto{
			IdSeller: int64(data.IdentitasSeller.IdSeller),
			Key:      keyz,
			Format:   data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).
			Model(&models.MediaSellerProfilFoto{}).
			Where(&models.MediaSellerProfilFoto{
				ID: id_data_profil_foto,
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

func HapusFotoProfilSeller(ctx context.Context, data PayloadHapusFotoProfilSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusFotoProfilSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_foto_profil_seller models.MediaSellerProfilFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaSellerProfilFoto{}).Where(&models.MediaSellerProfilFoto{
		ID:  data.IdMediaSeller,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_foto_profil_seller).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_foto_profil_seller.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaSellerProfilFoto{}).Where(&models.MediaSellerProfilFoto{
		ID: data.IdMediaSeller,
	}).Delete(&models.MediaSellerProfilFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
	}
}
