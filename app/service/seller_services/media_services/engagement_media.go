package seller_media_services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"

	data_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/data"
	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/enums/media_foto_ekstensi"
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
			Updates(&models.MediaSellerProfilFoto{
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

func UbahFotoBannerSeller(ctx context.Context, data PayloadUbahFotoBannerSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "UbahFotoBannerSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var IdDataBanner int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaSellerBannerFoto{}).Select("id").Where(&models.MediaSellerBannerFoto{
		IdSeller: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&IdDataBanner).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaSellerBannerFoto{}.PathName() +
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

	if IdDataBanner == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaSellerBannerFoto{
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
		if err := db.Write.WithContext(ctx).Model(&models.MediaSellerBannerFoto{}).Where(&models.MediaSellerBannerFoto{
			ID: IdDataBanner,
		}).Updates(&models.MediaSellerBannerFoto{
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
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}

}

func HapusFotoBannerSeller(ctx context.Context, data PayloadHapusFotoBannerSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusFotoBannerSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var IdDataMediaFoto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaSellerBannerFoto{}).Select("id").Where(&models.MediaSellerBannerFoto{
		ID:       data.IdMediaBannerSellerFoto,
		IdSeller: int64(data.IdentitasSeller.IdSeller),
		Key:      data.KeyFoto,
	}).Limit(1).Scan(&IdDataMediaFoto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if IdDataMediaFoto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemkan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaSellerBannerFoto{}).Where(&models.MediaSellerBannerFoto{
		ID: data.IdMediaBannerSellerFoto,
	}).Delete(&models.MediaSellerBannerFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal menghapus coba lagi",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahkanFotoTokoFisikSeller(ctx context.Context, data PayloadTambahkanFotoTokoFisikSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUploadBurst {
	services := "TambahkanFotoTokoFisikSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	totalData := len(data.Ekstensi)

	if totalData > 20 {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusRequestEntityTooLarge,
			Services: services,
		}
	}

	var DataFotoSave []models.MediaSellerTokoFisikFoto = make([]models.MediaSellerTokoFisikFoto, 0, totalData)
	var DataUrlAndKeyResponse []response.UrlAndKey = make([]response.UrlAndKey, 0, totalData)

	for i := 0; i < totalData; i++ {
		ext := strings.ToLower(data.Ekstensi[i])
		if !media_foto_ekstensi.ValidExt[ext] {
			return &response.ResponseMediaUploadBurst{
				Status:   http.StatusBadRequest,
				Services: services,
			}
		}

		keyz := models.MediaSellerTokoFisikFoto{}.PathName() +
			strconv.Itoa(int(data.IdentitasSeller.IdSeller)) + "/" +
			helper.GenerateMediaKeyPhoto() + "." + ext

		url, errPPO := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*10)
		if errPPO != nil {
			return &response.ResponseMediaUploadBurst{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		DataFotoSave = append(DataFotoSave, models.MediaSellerTokoFisikFoto{
			IdSeller: data.IdentitasSeller.IdSeller,
			Key:      keyz,
			Format:   ext,
		})

		DataUrlAndKeyResponse = append(DataUrlAndKeyResponse, response.UrlAndKey{
			UrlUpload: url.String(),
			Key:       keyz,
		})
	}

	if err := db.Write.WithContext(ctx).CreateInBatches(&DataFotoSave, totalData).Error; err != nil {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	return &response.ResponseMediaUploadBurst{
		Status:    http.StatusOK,
		Services:  services,
		UrlAndKey: DataUrlAndKeyResponse,
	}
}

func HapusFotoTokoFisikSeller(ctx context.Context, data PayloadHapusFotoTokoFisikSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusFotoTokoFisikSeller"
	var wg sync.WaitGroup
	var mu sync.RWMutex

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	dataTotal := len(data.DataMediaFotoTokoFisik)
	errChan := make(chan error, dataTotal)
	var idsHapus []int64 = make([]int64, 0, dataTotal)
	for i := 0; i < dataTotal; i++ {
		wg.Add(1)
		go func(angka int) {
			defer wg.Done()
			var id_data_media_foto_toko_fisik int64 = 0
			if err := db.Read.WithContext(ctx).Select("id").Model(&models.MediaSellerTokoFisikFoto{}).Where(&models.MediaSellerTokoFisikFoto{
				ID:  data.DataMediaFotoTokoFisik[angka].IdMedia,
				Key: data.DataMediaFotoTokoFisik[angka].KeyMedia,
			}).Limit(1).Scan(&id_data_media_foto_toko_fisik).Error; err != nil {
				errChan <- fmt.Errorf("gagal")
				return
			}

			if id_data_media_foto_toko_fisik == 0 {
				errChan <- fmt.Errorf("gagal")
				return
			}

			ctxRemove, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			if err := ms.RemoveObject(ctxRemove, data_cache.BucketFotoName, data.DataMediaFotoTokoFisik[angka].KeyMedia, minio.RemoveObjectOptions{
				ForceDelete: true,
			}); err != nil {
				errChan <- fmt.Errorf("gagal")
				return
			}

			mu.Lock()
			idsHapus = append(idsHapus, id_data_media_foto_toko_fisik)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal data tidak valid",
			}
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaSellerTokoFisikFoto{}).Where("id IN ?", idsHapus).Delete(&models.MediaSellerTokoFisikFoto{}).Error; err != nil {
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

func UbahFotoEtalaseSeller(ctx context.Context, data PayloadUbahFotoEtalase, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "UbahFotoEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_etalase int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Etalase{}).Select("id").Where(&models.Etalase{
		ID:       data.IdEtalase,
		SellerID: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Take(&id_data_etalase).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_media_etalase_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Select("id").Where(&models.MediaEtalaseFoto{
		IdEtalase: id_data_etalase,
	}).Limit(1).Scan(id_data_media_etalase_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaEtalaseFoto{}.PathName() + strconv.Itoa(int(id_data_etalase)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrl := url.String()

	if id_data_media_etalase_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaEtalaseFoto{
			IdEtalase: id_data_etalase,
			Key:       keyz,
			Format:    data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Where(&models.MediaEtalaseFoto{
			ID: id_data_etalase,
		}).Updates(&models.MediaEtalaseFoto{
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

func HapusFotoEtalaseSeller(ctx context.Context, data PayloadHapusFotoEtalase, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusFotoEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status: http.StatusNotFound,
		}
	}

	var id_data_media_etalase_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Select("id").Where(&models.MediaEtalaseFoto{
		ID:  data.IdMediaEtalaseFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_data_media_etalase_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_media_etalase_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data media foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Where(&models.MediaEtalaseFoto{
		ID: id_data_media_etalase_foto,
	}).Delete(&models.MediaEtalaseFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk, coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahkanMediaBarangIndukFoto(ctx context.Context, data PayloadTambahBarangIndukFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUploadBurst {
	services := "TambahkanMediaBarangIndukFoto"
	const Limit uint8 = 10

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       data.IdBarangInduk,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang_induk).Error; err != nil {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_barang_induk == 0 {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	dataLength := len(data.Ekstensi)

	if dataLength > 10 {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var DataMediaBarangIndukFoto []models.MediaBarangIndukFoto = make([]models.MediaBarangIndukFoto, 0, dataLength)
	var DataUrlAndKeyUpload []response.UrlAndKey = make([]response.UrlAndKey, 0, dataLength)

	var idsDataMediaBarangInduk []int64
	if err := db.Read.WithContext(ctx).Model(&models.MediaBarangIndukFoto{}).Select("id").Where(&models.MediaBarangIndukFoto{
		IdBarangInduk: int64(data.IdBarangInduk),
	}).Limit(int(Limit)).Scan(&idsDataMediaBarangInduk).Error; err != nil {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	ExistPhoto := len(idsDataMediaBarangInduk)
	if ExistPhoto >= int(Limit) {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	uploadCount := dataLength
	available := int(Limit) - ExistPhoto
	if uploadCount > available {
		uploadCount = available
	}

	for i := 0; i < uploadCount; i++ {
		if !media_foto_ekstensi.ValidExt[data.Ekstensi[i]] {
			return &response.ResponseMediaUploadBurst{
				Status:   http.StatusBadRequest,
				Services: services,
			}
		}

		keyz := models.MediaBarangIndukFoto{}.PathName() + strconv.Itoa(int(id_data_barang_induk)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi[i]

		url, err := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
		if err != nil {
			return &response.ResponseMediaUploadBurst{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		DataMediaBarangIndukFoto = append(DataMediaBarangIndukFoto, models.MediaBarangIndukFoto{
			IdBarangInduk: id_data_barang_induk,
			Key:           keyz,
			Format:        data.Ekstensi[i],
		})

		DataUrlAndKeyUpload = append(DataUrlAndKeyUpload, response.UrlAndKey{
			Key:       keyz,
			UrlUpload: url.String(),
		})
	}

	if err := db.Write.WithContext(ctx).CreateInBatches(&DataMediaBarangIndukFoto, uploadCount).Error; err != nil {
		return &response.ResponseMediaUploadBurst{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	return &response.ResponseMediaUploadBurst{
		Status:    http.StatusOK,
		Services:  services,
		UrlAndKey: DataUrlAndKeyUpload,
	}
}

func HapusMediaBarangIndukFoto(ctx context.Context, data PayloadHapusBarangIndukFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaBarangIndukFoto"
	var wg sync.WaitGroup
	var mu sync.RWMutex

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var dataLength = len(data.DataMediaBarangIndukFoto)

	var idsDataMediaBarangIndukFoto []int64 = make([]int64, 0, dataLength)
	for i := range dataLength {
		wg.Add(1)
		go func(a int) {
			defer wg.Done()
			var id_data_media_barang_induk_foto int64 = 0
			if err := db.Read.WithContext(ctx).Model(&models.MediaBarangIndukFoto{}).Select("id").Where(&models.MediaBarangIndukFoto{
				ID:  data.DataMediaBarangIndukFoto[a].IdMedia,
				Key: data.DataMediaBarangIndukFoto[a].KeyMedia,
			}).Limit(1).Scan(&id_data_media_barang_induk_foto).Error; err != nil {
				return
			}

			mu.Lock()
			idsDataMediaBarangIndukFoto = append(idsDataMediaBarangIndukFoto, id_data_media_barang_induk_foto)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if err := db.Write.WithContext(ctx).Model(&models.MediaBarangIndukFoto{}).Where("id IN ?", idsDataMediaBarangIndukFoto).Delete(&models.MediaBarangIndukFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusInternalServerError,
		Services: services,
		Message:  "Berhasil",
	}
}
