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
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/media_ekstension"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_brand_data "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/brand_data"
	stsk_distributor_data "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/distributor_data"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	stsk_transaksi "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func UbahFotoProfilSeller(ctx context.Context, data PayloadUbahFotoProfilSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahFotoProfilSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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
		media_storage_database_seeders.BucketFotoName,
		keyz,
		time.Minute*10,
	)
	if err != nil {
		fmt.Println("Kena disini")
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUploadUrl := url.String()

	if id_data_profil_foto == 0 {
		newPhotoProfil := models.MediaSellerProfilFoto{
			IdSeller: int64(data.IdentitasSeller.IdSeller),
			Key:      keyz,
			Format:   data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newPhotoProfil).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Pp models.MediaSellerProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: Pp.IdSeller,
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.MediaSellerFotoProfil); err != nil {
				fmt.Println("Gagal incr count media seller profil foto ke threshold seller")
			}

			photoProfilCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Pp).SetTableName(Pp.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, photoProfilCreatePublish); err != nil {
				fmt.Println("Gagal publish profil foto create ke message broker")
			}
		}(newPhotoProfil, db.Write, cud_publisher)
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

		go func(IdPp int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataPhotoProfilUpdated models.MediaSellerProfilFoto
			if err := Read.WithContext(konteks).Model(&models.MediaSellerProfilFoto{}).Where(&models.MediaSellerProfilFoto{
				ID: IdPp,
			}).Limit(1).Take(&dataPhotoProfilUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data photo profil seller")
			}

			photoProfilUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPhotoProfilUpdated).SetTableName(dataPhotoProfilUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, photoProfilUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update photo profil seller ke message broker")
			}
		}(id_data_profil_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}
}

func HapusFotoProfilSeller(ctx context.Context, data PayloadHapusFotoProfilSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusFotoProfilSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	go func(Dmfps models.MediaSellerProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: Dmfps.IdSeller,
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.MediaSellerFotoProfil); err != nil {
			fmt.Println("Gagal decr photo profil foto ke treshold selller")
		}

		profilPhotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmfps).SetTableName(Dmfps.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, profilPhotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete profil foto seller ke message broker")
		}
	}(data_media_foto_profil_seller, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
	}
}

func UbahFotoBannerSeller(ctx context.Context, data PayloadUbahFotoBannerSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahFotoBannerSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	if IdDataBanner == 0 {
		newBannerFoto := models.MediaSellerBannerFoto{
			IdSeller: int64(data.IdentitasSeller.IdSeller),
			Key:      keyz,
			Format:   data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newBannerFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Bf models.MediaSellerBannerFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: Bf.IdSeller,
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.MediaSellerBannerFoto); err != nil {
				fmt.Println("Gagal incr count media seller banner foto ke threshold seller")
			}

			bannerFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bf).SetTableName(Bf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bannerFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish banner foto create ke message broker")
			}
		}(newBannerFoto, db.Write, cud_publisher)
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

		go func(IdBf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBannerFotoUpdated models.MediaSellerBannerFoto
			if err := Read.WithContext(konteks).Model(&models.MediaSellerBannerFoto{}).Where(&models.MediaSellerBannerFoto{
				ID: IdBf,
			}).Limit(1).Take(&dataBannerFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data banner foto seller")
			}

			bannerFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBannerFotoUpdated).SetTableName(dataBannerFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bannerFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update banner foto seller ke message broker")
			}
		}(IdDataBanner, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}
}

func HapusFotoBannerSeller(ctx context.Context, data PayloadHapusFotoBannerSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusFotoBannerSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_banner_foto_seller models.MediaSellerBannerFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaSellerBannerFoto{}).Where(&models.MediaSellerBannerFoto{
		ID:       data.IdMediaBannerSellerFoto,
		IdSeller: int64(data.IdentitasSeller.IdSeller),
		Key:      data.KeyFoto,
	}).Limit(1).Scan(&data_media_banner_foto_seller).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_banner_foto_seller.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
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

	go func(Dmbfs models.MediaSellerBannerFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: Dmbfs.IdSeller,
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.MediaSellerBannerFoto); err != nil {
			fmt.Println("Gagal decr banner foto ke threshold seller")
		}

		bannerFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbfs).SetTableName(Dmbfs.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bannerFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete banner foto seller ke message broker")
		}
	}(data_media_banner_foto_seller, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahkanFotoTokoFisikSeller(ctx context.Context, data PayloadTambahkanFotoTokoFisikSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahkanFotoTokoFisikSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	totalData := len(data.Ekstensi)

	if totalData > 20 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusRequestEntityTooLarge,
			Services: services,
		}
	}

	var DataFotoSave []models.MediaSellerTokoFisikFoto = make([]models.MediaSellerTokoFisikFoto, 0, totalData)
	var DataUrlAndKeyResponse []response.UrlAndKey = make([]response.UrlAndKey, 0, totalData)

	for i := 0; i < totalData; i++ {
		ext := strings.ToLower(data.Ekstensi[i])
		if !media_ekstension.PhotoValidExt[ext] {
			return &response.ResponseMediaUpload{
				Status:   http.StatusBadRequest,
				Services: services,
			}
		}

		keyz := models.MediaSellerTokoFisikFoto{}.PathName() +
			strconv.Itoa(int(data.IdentitasSeller.IdSeller)) + "/" +
			helper.GenerateMediaKeyPhoto() + "." + ext

		url, errPPO := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*10)
		if errPPO != nil {
			return &response.ResponseMediaUpload{
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
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	go func(Tffs []models.MediaSellerTokoFisikFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if len(Tffs) > 0 {
			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(Tffs[0].IdSeller),
			}

			if err := thresholdSeller.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{FieldName: stsk_seller.MediaSellerTokoFisikFoto, Count: len(Tffs)}}); err != nil {
				fmt.Println("Gagal incr count media seller toko fisik foto ke threshold seller")
			}

			for _, foto := range Tffs {
				tokoFisikFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName())
				if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, tokoFisikFotoCreatePublish); err != nil {
					fmt.Println("Gagal publish toko fisik foto create ke message broker")
				}
			}
		}
	}(DataFotoSave, db.Write, cud_publisher)

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlAndKey: DataUrlAndKeyResponse,
	}
}

