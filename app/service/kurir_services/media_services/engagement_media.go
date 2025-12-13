package kurir_media_services

import (
	"context"
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

	keyz := models.MediaKurirProfilFoto{}.PathName() + strconv.Itoa(int(data.IdentitasKurir.IdKurir)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

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

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

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

	if err := ms.RemoveObject(ctx, data_cache.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
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

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

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

	if err := ms.RemoveObject(ctx, data_cache.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
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

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

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

	if err := ms.RemoveObject(ctx, data_cache.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
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
