package kurir_media_services

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	pengiriman_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/pengiriman"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
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

	keyz := models.MediaKurirProfilFoto{}.PathName() + strconv.Itoa(int(data.IdentitasKurir.IdKurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
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

func TambahMediaInformasiKendaraanKurirKendaraanFoto(ctx context.Context, data PayloadTambahMediaInformasiKurirKendaraanFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaInformasiKendaraanKurirKendaraanFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_informasi_kendaraan_kurir int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		IDkurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kendaraan_kurir).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_informasi_kendaraan_kurir == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_informasi_kendaraan_kurir_kendaraan_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Select("id").Where(&models.MediaInformasiKendaraanKurirKendaraanFoto{
		IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
	}).Limit(1).Scan(&id_media_informasi_kendaraan_kurir_kendaraan_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaInformasiKendaraanKurirKendaraanFoto{}.PathName() + strconv.Itoa(int(id_data_informasi_kendaraan_kurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_informasi_kendaraan_kurir_kendaraan_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaInformasiKendaraanKurirKendaraanFoto{
			IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
			Key:                       keyz,
			Format:                    data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Where(&models.MediaInformasiKendaraanKurirKendaraanFoto{
			ID: id_media_informasi_kendaraan_kurir_kendaraan_foto,
		}).Updates(&models.MediaInformasiKendaraanKurirKendaraanFoto{
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
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaInformasiKendaraanKurirKendaraanFoto(ctx context.Context, data PayloadHapusMediaInformasiKurirKendaraanFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaInformasiKendaraanKurirKendaraanFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_media_informasi_kendaraan_kurir_kendaraan_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Select("id").Where(&models.MediaInformasiKendaraanKurirKendaraanFoto{
		ID:  data.IdMediaKendaraanKurirKendaraanFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_informasi_kendaraan_kurir_kendaraan_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_informasi_kendaraan_kurir_kendaraan_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Where(&models.MediaInformasiKendaraanKurirKendaraanFoto{
		ID: id_media_informasi_kendaraan_kurir_kendaraan_foto,
	}).Delete(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahInformasiKendaraanKurirBPKBFoto(ctx context.Context, data PayloadTambahMediaInformasiKendaraanKurirBPKBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahInformasiKendaraanKurirBPKBFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_informasi_kendaraan_kurir int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		IDkurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kendaraan_kurir).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_informasi_kendaraan_kurir == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_informasi_kendaraan_kurir_bpkb_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirBPKBFoto{}).Select("id").Where(&models.MediaInformasiKendaraanKurirBPKBFoto{
		IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
	}).Limit(1).Scan(&id_media_informasi_kendaraan_kurir_bpkb_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaInformasiKendaraanKurirBPKBFoto{}.PathName() + strconv.Itoa(int(id_data_informasi_kendaraan_kurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_informasi_kendaraan_kurir_bpkb_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaInformasiKendaraanKurirBPKBFoto{
			IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
			Key:                       keyz,
			Format:                    data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirBPKBFoto{}).Where(&models.MediaInformasiKendaraanKurirBPKBFoto{
			ID: id_media_informasi_kendaraan_kurir_bpkb_foto,
		}).Updates(&models.MediaInformasiKendaraanKurirBPKBFoto{
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
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusInformasiKendaraanKurirBPKBFoto(ctx context.Context, data PayloadHapusMediaInformasiKendaraanKurirBPKBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusInformasiKendaraanKurirBPKBFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_media_informasi_kendaraan_kurir_bpkb_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirBPKBFoto{}).Select("id").Where(&models.MediaInformasiKendaraanKurirBPKBFoto{
		ID:  data.IdMediaInformasiKendaraanKurirBPKBFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_informasi_kendaraan_kurir_bpkb_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_informasi_kendaraan_kurir_bpkb_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirBPKBFoto{}).Where(&models.MediaInformasiKendaraanKurirBPKBFoto{
		ID: id_media_informasi_kendaraan_kurir_bpkb_foto,
	}).Delete(&models.MediaInformasiKendaraanKurirBPKBFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahInformasiKendaraanKurirSTNKFoto(ctx context.Context, data PayloadTambahMediaInformasiKendaraanKurirSTNKFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahInformasiKendaraanKurirSTNKFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_informasi_kendaraan_kurir int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		IDkurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kendaraan_kurir).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_informasi_kendaraan_kurir == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_informasi_kendaraan_kurir_stnk_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirSTNKFoto{}).Select("id").Where(&models.MediaInformasiKendaraanKurirSTNKFoto{
		IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
	}).Limit(1).Scan(&id_media_informasi_kendaraan_kurir_stnk_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaInformasiKendaraanKurirSTNKFoto{}.PathName() + strconv.Itoa(int(id_data_informasi_kendaraan_kurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_informasi_kendaraan_kurir_stnk_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaInformasiKendaraanKurirSTNKFoto{
			IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
			Key:                       keyz,
			Format:                    data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirSTNKFoto{}).Where(&models.MediaInformasiKendaraanKurirSTNKFoto{
			ID: id_media_informasi_kendaraan_kurir_stnk_foto,
		}).Updates(&models.MediaInformasiKendaraanKurirSTNKFoto{
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
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusInformasiKendaraanKurirSTNKFoto(ctx context.Context, data PayloadHapusMediaInformasiKendaraanKurirSTNKFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusInformasiKendaraanKurirSTNKFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_media_informasi_kendaraan_kurir_stnk_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirSTNKFoto{}).Select("id").Where(&models.MediaInformasiKendaraanKurirSTNKFoto{
		ID:  data.IdMediaInformasiKendaraanKurirSTNKFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_informasi_kendaraan_kurir_stnk_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_informasi_kendaraan_kurir_stnk_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirSTNKFoto{}).Where(&models.MediaInformasiKendaraanKurirSTNKFoto{
		ID: id_media_informasi_kendaraan_kurir_stnk_foto,
	}).Delete(&models.MediaInformasiKendaraanKurirSTNKFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaInformasiKurirKTPFoto(ctx context.Context, data PayloadTambahMediaInformasiKurirKTPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaInformasiKurirKTPFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_informasi_kurir int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKurir{}).Select("id").Where(&models.InformasiKurir{
		IDkurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kurir).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_informasi_kurir == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_informasi_kurir_ktp_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKurirKTPFoto{}).Select("id").Where(&models.MediaInformasiKurirKTPFoto{
		IdInformasiKurir: id_data_informasi_kurir,
	}).Limit(1).Scan(&id_media_informasi_kurir_ktp_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaInformasiKurirKTPFoto{}.PathName() + strconv.Itoa(int(id_data_informasi_kurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_informasi_kurir_ktp_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaInformasiKurirKTPFoto{
			IdInformasiKurir: id_data_informasi_kurir,
			Key:              keyz,
			Format:           data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKurirKTPFoto{}).Where(&models.MediaInformasiKurirKTPFoto{
			ID: id_media_informasi_kurir_ktp_foto,
		}).Updates(&models.MediaInformasiKurirKTPFoto{
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
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaInformasiKurirKTPFoto(ctx context.Context, data PayloadHapusMediaInformasiKurirKTPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaInformasiKurirKTPFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var id_media_informasi_kurir_ktp_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKurirKTPFoto{}).Select("id").Where(&models.MediaInformasiKurirKTPFoto{
		ID:  data.IdMediaInformasiKurirKTPFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_informasi_kurir_ktp_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_informasi_kurir_ktp_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaInformasiKurirKTPFoto{}).Where(&models.MediaInformasiKurirKTPFoto{
		ID: id_media_informasi_kurir_ktp_foto,
	}).Delete(&models.MediaInformasiKurirKTPFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaPengirimanPickedUpFoto(ctx context.Context, data PayloadTambahPengirimanPickedUpFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaPengirimanPickedUpFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_pengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Pengiriman{}).Select("id").Where(&models.Pengiriman{
		ID:      data.IdPengiriman,
		IdKurir: &data.IdentitasKurir.IdKurir,
		Status:  pengiriman_enums.PickedUp,
	}).Limit(1).Scan(&id_data_pengiriman).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_pengiriman == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_pengiriman_picked_up_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaPengirimanPickedUpFoto{}).Select("id").Where(&models.MediaPengirimanPickedUpFoto{
		IdPengiriman: id_data_pengiriman,
	}).Limit(1).Scan(&id_media_pengiriman_picked_up_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_pengiriman_picked_up_foto != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	keyz := models.MediaPengirimanPickedUpFoto{}.PathName() + strconv.Itoa(int(id_data_pengiriman)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.MediaPengirimanPickedUpFoto{
		IdPengiriman: id_data_pengiriman,
		Key:          keyz,
		Format:       data.Ekstensi,
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

func TambahMediaPengirimanSampaiFoto(ctx context.Context, data PayloadTambahPengirimanSampaiFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaPengirimanSampaiFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_pengiriman int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Pengiriman{}).Select("id").Where(&models.Pengiriman{
		ID:      data.IdPengiriman,
		IdKurir: &data.IdentitasKurir.IdKurir,
		Status:  pengiriman_enums.Sampai,
	}).Limit(1).Scan(&id_data_pengiriman).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_pengiriman == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_pengiriman_sampai_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaPengirimanSampaiFoto{}).Select("id").Where(&models.MediaPengirimanSampaiFoto{
		IdPengiriman: id_data_pengiriman,
	}).Limit(1).Scan(&id_media_pengiriman_sampai_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_pengiriman_sampai_foto != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	keyz := models.MediaPengirimanSampaiFoto{}.PathName() + strconv.Itoa(int(id_data_pengiriman)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.MediaPengirimanSampaiFoto{
		IdPengiriman: id_data_pengiriman,
		Key:          keyz,
		Format:       data.Ekstensi,
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

func TambahMediaPengirimanEkspedisiPickedUpFoto(ctx context.Context, data PayloadTambahPengirimanEkspedisiPickedUpFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaPengirimanEkspedisiPickedUpFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_pengiriman_ekspedisi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.PengirimanEkspedisi{}).Select("id").Where(&models.PengirimanEkspedisi{
		ID:      data.IdPengirimanEkspedisi,
		IdKurir: &data.IdentitasKurir.IdKurir,
		Status:  pengiriman_enums.PickedUp,
	}).Limit(1).Scan(&id_data_pengiriman_ekspedisi).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_pengiriman_ekspedisi == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_pengiriman_ekspedisi_picked_up_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaPengirimanEkspedisiPickedUpFoto{}).Select("id").Where(&models.MediaPengirimanEkspedisiPickedUpFoto{
		IdPengirimanEkspedisi: id_data_pengiriman_ekspedisi,
	}).Limit(1).Scan(&id_media_pengiriman_ekspedisi_picked_up_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_pengiriman_ekspedisi_picked_up_foto != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	keyz := models.MediaPengirimanEkspedisiPickedUpFoto{}.PathName() + strconv.Itoa(int(id_data_pengiriman_ekspedisi)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.MediaPengirimanEkspedisiPickedUpFoto{
		IdPengirimanEkspedisi: id_data_pengiriman_ekspedisi,
		Key:                   keyz,
		Format:                data.Ekstensi,
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

func TambahMediaPengirimanEkspedisiSampaiAgentFoto(ctx context.Context, data PayloadTambahPengirimanEkspedisiPickedUpFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaPengirimanSampaiAgentFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_pengiriman_ekspedisi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.PengirimanEkspedisi{}).Select("id").Where(&models.PengirimanEkspedisi{
		ID:      data.IdPengirimanEkspedisi,
		IdKurir: &data.IdentitasKurir.IdKurir,
		Status:  pengiriman_enums.SampaiAgentEkspedisi,
	}).Limit(1).Scan(&id_data_pengiriman_ekspedisi).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_pengiriman_ekspedisi == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_pengiriman_ekspedisi_picked_up_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaPengirimanEkspedisiSampaiAgentFoto{}).Select("id").Where(&models.MediaPengirimanEkspedisiSampaiAgentFoto{
		IdPengirimanEkspedisi: id_data_pengiriman_ekspedisi,
	}).Limit(1).Scan(&id_media_pengiriman_ekspedisi_picked_up_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_pengiriman_ekspedisi_picked_up_foto != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	keyz := models.MediaPengirimanEkspedisiSampaiAgentFoto{}.PathName() + strconv.Itoa(int(id_data_pengiriman_ekspedisi)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.MediaPengirimanEkspedisiSampaiAgentFoto{
		IdPengirimanEkspedisi: id_data_pengiriman_ekspedisi,
		Key:                   keyz,
		Format:                data.Ekstensi,
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
