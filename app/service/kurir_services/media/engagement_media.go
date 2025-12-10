package kurir_media_services

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

func UbahKurirProfilFoto(ctx context.Context, data PayloadUbahKurirProfilFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "UbahKurirProfilFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_media_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaKurirProfilFoto{}).Select("id").Where(&models.MediaKurirProfilFoto{
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_media_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := fmt.Sprintf(models.MediaKurirProfilFoto{}.PathName() + strconv.Itoa(int(data.IdentitasKurir.IdKurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi)

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrl := url.String()

	if id_data_media_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaKurirProfilFoto{
			IdKurir: data.IdentitasKurir.IdKurir,
			Key:     keyz,
			Format:  data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaKurirProfilFoto{}).Where(&models.MediaKurirProfilFoto{
			ID: id_data_media_foto,
		}).Updates(&models.MediaKurirProfilFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUrl,
		Key:       keyz,
	}
}
