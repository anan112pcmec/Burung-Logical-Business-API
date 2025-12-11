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
	"github.com/anan112pcmec/Burung-backend-1/app/database/enums/media_ekstension"
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

	if !media_ekstension.PhotoValidExt[data.Ekstensi] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusBadRequest,
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

	if !media_ekstension.PhotoValidExt[data.Ekstensi] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusBadRequest,
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
		if !media_ekstension.PhotoValidExt[ext] {
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

	if !media_ekstension.PhotoValidExt[data.Ekstensi] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusBadRequest,
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
		if !media_ekstension.PhotoValidExt[data.Ekstensi[i]] {
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

func UbahBarangIndukVideo(ctx context.Context, data PayloadUbahVideoBarangInduk, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "BarangIndukVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

	var id_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       int32(data.IdBarangInduk),
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_barang_induk).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_barang_induk == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_data_barang_induk_video int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBarangIndukVideo{}).Select("id").Where(&models.MediaBarangIndukVideo{
		IdBarangInduk: id_barang_induk,
	}).Limit(1).Scan(&id_media_data_barang_induk_video).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBarangIndukVideo{}.PathName() + strconv.Itoa(int(id_barang_induk)) + "/" + helper.GenerateMediaKeyVideo() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketVideoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUploadUrl := url.String()
	if id_media_data_barang_induk_video == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaBarangIndukVideo{
			IdBarangInduk: data.IdBarangInduk,
			Key:           keyz,
			Format:        data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBarangIndukVideo{}).Where(&models.MediaBarangIndukVideo{
			ID: id_media_data_barang_induk_video,
		}).Updates(&models.MediaBarangIndukVideo{
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

func HapusBarangIndukVideo(ctx context.Context, data PayloadHapusVideoBarangInduk, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusBarangIndukVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_media_barang_induk_video int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBarangIndukVideo{}).Select("id").Where(&models.MediaBarangIndukVideo{
		ID:  data.IdMediaBarangIndukVideo,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_data_media_barang_induk_video).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_media_barang_induk_video == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal menemukan video",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBarangIndukVideo{}).Where(&models.MediaBarangIndukVideo{
		ID: data.IdMediaBarangIndukVideo,
	}).Delete(&models.MediaBarangIndukVideo{}).Error; err != nil {
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

func UbahKategoriBarangFoto(ctx context.Context, data PayloadUbahKategoriBarangFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "UbahKategoriBarangFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.PhotoValidExt[data.Ekstensi] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       int32(data.IdBarangInduk),
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang_induk).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_barang_induk == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_kategori_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
		ID:            data.IdKategoriBarang,
		IdBarangInduk: int32(data.IdBarangInduk),
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_kategori_barang_induk).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_kategori_barang_induk == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_kategori_barang_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaKategoriBarangFoto{}).Select("id").Where(&models.MediaKategoriBarangFoto{
		IdKategoriBarang: data.IdKategoriBarang,
		IdBarangInduk:    data.IdBarangInduk,
	}).Limit(1).Scan(&id_media_kategori_barang_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaKategoriBarangFoto{}.PathName() + strconv.Itoa(int(data.IdKategoriBarang)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, url_err := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
	if url_err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrlUpload := url.String()

	if id_media_kategori_barang_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaKategoriBarangFoto{
			IdKategoriBarang: data.IdKategoriBarang,
			IdBarangInduk:    data.IdBarangInduk,
			Key:              keyz,
			Format:           data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaKategoriBarangFoto{}).Where(&models.MediaKategoriBarangFoto{
			ID: id_media_kategori_barang_foto,
		}).Updates(&models.MediaKategoriBarangFoto{
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
		UrlUpload: minIOUrlUpload,
	}
}

func HapusKategoriBarangFoto(ctx context.Context, data PayloadHapusKategoriBarangFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusKategoriBarangFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_media_kategori_barang_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaKategoriBarangFoto{}).Select("id").Where(&models.MediaKategoriBarangFoto{
		ID:  data.IdMediaKategoriBarangFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_kategori_barang_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_kategori_barang_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaKategoriBarangFoto{}).Where(&models.MediaKategoriBarangFoto{
		ID: id_media_kategori_barang_foto,
	}).Delete(&models.MediaKategoriBarangFoto{}).Error; err != nil {
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

func TambahDistributorDataDokumen(ctx context.Context, data PayloadMediaDistributorDataDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahDistributorDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.DokumenValidExt[data.Ekstensi] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_distributor_data int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_distributor_data).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_distributor_data == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_distributor_data_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Select("id").Where(&models.MediaDistributorDataDokumen{
		IdDistributorData: id_distributor_data,
	}).Limit(1).Scan(&id_media_distributor_data_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaDistributorDataDokumen{}.PathName() + strconv.Itoa(int(data.IdDistributorData)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketDokumenName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_media_distributor_data_dokumen == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaDistributorDataDokumen{
			IdDistributorData: data.IdDistributorData,
			Key:               keyz,
			Format:            data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Where(&models.MediaDistributorDataDokumen{
			ID: id_distributor_data,
		}).Updates(&models.MediaDistributorDataDokumen{
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

func HapusMediaDistributorDataDokumen(ctx context.Context, data PayloadHapusMediaDistributorDataDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusKategoriBarangFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_distributor_data int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_distributor_data).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_distributor_data == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data distributor tidak ditemukan",
		}
	}

	var id_media_distributor_data_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Select("id").Where(&models.MediaDistributorDataDokumen{
		ID:  data.IdMediaDistributorDataDokumen,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_distributor_data_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_distributor_data_dokumen == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data dokumen tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Where(&models.MediaDistributorDataDokumen{
		ID: id_media_distributor_data_dokumen,
	}).Delete(&models.MediaDistributorDataDokumen{}).Error; err != nil {
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

func TambahMediaDistributorDataNPWPFoto(ctx context.Context, data PayloadTambahMediaDistributorDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaDistributorDataNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_distributor).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_distributor_data_npwp_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataNPWPFoto{}).Select("id").Where(&models.MediaDistributorDataNPWPFoto{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_npwp_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaDistributorDataNPWPFoto{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_npwp_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaDistributorDataNPWPFoto{
			IdDistributorData: id_data_distributor,
			Key:               keyz,
			Format:            data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNPWPFoto{}).Where(&models.MediaDistributorDataNPWPFoto{
			ID: id_media_distributor_data_npwp_foto,
		}).Updates(&models.MediaDistributorDataNPWPFoto{
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

func HapusMediaDistributorDataNPWPFoto(ctx context.Context, data PayloadHapusMediaDistributorDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaDistributorDataNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var id_data_media_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataNPWPFoto{}).Select("id").Where(&models.MediaDistributorDataNPWPFoto{
		ID: data.IdMediaDistributorDataNPWPFoto,
	}).Limit(1).Scan(&id_data_media_distributor).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_media_distributor == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNPWPFoto{}).Where(&models.MediaDistributorDataNPWPFoto{
		ID: id_data_media_distributor,
	}).Delete(&models.MediaDistributorDataNPWPFoto{}).Error; err != nil {
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

func TambahDistributorDataNIBFoto(ctx context.Context, data PayloadTambahDistributorDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahDistributorDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_distributor).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_distributor_data_nib_foto int64 = 0
	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Select("id").Where(&models.MediaDistributorDataNIBFoto{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_nib_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaDistributorDataNIBFoto{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_nib_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaDistributorDataNIBFoto{
			IdDistributorData: id_data_distributor,
			Key:               keyz,
			Format:            data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Where(&models.MediaDistributorDataNIBFoto{
			ID: id_media_distributor_data_nib_foto,
		}).Updates(&models.MediaDistributorDataNIBFoto{
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

func HapusDistributorDataNIBFoto(ctx context.Context, data PayloadHapusDistributorDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusDistributorDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var id_data_media_distributor_nib_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Select("id").Where(&models.MediaDistributorDataNIBFoto{
		ID:                data.IdMediaDistributorDataNIBFoto,
		IdDistributorData: data.IdDistributorData,
		Key:               data.KeyFoto,
	}).Limit(1).Scan(&id_data_media_distributor_nib_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_media_distributor_nib_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Foto Tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Where(&models.MediaDistributorDataNIBFoto{
		ID: id_data_media_distributor_nib_foto,
	}).Delete(&models.MediaDistributorDataNIBFoto{}).Error; err != nil {
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

func TambahDistributorDataSuratKerjasamaDokumen(ctx context.Context, data PayloadTambahDistributorDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahDistributorDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_distributor).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_distributor_data_surat_kerjasama_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Select("id").Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaDistributorDataSuratKerjasamaDokumen{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_surat_kerjasama_dokumen == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaDistributorDataSuratKerjasamaDokumen{
			IdDistributorData: id_data_distributor,
			Key:               keyz,
			Format:            data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
			ID: id_media_distributor_data_surat_kerjasama_dokumen,
		}).Updates(&models.MediaDistributorDataSuratKerjasamaDokumen{
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

func HapusDistributorDataSuratKerjasamaDataDokumen(ctx context.Context, data PayloadHapusDistributorDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusDistributorDataSuratKerjasamaDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
		ID:       data.IdDistributorData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_distributor).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_distributor == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data distributor tidak ditemukan",
		}
	}

	var id_media_distributor_data_surat_kerjasama_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Select("id").Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_distributor_data_surat_kerjasama_dokumen == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
		ID: id_media_distributor_data_surat_kerjasama_dokumen,
	}).Delete(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Error; err != nil {
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

func TambahBrandDataPerwakilanDokumen(ctx context.Context, data PayloadTambahBrandDataPerwakilanDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahBrandDataPerwakilanDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.DokumenValidExt[strings.ToLower(data.Ekstensi)] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_brand_data_perwakilan_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataPerwakilanDokumen{}).Select("id").Where(&models.MediaBrandDataPerwakilanDokumen{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_perwakilan_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataPerwakilanDokumen{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketDokumenName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_perwakilan_dokumen == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaBrandDataPerwakilanDokumen{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataPerwakilanDokumen{}).Where(&models.MediaBrandDataPerwakilanDokumen{
			ID: id_media_brand_data_perwakilan_dokumen,
		}).Updates(&models.MediaBrandDataPerwakilanDokumen{
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
		UrlUpload: url.String(),
		Key:       keyz,
	}
}

func HapusMediaBrandDataPerwakilanDokumen(ctx context.Context, data PayloadHapusBrandDataPerwakilanDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaBrandDataPerwakilanDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data brand tidak ditemukan",
		}
	}

	var id_media_brand_data_perwakilan_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataPerwakilanDokumen{}).Select("id").Where(&models.MediaBrandDataPerwakilanDokumen{
		ID:  data.IdMediaBrandDataPerwakilanDokumen,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&id_media_brand_data_perwakilan_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_brand_data_perwakilan_dokumen == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data dokumen tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataPerwakilanDokumen{}).Select("id").Where(&models.MediaBrandDataPerwakilanDokumen{
		ID: id_media_brand_data_perwakilan_dokumen,
	}).Delete(&models.MediaBrandDataPerwakilanDokumen{}).Error; err != nil {
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

func TambahMediaBrandDataSertifikatFoto(ctx context.Context, data PayloadTambahBrandDataSertifikatFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaBrandDataSertifikatFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.PhotoValidExt[strings.ToLower(data.Ekstensi)] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_brand_data_sertifikat_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataSertifikatFoto{}).Select("id").Where(&models.MediaBrandDataSertifikatFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_sertifikat_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataSertifikatFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_sertifikat_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaBrandDataSertifikatFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataSertifikatFoto{}).Where(&models.MediaBrandDataSertifikatFoto{
			ID: id_media_brand_data_sertifikat_foto,
		}).Updates(&models.MediaBrandDataSertifikatFoto{
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

func HapusMediaBrandDataSertifikatFoto(ctx context.Context, data PayloadHapusBrandDataSertifikatFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaBrandDataSertifikatFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data brand tidak ditemukan",
		}
	}

	var id_media_brand_data_sertifikat_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataSertifikatFoto{}).Select("id").Where(&models.MediaBrandDataSertifikatFoto{
		ID: data.IdMediaBrandDataSertifikatFoto,
	}).Limit(1).Scan(&id_media_brand_data_sertifikat_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_brand_data_sertifikat_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataSertifikatFoto{}).Where(&models.MediaBrandDataSertifikatFoto{
		ID: id_media_brand_data_sertifikat_foto,
	}).Delete(&models.MediaBrandDataSertifikatFoto{}).Error; err != nil {
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

func TambahMediaBrandDataNIBFoto(ctx context.Context, data PayloadTambahMediaBrandDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaBrandDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.PhotoValidExt[strings.ToLower(data.Ekstensi)] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_brand_data_nib_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataNIBFoto{}).Select("id").Where(&models.MediaBrandDataNIBFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_nib_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataNIBFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_nib_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaBrandDataNIBFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataNIBFoto{}).Where(&models.MediaBrandDataNIBFoto{
			ID: id_media_brand_data_nib_foto,
		}).Updates(&models.MediaBrandDataNIBFoto{
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

func HapusMediaBrandDataNIBFoto(ctx context.Context, data PayloadHapusMediaBrandDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaBrandDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data brand tidak ditemukan",
		}
	}

	var id_media_data_nib_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataNIBFoto{}).Select("id").Where(&models.MediaBrandDataNIBFoto{
		ID:          data.IdMediaBrandDataNIBFoto,
		IdBrandData: id_data_brand,
		Key:         data.KeyFoto,
	}).Limit(1).Scan(&id_media_data_nib_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_data_nib_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataNIBFoto{}).Where(&models.MediaBrandDataNIBFoto{
		ID: id_media_data_nib_foto,
	}).Delete(&models.MediaBrandDataNIBFoto{}).Error; err != nil {
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

func TambahMediaBrandNPWPFoto(ctx context.Context, data PayloadTambahMediaBrandDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaBrandNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.PhotoValidExt[strings.ToLower(data.Ekstensi)] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_brand_data_npwp_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataNPWPFoto{}).Select("id").Where(&models.MediaBrandDataNPWPFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_npwp_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataNPWPFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_npwp_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaBrandDataNPWPFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataNPWPFoto{}).Where(&models.MediaBrandDataNPWPFoto{
			ID: id_media_brand_data_npwp_foto,
		}).Updates(&models.MediaBrandDataNPWPFoto{
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

func HapusMediaBrandNPWPFoto(ctx context.Context, data PayloadHapusMediaBrandDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaBrandNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data brand tidak ditemukan",
		}
	}

	var id_media_brand_data_npwp_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataNPWPFoto{}).Select("id").Where(&models.MediaBrandDataNPWPFoto{
		ID: data.IdMediaBrandDataNPWPFoto,
	}).Limit(1).Scan(&id_media_brand_data_npwp_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_media_brand_data_npwp_foto == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataNPWPFoto{}).Where(&models.MediaBrandDataNPWPFoto{
		ID: id_media_brand_data_npwp_foto,
	}).Delete(&models.MediaBrandDataNPWPFoto{}).Error; err != nil {
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

func TambahMediaBrandDataLogoFoto(ctx context.Context, data PayloadTambahMediaBrandDataLogoFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahMediaBrandDataLogoBrandFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.PhotoValidExt[strings.ToLower(data.Ekstensi)] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_brand_data_logo_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataLogoFoto{}).Select("id").Where(&models.MediaBrandDataLogoFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_logo_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_logo_foto == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataLogoFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, data_cache.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_logo_foto == 0 {
		if err := db.Write.WithContext(ctx).Create(&models.MediaBrandDataLogoFoto{
			IdBrandData: data.IdBrandData,
			Key:         keyz,
			Format:      data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataLogoFoto{}).Where(&models.MediaBrandDataLogoFoto{
			ID: id_media_brand_data_logo_foto,
		}).Updates(&models.MediaBrandDataLogoFoto{
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

func HapusMediaBrandDataLogo(ctx context.Context, data PayloadHapusMediaBrandDataLogoFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseForm {
	services := "HapusMediaBrandDataLogo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data brand tidak ditemukan",
		}
	}

	var id_media_brand_data_logo int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataLogoFoto{}).Select("id").Where(&models.MediaBrandDataLogoFoto{
		ID:          data.IdMediaBrandDataLogoBrandFoto,
		IdBrandData: id_data_brand,
		Key:         data.KeyFoto,
	}).Limit(1).Scan(&id_media_brand_data_logo).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_logo == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataLogoFoto{}).Where(&models.MediaBrandDataLogoFoto{
		ID: id_media_brand_data_logo,
	}).Delete(&models.MediaBrandDataLogoFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahBrandDataSuratKerjasamaDokumen(ctx context.Context, data PayloadTambahBrandDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
	services := "TambahBrandDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if !media_ekstension.PhotoValidExt[strings.ToLower(data.Ekstensi)] {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BrandData{}).Select("id").Where(&models.BrandData{
		ID:       data.IdBrandData,
		SellerId: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_brand).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}
	if id_data_brand == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_brand_data_surat_kerjasama_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataSuratKerjasamaDokumen{}).Select("id").Where(&models.MediaBrandDataSuratKerjasamaFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataSuratKerjasamaDokumen{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	return &response.ResponseMediaUpload{
		Status:   http.StatusOK,
		Services: services,
	}
}