func HapusFotoTokoFisikSeller(ctx context.Context, data PayloadHapusFotoTokoFisikSeller, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusFotoTokoFisikSeller"
	var wg sync.WaitGroup
	var mu sync.RWMutex

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	dataTotal := len(data.DataMediaFotoTokoFisik)
	errChan := make(chan error, dataTotal)
	var idsHapus []int64 = make([]int64, 0, dataTotal)
	var dataFotoHapus []models.MediaSellerTokoFisikFoto = make([]models.MediaSellerTokoFisikFoto, 0, dataTotal)

	for i := 0; i < dataTotal; i++ {
		wg.Add(1)
		go func(angka int) {
			defer wg.Done()
			var data_media_foto_toko_fisik models.MediaSellerTokoFisikFoto
			if err := db.Read.WithContext(ctx).Model(&models.MediaSellerTokoFisikFoto{}).Where(&models.MediaSellerTokoFisikFoto{
				ID:  data.DataMediaFotoTokoFisik[angka].IdMedia,
				Key: data.DataMediaFotoTokoFisik[angka].KeyMedia,
			}).Limit(1).Scan(&data_media_foto_toko_fisik).Error; err != nil {
				errChan <- fmt.Errorf("gagal")
				return
			}

			if data_media_foto_toko_fisik.ID == 0 {
				errChan <- fmt.Errorf("gagal")
				return
			}

			ctxRemove, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			if err := ms.RemoveObject(ctxRemove, media_storage_database_seeders.BucketFotoName, data.DataMediaFotoTokoFisik[angka].KeyMedia, minio.RemoveObjectOptions{
				ForceDelete: true,
			}); err != nil {
				errChan <- fmt.Errorf("gagal")
				return
			}

			mu.Lock()
			idsHapus = append(idsHapus, data_media_foto_toko_fisik.ID)
			dataFotoHapus = append(dataFotoHapus, data_media_foto_toko_fisik)
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

	go func(Dtfh []models.MediaSellerTokoFisikFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if len(Dtfh) > 0 {
			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(Dtfh[0].IdSeller),
			}

			if err := thresholdSeller.CustomDecrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{FieldName: stsk_seller.MediaSellerTokoFisikFoto, Count: len(Dtfh)}}); err != nil {
				fmt.Println("Gagal decr foto toko fisik ke threshold seller")
			}

			for _, foto := range Dtfh {
				tokoFisikFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName())
				if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, tokoFisikFotoDeletePublish); err != nil {
					fmt.Println("Gagal publish delete foto toko fisik seller ke message broker")
				}
			}
		}
	}(dataFotoHapus, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UbahFotoEtalaseSeller(ctx context.Context, data PayloadUbahFotoEtalase, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahFotoEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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
	}).Limit(1).Scan(&id_data_media_etalase_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaEtalaseFoto{}.PathName() + strconv.Itoa(int(id_data_etalase)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrl := url.String()

	if id_data_media_etalase_foto == 0 {
		newEtalaseFoto := models.MediaEtalaseFoto{
			IdEtalase: id_data_etalase,
			Key:       keyz,
			Format:    data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newEtalaseFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Ef models.MediaEtalaseFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			etalaseFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ef).SetTableName(Ef.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish etalase foto create ke message broker")
			}
		}(newEtalaseFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Where(&models.MediaEtalaseFoto{
			ID: id_data_media_etalase_foto,
		}).Updates(&models.MediaEtalaseFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdEf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataEtalaseFotoUpdated models.MediaEtalaseFoto
			if err := Read.WithContext(konteks).Model(&models.MediaEtalaseFoto{}).Where(&models.MediaEtalaseFoto{
				ID: IdEf,
			}).Limit(1).Take(&dataEtalaseFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data etalase foto")
			}

			etalaseFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataEtalaseFotoUpdated).SetTableName(dataEtalaseFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update etalase foto ke message broker")
			}
		}(id_data_media_etalase_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUrl,
		Key:       keyz,
	}
}

func HapusFotoEtalaseSeller(ctx context.Context, data PayloadHapusFotoEtalase, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusFotoEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_etalase_foto models.MediaEtalaseFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Where(&models.MediaEtalaseFoto{
		ID:  data.IdMediaEtalaseFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_etalase_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_etalase_foto.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data media foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaEtalaseFoto{}).Where(&models.MediaEtalaseFoto{
		ID: data_media_etalase_foto.ID,
	}).Delete(&models.MediaEtalaseFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk, coba lagi lain waktu",
		}
	}

	go func(Dmef models.MediaEtalaseFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		etalaseFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmef).SetTableName(Dmef.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete etalase foto ke message broker")
		}
	}(data_media_etalase_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahkanMediaBarangIndukFoto(ctx context.Context, data PayloadTambahBarangIndukFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahkanMediaBarangIndukFoto"
	const Limit uint8 = 10

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       data.IdBarangInduk,
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

	dataLength := len(data.Ekstensi)

	if dataLength > 10 {
		return &response.ResponseMediaUpload{
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
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	ExistPhoto := len(idsDataMediaBarangInduk)
	if ExistPhoto >= int(Limit) {
		return &response.ResponseMediaUpload{
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
			return &response.ResponseMediaUpload{
				Status:   http.StatusBadRequest,
				Services: services,
			}
		}

		keyz := models.MediaBarangIndukFoto{}.PathName() + strconv.Itoa(int(id_data_barang_induk)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi[i]

		url, err := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
		if err != nil {
			return &response.ResponseMediaUpload{
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
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	go func(Mbif []models.MediaBarangIndukFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Mbif[0].IdBarangInduk,
		}

		if err := thresholdBarangInduk.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{
			FieldName: stsk_baranginduk.MediaBarangIndukFoto,
			Count:     len(Mbif),
		}}); err != nil {
			fmt.Println("Gagal incr count media barang induk foto threshold Barang induk")
		}

		for _, foto := range Mbif {
			barangIndukFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangIndukFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish barang induk foto create ke message broker")
			}
		}
	}(DataMediaBarangIndukFoto, db.Write, cud_publisher)

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlAndKey: DataUrlAndKeyUpload,
	}
}

func HapusMediaBarangIndukFoto(ctx context.Context, data PayloadHapusBarangIndukFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaBarangIndukFoto"
	var wg sync.WaitGroup
	var mu sync.RWMutex

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var dataLength = len(data.DataMediaBarangIndukFoto)

	var dataMediaBarangIndukFotoHapus []models.MediaBarangIndukFoto = make([]models.MediaBarangIndukFoto, 0, dataLength)
	for i := range dataLength {
		wg.Add(1)
		go func(a int) {
			defer wg.Done()
			var data_media_barang_induk_foto models.MediaBarangIndukFoto
			if err := db.Read.WithContext(ctx).Model(&models.MediaBarangIndukFoto{}).Where(&models.MediaBarangIndukFoto{
				ID:  data.DataMediaBarangIndukFoto[a].IdMedia,
				Key: data.DataMediaBarangIndukFoto[a].KeyMedia,
			}).Limit(1).Scan(&data_media_barang_induk_foto).Error; err != nil {
				return
			}

			if data_media_barang_induk_foto.ID == 0 {
				return
			}

			mu.Lock()
			dataMediaBarangIndukFotoHapus = append(dataMediaBarangIndukFotoHapus, data_media_barang_induk_foto)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if len(dataMediaBarangIndukFotoHapus) == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	var idsHapus []int64 = make([]int64, 0, len(dataMediaBarangIndukFotoHapus))
	for _, foto := range dataMediaBarangIndukFotoHapus {
		idsHapus = append(idsHapus, foto.ID)
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBarangIndukFoto{}).Where("id IN ?", idsHapus).Delete(&models.MediaBarangIndukFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbifh []models.MediaBarangIndukFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Dmbifh[0].IdBarangInduk,
		}

		if err := barangIndukThreshold.CustomDecrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{
			FieldName: stsk_baranginduk.MediaBarangIndukFoto,
		}}); err != nil {
			fmt.Println("Gagal decr count media barang induk foto threshold Barang induk")
		}

		for _, foto := range Dmbifh {
			barangIndukFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName())
			if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangIndukFotoDeletePublish); err != nil {
				fmt.Println("Gagal publish delete barang induk foto ke message broker")
			}
		}
	}(dataMediaBarangIndukFotoHapus, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UbahBarangIndukVideo(ctx context.Context, data PayloadUbahVideoBarangInduk, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahBarangIndukVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketVideoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUploadUrl := url.String()
	if id_media_data_barang_induk_video == 0 {
		newBarangIndukVideo := models.MediaBarangIndukVideo{
			IdBarangInduk: data.IdBarangInduk,
			Key:           keyz,
			Format:        data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newBarangIndukVideo).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Biv models.MediaBarangIndukVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
				IdBarangInduk: Biv.IdBarangInduk,
			}

			if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.MediaBarangIndukVideo); err != nil {
				fmt.Println("gagal publish create video barang induk ke message broker ")
			}

			barangIndukVideoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Biv).SetTableName(Biv.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangIndukVideoCreatePublish); err != nil {
				fmt.Println("Gagal publish barang induk video create ke message broker")
			}
		}(newBarangIndukVideo, db.Write, cud_publisher)
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

		go func(IdBiv int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBarangIndukVideoUpdated models.MediaBarangIndukVideo
			if err := Read.WithContext(konteks).Model(&models.MediaBarangIndukVideo{}).Where(&models.MediaBarangIndukVideo{
				ID: IdBiv,
			}).Limit(1).Take(&dataBarangIndukVideoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data barang induk video")
			}

			barangIndukVideoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBarangIndukVideoUpdated).SetTableName(dataBarangIndukVideoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangIndukVideoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update barang induk video ke message broker")
			}
		}(id_media_data_barang_induk_video, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUploadUrl,
		Key:       keyz,
	}
}

func HapusBarangIndukVideo(ctx context.Context, data PayloadHapusVideoBarangInduk, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusBarangIndukVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_barang_induk_video models.MediaBarangIndukVideo
	if err := db.Read.WithContext(ctx).Model(&models.MediaBarangIndukVideo{}).Where(&models.MediaBarangIndukVideo{
		ID:  data.IdMediaBarangIndukVideo,
		Key: data.KeyVideo,
	}).Limit(1).Scan(&data_media_barang_induk_video).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_barang_induk_video.ID == 0 {
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

	go func(Dmbiv models.MediaBarangIndukVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Dmbiv.IdBarangInduk,
		}

		if err := thresholdBarangInduk.Decrement(konteks, Trh, stsk_baranginduk.MediaBarangIndukVideo); err != nil {
			fmt.Println("gagal publish delete video barang induk ke message broker ")
		}

		barangIndukVideoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbiv).SetTableName(Dmbiv.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangIndukVideoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete barang induk video ke message broker")
		}
	}(data_media_barang_induk_video, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UbahKategoriBarangFoto(ctx context.Context, data PayloadUbahKategoriBarangFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahKategoriBarangFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, url_err := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if url_err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrlUpload := url.String()

	if id_media_kategori_barang_foto == 0 {
		newKategoriBarangFoto := models.MediaKategoriBarangFoto{
			IdKategoriBarang: data.IdKategoriBarang,
			IdBarangInduk:    data.IdBarangInduk,
			Key:              keyz,
			Format:           data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newKategoriBarangFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mkbf models.MediaKategoriBarangFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: Mkbf.IdKategoriBarang,
			}

			if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.MediaKategoriBarangFoto); err != nil {
				fmt.Println("Gagal incr count media kategori barang foto ke threshold kategori barang")
			}

			kategoriBarangFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mkbf).SetTableName(Mkbf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish kategori barang foto create ke message broker")
			}
		}(newKategoriBarangFoto, db.Write, cud_publisher)
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

		go func(IdMkbf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataKategoriBarangFotoUpdated models.MediaKategoriBarangFoto
			if err := Read.WithContext(konteks).Model(&models.MediaKategoriBarangFoto{}).Where(&models.MediaKategoriBarangFoto{
				ID: IdMkbf,
			}).Limit(1).Take(&dataKategoriBarangFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data kategori barang foto")
			}

			kategoriBarangFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangFotoUpdated).SetTableName(dataKategoriBarangFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update kategori barang foto ke message broker")
			}
		}(id_media_kategori_barang_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: minIOUrlUpload,
	}
}

func HapusKategoriBarangFoto(ctx context.Context, data PayloadHapusKategoriBarangFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKategoriBarangFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_kategori_barang_foto models.MediaKategoriBarangFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaKategoriBarangFoto{}).Where(&models.MediaKategoriBarangFoto{
		ID:  data.IdMediaKategoriBarangFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_kategori_barang_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_kategori_barang_foto.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaKategoriBarangFoto{}).Where(&models.MediaKategoriBarangFoto{
		ID: data_media_kategori_barang_foto.ID,
	}).Delete(&models.MediaKategoriBarangFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmkbf models.MediaKategoriBarangFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Dmkbf.IdKategoriBarang,
		}

		if err := thresholdKategoriBarang.Decrement(konteks, Trh, stsk_kategori_barang.MediaKategoriBarangFoto); err != nil {
			fmt.Println("Gagal decr media kategori barang foto ke threshold kategori barang")
		}

		kategoriBarangFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmkbf).SetTableName(Dmkbf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete kategori barang foto ke message broker")
		}
	}(data_media_kategori_barang_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahDistributorDataDokumen(ctx context.Context, data PayloadMediaDistributorDataDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahDistributorDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_dokumen == 0 {
		newDistributorDataDokumen := models.MediaDistributorDataDokumen{
			IdDistributorData: data.IdDistributorData,
			Key:               keyz,
			Format:            data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newDistributorDataDokumen).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mddd models.MediaDistributorDataDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddd.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataDokumen); err != nil {
				fmt.Println("Gagal incr count media distributor data dokumen ke threshold distributor data")
			}

			distributorDataDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddd).SetTableName(Mddd.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data dokumen create ke message broker")
			}
		}(newDistributorDataDokumen, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Where(&models.MediaDistributorDataDokumen{
			ID: id_media_distributor_data_dokumen,
		}).Updates(&models.MediaDistributorDataDokumen{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMddd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataDistributorDataDokumenUpdated models.MediaDistributorDataDokumen
			if err := Read.WithContext(konteks).Model(&models.MediaDistributorDataDokumen{}).Where(&models.MediaDistributorDataDokumen{
				ID: IdMddd,
			}).Limit(1).Take(&dataDistributorDataDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data dokumen")
			}

			distributorDataDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataDokumenUpdated).SetTableName(dataDistributorDataDokumenUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataDokumenUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update distributor data dokumen ke message broker")
			}
		}(id_media_distributor_data_dokumen, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaDistributorDataDokumen(ctx context.Context, data PayloadHapusMediaDistributorDataDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaDistributorDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_distributor_data int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.DistributorData{}).Select("id").Where(&models.DistributorData{
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

	var data_media_distributor_data_dokumen models.MediaDistributorDataDokumen
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Where(&models.MediaDistributorDataDokumen{
		ID:  data.IdMediaDistributorDataDokumen,
		Key: data.KeyDokumen,
	}).Limit(1).Scan(&data_media_distributor_data_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_distributor_data_dokumen.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data dokumen tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketDokumenName, data.KeyDokumen, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataDokumen{}).Where(&models.MediaDistributorDataDokumen{
		ID: data_media_distributor_data_dokumen.ID,
	}).Delete(&models.MediaDistributorDataDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmddd models.MediaDistributorDataDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmddd.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataDokumen); err != nil {
			fmt.Println("Gagal decr media distributor data dokumen ke threshold distributor data")
		}

		distributorDataDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmddd).SetTableName(Dmddd.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataDokumenDeletePublish); err != nil {
			fmt.Println("Gagal publish delete distributor data dokumen ke message broker")
		}
	}(data_media_distributor_data_dokumen, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaDistributorDataNPWPFoto(ctx context.Context, data PayloadTambahMediaDistributorDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaDistributorDataNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_npwp_foto == 0 {
		newDistributorDataNPWPFoto := models.MediaDistributorDataNPWPFoto{
			IdDistributorData: id_data_distributor,
			Key:               keyz,
			Format:            data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newDistributorDataNPWPFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mddnf models.MediaDistributorDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddnf.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataNpwpFoto); err != nil {
				fmt.Println("Gagal incr count media distributor data npwp foto ke threshold distributor data")
			}

			distributorDataNPWPFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddnf).SetTableName(Mddnf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNPWPFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data npwp foto create ke message broker")
			}
		}(newDistributorDataNPWPFoto, db.Write, cud_publisher)
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

		go func(IdMddnf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataDistributorDataNPWPFotoUpdated models.MediaDistributorDataNPWPFoto
			if err := Read.WithContext(konteks).Model(&models.MediaDistributorDataNPWPFoto{}).Where(&models.MediaDistributorDataNPWPFoto{
				ID: IdMddnf,
			}).Limit(1).Take(&dataDistributorDataNPWPFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data npwp foto")
			}

			distributorDataNPWPFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataNPWPFotoUpdated).SetTableName(dataDistributorDataNPWPFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNPWPFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update distributor data npwp foto ke message broker")
			}
		}(id_media_distributor_data_npwp_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaDistributorDataNPWPFoto(ctx context.Context, data PayloadHapusMediaDistributorDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaDistributorDataNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var data_media_distributor_npwp_foto models.MediaDistributorDataNPWPFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataNPWPFoto{}).Where(&models.MediaDistributorDataNPWPFoto{
		ID:  data.IdMediaDistributorDataNPWPFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_distributor_npwp_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_distributor_npwp_foto.ID == 0 {
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

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNPWPFoto{}).Where(&models.MediaDistributorDataNPWPFoto{
		ID: data_media_distributor_npwp_foto.ID,
	}).Delete(&models.MediaDistributorDataNPWPFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmdnf models.MediaDistributorDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmdnf.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataNpwpFoto); err != nil {
			fmt.Println("Gagal decr media distributor data npwp foto ke threshold distributor data")
		}

		distributorDataNPWPFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmdnf).SetTableName(Dmdnf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNPWPFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete distributor data npwp foto ke message broker")
		}
	}(data_media_distributor_npwp_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahDistributorDataNIBFoto(ctx context.Context, data PayloadTambahDistributorDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahDistributorDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Select("id").Where(&models.MediaDistributorDataNIBFoto{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_nib_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaDistributorDataNIBFoto{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_nib_foto == 0 {
		newDistributorDataNIBFoto := models.MediaDistributorDataNIBFoto{
			IdDistributorData: id_data_distributor,
			Key:               keyz,
			Format:            data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newDistributorDataNIBFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mddnibf models.MediaDistributorDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddnibf.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataNibFoto); err != nil {
				fmt.Println("Gagal incr count media distributor data nib foto ke threshold distributor data")
			}

			distributorDataNIBFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddnibf).SetTableName(Mddnibf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNIBFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data nib foto create ke message broker")
			}
		}(newDistributorDataNIBFoto, db.Write, cud_publisher)
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

		go func(IdMddnibf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataDistributorDataNIBFotoUpdated models.MediaDistributorDataNIBFoto
			if err := Read.WithContext(konteks).Model(&models.MediaDistributorDataNIBFoto{}).Where(&models.MediaDistributorDataNIBFoto{
				ID: IdMddnibf,
			}).Limit(1).Take(&dataDistributorDataNIBFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data nib foto")
			}

			distributorDataNIBFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataNIBFotoUpdated).SetTableName(dataDistributorDataNIBFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNIBFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update distributor data nib foto ke message broker")
			}
		}(id_media_distributor_data_nib_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusDistributorDataNIBFoto(ctx context.Context, data PayloadHapusDistributorDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDistributorDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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
			Message:  "Gagal data tidak ditemukan",
		}
	}

	var data_media_distributor_nib_foto models.MediaDistributorDataNIBFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Where(&models.MediaDistributorDataNIBFoto{
		ID:                data.IdMediaDistributorDataNIBFoto,
		IdDistributorData: data.IdDistributorData,
		Key:               data.KeyFoto,
	}).Limit(1).Scan(&data_media_distributor_nib_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_distributor_nib_foto.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Foto Tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketFotoName, data.KeyFoto, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataNIBFoto{}).Where(&models.MediaDistributorDataNIBFoto{
		ID: data_media_distributor_nib_foto.ID,
	}).Delete(&models.MediaDistributorDataNIBFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmdnibf models.MediaDistributorDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmdnibf.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataNibFoto); err != nil {
			fmt.Println("Gagal decr media distributor data nib foto ke threshold distributor data")
		}

		distributorDataNIBFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmdnibf).SetTableName(Dmdnibf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNIBFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete distributor data nib foto ke message broker")
		}
	}(data_media_distributor_nib_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahDistributorDataSuratKerjasamaDokumen(ctx context.Context, data PayloadTambahDistributorDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahDistributorDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_surat_kerjasama_dokumen == 0 {
		newDistributorDataSuratKerjasamaDokumen := models.MediaDistributorDataSuratKerjasamaDokumen{
			IdDistributorData: id_data_distributor,
			Key:               keyz,
			Format:            data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newDistributorDataSuratKerjasamaDokumen).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mddskd models.MediaDistributorDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddskd.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataSuratKerjasamaDokumen); err != nil {
				fmt.Println("Gagal incr count media distributor data surat kerjasama dokumen ke threshold distributor data")
			}

			distributorDataSuratKerjasamaDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddskd).SetTableName(Mddskd.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataSuratKerjasamaDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data surat kerjasama dokumen create ke message broker")
			}
		}(newDistributorDataSuratKerjasamaDokumen, db.Write, cud_publisher)
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

		go func(IdMddskd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataDistributorDataSuratKerjasamaDokumenUpdated models.MediaDistributorDataSuratKerjasamaDokumen
			if err := Read.WithContext(konteks).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
				ID: IdMddskd,
			}).Limit(1).Take(&dataDistributorDataSuratKerjasamaDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data surat kerjasama dokumen")
			}

			distributorDataSuratKerjasamaDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataSuratKerjasamaDokumenUpdated).SetTableName(dataDistributorDataSuratKerjasamaDokumenUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataSuratKerjasamaDokumenUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update distributor data surat kerjasama dokumen ke message broker")
			}
		}(id_media_distributor_data_surat_kerjasama_dokumen, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusDistributorDataSuratKerjasamaDataDokumen(ctx context.Context, data PayloadHapusDistributorDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDistributorDataSuratKerjasamaDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_distributor_data_surat_kerjasama_dokumen models.MediaDistributorDataSuratKerjasamaDokumen
	if err := db.Read.WithContext(ctx).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
		IdDistributorData: id_data_distributor,
		Key:               data.KeyDokumen,
	}).Limit(1).Scan(&data_media_distributor_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_distributor_data_surat_kerjasama_dokumen.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data dokumen tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketDokumenName, data.KeyDokumen, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&models.MediaDistributorDataSuratKerjasamaDokumen{
		ID: data_media_distributor_data_surat_kerjasama_dokumen.ID,
	}).Delete(&models.MediaDistributorDataSuratKerjasamaDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmdskd models.MediaDistributorDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmdskd.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataSuratKerjasamaDokumen); err != nil {
			fmt.Println("Gagal decr media distributor data surat kerjasama dokumen ke threshold distributor data")
		}

		distributorDataSuratKerjasamaDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmdskd).SetTableName(Dmdskd.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataSuratKerjasamaDokumenDeletePublish); err != nil {
			fmt.Println("Gagal publish delete distributor data surat kerjasama dokumen ke message broker")
		}
	}(data_media_distributor_data_surat_kerjasama_dokumen, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahBrandDataPerwakilanDokumen(ctx context.Context, data PayloadTambahBrandDataPerwakilanDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahBrandDataPerwakilanDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_perwakilan_dokumen == 0 {
		newBrandDataPerwakilanDokumen := models.MediaBrandDataPerwakilanDokumen{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}

		if err := db.Write.WithContext(ctx).Create(&newBrandDataPerwakilanDokumen).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mbdpd models.MediaBrandDataPerwakilanDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdpd.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataPerwakilanDokumen); err != nil {
				fmt.Println("Gagal incr count media brand data perwakilan dokumen ke threshold brand data")
			}

			brandDataPerwakilanDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdpd).SetTableName(Mbdpd.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataPerwakilanDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data perwakilan dokumen create ke message broker")
			}
		}(newBrandDataPerwakilanDokumen, db.Write, cud_publisher)
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

		go func(IdMbdpd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBrandDataPerwakilanDokumenUpdated models.MediaBrandDataPerwakilanDokumen
			if err := Read.WithContext(konteks).Model(&models.MediaBrandDataPerwakilanDokumen{}).Where(&models.MediaBrandDataPerwakilanDokumen{
				ID: IdMbdpd,
			}).Limit(1).Take(&dataBrandDataPerwakilanDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data perwakilan dokumen")
			}

			brandDataPerwakilanDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataPerwakilanDokumenUpdated).SetTableName(dataBrandDataPerwakilanDokumenUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataPerwakilanDokumenUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update brand data perwakilan dokumen ke message broker")
			}
		}(id_media_brand_data_perwakilan_dokumen, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaBrandDataPerwakilanDokumen(ctx context.Context, data PayloadHapusBrandDataPerwakilanDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaBrandDataPerwakilanDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_brand_data_perwakilan_dokumen models.MediaBrandDataPerwakilanDokumen
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataPerwakilanDokumen{}).Where(&models.MediaBrandDataPerwakilanDokumen{
		ID:  data.IdMediaBrandDataPerwakilanDokumen,
		Key: data.KeyDokumen,
	}).Limit(1).Scan(&data_media_brand_data_perwakilan_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_brand_data_perwakilan_dokumen.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data dokumen tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketDokumenName, data.KeyDokumen, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataPerwakilanDokumen{}).Where(&models.MediaBrandDataPerwakilanDokumen{
		ID: data_media_brand_data_perwakilan_dokumen.ID,
	}).Delete(&models.MediaBrandDataPerwakilanDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdpd models.MediaBrandDataPerwakilanDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdpd.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataPerwakilanDokumen); err != nil {
			fmt.Println("Gagal decr media brand data perwakilan dokumen ke threshold brand data")
		}

		brandDataPerwakilanDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdpd).SetTableName(Dmbdpd.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataPerwakilanDokumenDeletePublish); err != nil {
			fmt.Println("Gagal publish delete brand data perwakilan dokumen ke message broker")
		}
	}(data_media_brand_data_perwakilan_dokumen, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaBrandDataSertifikatFoto(ctx context.Context, data PayloadTambahBrandDataSertifikatFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaBrandDataSertifikatFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_sertifikat_foto == 0 {
		newBrandDataSertifikatFoto := models.MediaBrandDataSertifikatFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}

		if err := db.Write.WithContext(ctx).Create(&newBrandDataSertifikatFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mbdsf models.MediaBrandDataSertifikatFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdsf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataSertifikatFoto); err != nil {
				fmt.Println("Gagal incr count media brand data sertifikat foto ke threshold brand data")
			}

			brandDataSertifikatFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdsf).SetTableName(Mbdsf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSertifikatFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data sertifikat foto create ke message broker")
			}
		}(newBrandDataSertifikatFoto, db.Write, cud_publisher)
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

		go func(IdMbdsf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBrandDataSertifikatFotoUpdated models.MediaBrandDataSertifikatFoto
			if err := Read.WithContext(konteks).Model(&models.MediaBrandDataSertifikatFoto{}).Where(&models.MediaBrandDataSertifikatFoto{
				ID: IdMbdsf,
			}).Limit(1).Take(&dataBrandDataSertifikatFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data sertifikat foto")
			}

			brandDataSertifikatFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataSertifikatFotoUpdated).SetTableName(dataBrandDataSertifikatFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSertifikatFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update brand data sertifikat foto ke message broker")
			}
		}(id_media_brand_data_sertifikat_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaBrandDataSertifikatFoto(ctx context.Context, data PayloadHapusBrandDataSertifikatFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaBrandDataSertifikatFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_brand_data_sertifikat_foto models.MediaBrandDataSertifikatFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataSertifikatFoto{}).Where(&models.MediaBrandDataSertifikatFoto{
		ID:  data.IdMediaBrandDataSertifikatFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_brand_data_sertifikat_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_brand_data_sertifikat_foto.ID == 0 {
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

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataSertifikatFoto{}).Where(&models.MediaBrandDataSertifikatFoto{
		ID: data_media_brand_data_sertifikat_foto.ID,
	}).Delete(&models.MediaBrandDataSertifikatFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdsf models.MediaBrandDataSertifikatFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdsf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataSertifikatFoto); err != nil {
			fmt.Println("Gagal decr media brand data sertifikat foto ke threshold brand data")
		}

		brandDataSertifikatFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdsf).SetTableName(Dmbdsf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSertifikatFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete brand data sertifikat foto ke message broker")
		}
	}(data_media_brand_data_sertifikat_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaBrandDataNIBFoto(ctx context.Context, data PayloadTambahMediaBrandDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaBrandDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_nib_foto == 0 {
		newBrandDataNIBFoto := models.MediaBrandDataNIBFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}

		if err := db.Write.WithContext(ctx).Create(&newBrandDataNIBFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mbdnibf models.MediaBrandDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdnibf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataNibFoto); err != nil {
				fmt.Println("Gagal incr count media brand data nib foto ke threshold brand data")
			}

			brandDataNIBFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdnibf).SetTableName(Mbdnibf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNIBFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data nib foto create ke message broker")
			}
		}(newBrandDataNIBFoto, db.Write, cud_publisher)
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

		go func(IdMbdnibf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBrandDataNIBFotoUpdated models.MediaBrandDataNIBFoto
			if err := Read.WithContext(konteks).Model(&models.MediaBrandDataNIBFoto{}).Where(&models.MediaBrandDataNIBFoto{
				ID: IdMbdnibf,
			}).Limit(1).Take(&dataBrandDataNIBFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data nib foto")
			}

			brandDataNIBFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataNIBFotoUpdated).SetTableName(dataBrandDataNIBFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNIBFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update brand data nib foto ke message broker")
			}
		}(id_media_brand_data_nib_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaBrandDataNIBFoto(ctx context.Context, data PayloadHapusMediaBrandDataNIBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaBrandDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_brand_data_nib_foto models.MediaBrandDataNIBFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataNIBFoto{}).Where(&models.MediaBrandDataNIBFoto{
		ID:          data.IdMediaBrandDataNIBFoto,
		IdBrandData: id_data_brand,
		Key:         data.KeyFoto,
	}).Limit(1).Scan(&data_media_brand_data_nib_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_brand_data_nib_foto.ID == 0 {
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

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataNIBFoto{}).Where(&models.MediaBrandDataNIBFoto{
		ID: data_media_brand_data_nib_foto.ID,
	}).Delete(&models.MediaBrandDataNIBFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdnibf models.MediaBrandDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdnibf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataNibFoto); err != nil {
			fmt.Println("Gagal decr media brand data nib foto ke threshold brand data")
		}

		brandDataNIBFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdnibf).SetTableName(Dmbdnibf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNIBFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete brand data nib foto ke message broker")
		}
	}(data_media_brand_data_nib_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaBrandNPWPFoto(ctx context.Context, data PayloadTambahMediaBrandDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaBrandNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_npwp_foto == 0 {
		newBrandDataNPWPFoto := models.MediaBrandDataNPWPFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}

		if err := db.Write.WithContext(ctx).Create(&newBrandDataNPWPFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mbdnf models.MediaBrandDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdnf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataNpwpFoto); err != nil {
				fmt.Println("Gagal incr count media brand data npwp foto ke threshold brand data")
			}

			brandDataNPWPFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdnf).SetTableName(Mbdnf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNPWPFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data npwp foto create ke message broker")
			}
		}(newBrandDataNPWPFoto, db.Write, cud_publisher)
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

		go func(IdMbdnf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBrandDataNPWPFotoUpdated models.MediaBrandDataNPWPFoto
			if err := Read.WithContext(konteks).Model(&models.MediaBrandDataNPWPFoto{}).Where(&models.MediaBrandDataNPWPFoto{
				ID: IdMbdnf,
			}).Limit(1).Take(&dataBrandDataNPWPFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data npwp foto")
			}

			brandDataNPWPFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataNPWPFotoUpdated).SetTableName(dataBrandDataNPWPFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNPWPFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update brand data npwp foto ke message broker")
			}
		}(id_media_brand_data_npwp_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaBrandNPWPFoto(ctx context.Context, data PayloadHapusMediaBrandDataNPWPFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaBrandNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_brand_data_npwp_foto models.MediaBrandDataNPWPFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataNPWPFoto{}).Where(&models.MediaBrandDataNPWPFoto{
		ID:  data.IdMediaBrandDataNPWPFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_brand_data_npwp_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_brand_data_npwp_foto.ID == 0 {
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

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataNPWPFoto{}).Where(&models.MediaBrandDataNPWPFoto{
		ID: data_media_brand_data_npwp_foto.ID,
	}).Delete(&models.MediaBrandDataNPWPFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdnf models.MediaBrandDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdnf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataNpwpFoto); err != nil {
			fmt.Println("Gagal decr media brand data npwp foto ke threshold brand data")
		}

		brandDataNPWPFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdnf).SetTableName(Dmbdnf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNPWPFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete brand data npwp foto ke message broker")
		}
	}(data_media_brand_data_npwp_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaBrandDataLogoFoto(ctx context.Context, data PayloadTambahMediaBrandDataLogoFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaBrandDataLogoBrandFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	keyz := models.MediaBrandDataLogoFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_logo_foto == 0 {
		newBrandDataLogoFoto := models.MediaBrandDataLogoFoto{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}

		if err := db.Write.WithContext(ctx).Create(&newBrandDataLogoFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mbdlf models.MediaBrandDataLogoFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdlf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataLogoBrandFoto); err != nil {
				fmt.Println("Gagal incr count media brand data logo foto ke threshold brand data")
			}

			brandDataLogoFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdlf).SetTableName(Mbdlf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataLogoFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data logo foto create ke message broker")
			}
		}(newBrandDataLogoFoto, db.Write, cud_publisher)
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

		go func(IdMbdlf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBrandDataLogoFotoUpdated models.MediaBrandDataLogoFoto
			if err := Read.WithContext(konteks).Model(&models.MediaBrandDataLogoFoto{}).Where(&models.MediaBrandDataLogoFoto{
				ID: IdMbdlf,
			}).Limit(1).Take(&dataBrandDataLogoFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data logo foto")
			}

			brandDataLogoFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataLogoFotoUpdated).SetTableName(dataBrandDataLogoFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataLogoFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update brand data logo foto ke message broker")
			}
		}(id_media_brand_data_logo_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaBrandDataLogo(ctx context.Context, data PayloadHapusMediaBrandDataLogoFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaBrandDataLogo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_brand_data_logo_foto models.MediaBrandDataLogoFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataLogoFoto{}).Where(&models.MediaBrandDataLogoFoto{
		ID:          data.IdMediaBrandDataLogoBrandFoto,
		IdBrandData: id_data_brand,
		Key:         data.KeyFoto,
	}).Limit(1).Scan(&data_media_brand_data_logo_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_brand_data_logo_foto.ID == 0 {
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

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataLogoFoto{}).Where(&models.MediaBrandDataLogoFoto{
		ID: data_media_brand_data_logo_foto.ID,
	}).Delete(&models.MediaBrandDataLogoFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdlf models.MediaBrandDataLogoFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdlf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataLogoBrandFoto); err != nil {
			fmt.Println("Gagal decr media brand data logo foto ke threshold brand data")
		}

		brandDataLogoFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdlf).SetTableName(Dmbdlf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataLogoFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete brand data logo foto ke message broker")
		}
	}(data_media_brand_data_logo_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahBrandDataSuratKerjasamaDokumen(ctx context.Context, data PayloadTambahBrandDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahBrandDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var id_media_brand_data_surat_kerjasama_dokumen int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataSuratKerjasamaDokumen{}).Select("id").Where(&models.MediaBrandDataSuratKerjasamaDokumen{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := models.MediaBrandDataSuratKerjasamaDokumen{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_surat_kerjasama_dokumen == 0 {
		newBrandDataSuratKerjasamaDokumen := models.MediaBrandDataSuratKerjasamaDokumen{
			IdBrandData: id_data_brand,
			Key:         keyz,
			Format:      data.Ekstensi,
		}

		if err := db.Write.WithContext(ctx).Create(&newBrandDataSuratKerjasamaDokumen).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mbdskd models.MediaBrandDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdskd.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataSuratKerjasamaDokumen); err != nil {
				fmt.Println("Gagal incr count media brand data surat kerjasama dokumen ke threshold brand data")
			}

			brandDataSuratKerjasamaDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdskd).SetTableName(Mbdskd.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSuratKerjasamaDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data surat kerjasama dokumen create ke message broker")
			}
		}(newBrandDataSuratKerjasamaDokumen, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&models.MediaBrandDataSuratKerjasamaDokumen{
			ID: id_media_brand_data_surat_kerjasama_dokumen,
		}).Updates(&models.MediaBrandDataSuratKerjasamaDokumen{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdskd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataBrandDataSuratKerjasamaDokumenUpdated models.MediaBrandDataSuratKerjasamaDokumen
			if err := Read.WithContext(konteks).Model(&models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&models.MediaBrandDataSuratKerjasamaDokumen{
				ID: IdMbdskd,
			}).Limit(1).Take(&dataBrandDataSuratKerjasamaDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data surat kerjasama dokumen")
			}

			brandDataSuratKerjasamaDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataSuratKerjasamaDokumenUpdated).SetTableName(dataBrandDataSuratKerjasamaDokumenUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSuratKerjasamaDokumenUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update brand data surat kerjasama dokumen ke message broker")
			}
		}(id_media_brand_data_surat_kerjasama_dokumen, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusBrandDataSuratKerjasamaDokumen(ctx context.Context, data PayloadHapusBrandDataSuratKerjasamaDokumen, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusBrandDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
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

	var data_media_brand_data_surat_kerjasama_dokumen models.MediaBrandDataSuratKerjasamaDokumen
	if err := db.Read.WithContext(ctx).Model(&models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&models.MediaBrandDataSuratKerjasamaDokumen{
		ID:          data.IdMediaBrandDataSuratKerjasamaDokumen,
		IdBrandData: id_data_brand,
		Key:         data.KeyDokumen,
	}).Limit(1).Scan(&data_media_brand_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_brand_data_surat_kerjasama_dokumen.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data dokumen tidak ditemukan",
		}
	}

	if err := ms.RemoveObject(ctx, media_storage_database_seeders.BucketDokumenName, data.KeyDokumen, minio.RemoveObjectOptions{}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&models.MediaBrandDataSuratKerjasamaDokumen{
		ID: data_media_brand_data_surat_kerjasama_dokumen.ID,
	}).Delete(&models.MediaBrandDataSuratKerjasamaDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdskd models.MediaBrandDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdskd.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataSuratKerjasamaDokumen); err != nil {
			fmt.Println("Gagal decr media brand data surat kerjasama dokumen ke threshold brand data")
		}

		brandDataSuratKerjasamaDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdskd).SetTableName(Dmbdskd.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSuratKerjasamaDokumenDeletePublish); err != nil {
			fmt.Println("Gagal publish delete brand data surat kerjasama dokumen ke message broker")
		}
	}(data_media_brand_data_surat_kerjasama_dokumen, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaTransaksiApprovedFoto(ctx context.Context, data PayloadTambahMediaTransaksiApprovedFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaTransaksiApprovedFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Select("id").Where(&models.Transaksi{
		ID:     data.IdTransaksi,
		Status: transaksi_enums.Diproses,
	}).Limit(1).Scan(&id_data_transaksi).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_transaksi == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_transaksi_approved_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaTransaksiApprovedFoto{}).Select("id").Where(&models.MediaTransaksiApprovedFoto{
		IdTransaksi: id_data_transaksi,
	}).Limit(1).Scan(&id_media_transaksi_approved_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_transaksi_approved_foto != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	keyz := models.MediaTransaksiApprovedFoto{}.PathName() + strconv.Itoa(int(id_data_transaksi)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	newTransaksiApprovedFoto := models.MediaTransaksiApprovedFoto{
		IdTransaksi: id_data_transaksi,
		Key:         keyz,
		Format:      data.Ekstensi,
	}

	if err := db.Write.WithContext(ctx).Create(&newTransaksiApprovedFoto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	go func(Mtaf models.MediaTransaksiApprovedFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdTransaksi := sot_threshold.TransaksiThreshold{
			IdTransaksi: Mtaf.IdTransaksi,
		}

		if err := thresholdTransaksi.Increment(konteks, Trh, stsk_transaksi.MediaTransaksiApprovedFoto); err != nil {
			fmt.Println("Gagal incr count media transaksi approved foto ke threshold transaksi")
		}

		transaksiApprovedFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mtaf).SetTableName(Mtaf.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiApprovedFotoCreatePublish); err != nil {
			fmt.Println("Gagal publish transaksi approved foto create ke message broker")
		}
	}(newTransaksiApprovedFoto, db.Write, cud_publisher)

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func TambahTransaksiApprovedVideo(ctx context.Context, data PayloadTambahMediaTransaksiApprovedVideo, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahTransaksiApprovedVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Select("id").Where(&models.Transaksi{
		ID:     data.IdTransaksi,
		Status: transaksi_enums.Diproses,
	}).Limit(1).Scan(&id_data_transaksi).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_data_transaksi == 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_media_transaksi_approved_video int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.MediaTransaksiApprovedVideo{}).Select("id").Where(&models.MediaTransaksiApprovedVideo{
		IdTransaksi: id_data_transaksi,
	}).Limit(1).Scan(&id_media_transaksi_approved_video).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_transaksi_approved_video != 0 {
		return &response.ResponseMediaUpload{
			Status:   http.StatusUnauthorized,
			Services: services,
		}
	}

	keyz := models.MediaTransaksiApprovedVideo{}.PathName() + strconv.Itoa(int(id_data_transaksi)) + "/" + helper.GenerateMediaKeyVideo() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketVideoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	newTransaksiApprovedVideo := models.MediaTransaksiApprovedVideo{
		IdTransaksi: id_data_transaksi,
		Key:         keyz,
		Format:      data.Ekstensi,
	}

	if err := db.Write.WithContext(ctx).Create(&newTransaksiApprovedVideo).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	go func(Mtav models.MediaTransaksiApprovedVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdTransaksi := sot_threshold.TransaksiThreshold{
			IdTransaksi: Mtav.IdTransaksi,
		}

		if err := thresholdTransaksi.Increment(konteks, Trh, stsk_transaksi.MediaTransaksiApprovedVideo); err != nil {
			fmt.Println("Gagal incr count media transaksi approved video ke threshold transaksi")
		}

		transaksiApprovedVideoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mtav).SetTableName(Mtav.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, transaksiApprovedVideoCreatePublish); err != nil {
			fmt.Println("Gagal publish transaksi approved video create ke message broker")
		}
	}(newTransaksiApprovedVideo, db.Write, cud_publisher)

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}
