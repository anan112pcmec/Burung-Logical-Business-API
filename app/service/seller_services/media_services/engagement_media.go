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

	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/media_ekstension"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_brand_data "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/brand_data"
	stsk_distributor_data "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/distributor_data"
	stsk_etalase "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/etalase"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	stsk_transaksi "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

func UbahFotoProfilSeller(ctx context.Context, data PayloadUbahFotoProfilSeller, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "UbahFotoProfilSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_profil_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaSellerProfilFoto{}).Select("id").Where(&sot_models.MediaSellerProfilFoto{
		IdSeller: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&id_data_profil_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaSellerProfilFoto{}.PathName() +
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
		newPhotoProfil := sot_models.MediaSellerProfilFoto{
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

		go func(Pp sot_models.MediaSellerProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: Pp.IdSeller,
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.MediaSellerFotoProfil); err != nil {
				fmt.Println("Gagal incr count media seller profil foto ke threshold seller")
			}

			photoProfilCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Pp).SetTableName(Pp.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, photoProfilCreatePublish); err != nil {
				fmt.Println("Gagal publish profil foto create ke message broker")
			}
		}(newPhotoProfil, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).
			Model(&sot_models.MediaSellerProfilFoto{}).
			Where(&sot_models.MediaSellerProfilFoto{
				ID: id_data_profil_foto,
			}).
			Updates(&sot_models.MediaSellerProfilFoto{
				Key:    keyz,
				Format: data.Ekstensi,
			}).Error; err != nil {

			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdPp int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataPhotoProfilUpdated sot_models.MediaSellerProfilFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaSellerProfilFoto{}).Where(&sot_models.MediaSellerProfilFoto{
				ID: IdPp,
			}).Limit(1).Take(&dataPhotoProfilUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data photo profil seller")
			}

			photoProfilUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPhotoProfilUpdated).SetTableName(dataPhotoProfilUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusFotoProfilSeller(ctx context.Context, data PayloadHapusFotoProfilSeller, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusFotoProfilSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_foto_profil_seller sot_models.MediaSellerProfilFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaSellerProfilFoto{}).Where(&sot_models.MediaSellerProfilFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaSellerProfilFoto{}).Where(&sot_models.MediaSellerProfilFoto{
		ID: data.IdMediaSeller,
	}).Delete(&sot_models.MediaSellerProfilFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmfps sot_models.MediaSellerProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: Dmfps.IdSeller,
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.MediaSellerFotoProfil); err != nil {
			fmt.Println("Gagal decr photo profil foto ke treshold selller")
		}

		profilPhotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmfps).SetTableName(Dmfps.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, profilPhotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete profil foto seller ke message broker")
		}
	}(data_media_foto_profil_seller, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
	}
}

func UbahFotoBannerSeller(ctx context.Context, data PayloadUbahFotoBannerSeller, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "UbahFotoBannerSeller"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaSellerBannerFoto{}).Select("id").Where(&sot_models.MediaSellerBannerFoto{
		IdSeller: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Scan(&IdDataBanner).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaSellerBannerFoto{}.PathName() +
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
		newBannerFoto := sot_models.MediaSellerBannerFoto{
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

		go func(Bf sot_models.MediaSellerBannerFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: Bf.IdSeller,
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.MediaSellerBannerFoto); err != nil {
				fmt.Println("Gagal incr count media seller banner foto ke threshold seller")
			}

			bannerFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Bf).SetTableName(Bf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, bannerFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish banner foto create ke message broker")
			}
		}(newBannerFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaSellerBannerFoto{}).Where(&sot_models.MediaSellerBannerFoto{
			ID: IdDataBanner,
		}).Updates(&sot_models.MediaSellerBannerFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdBf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBannerFotoUpdated sot_models.MediaSellerBannerFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaSellerBannerFoto{}).Where(&sot_models.MediaSellerBannerFoto{
				ID: IdBf,
			}).Limit(1).Take(&dataBannerFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data banner foto seller")
			}

			bannerFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBannerFotoUpdated).SetTableName(dataBannerFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusFotoBannerSeller(ctx context.Context, data PayloadHapusFotoBannerSeller, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusFotoBannerSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_banner_foto_seller sot_models.MediaSellerBannerFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaSellerBannerFoto{}).Where(&sot_models.MediaSellerBannerFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaSellerBannerFoto{}).Where(&sot_models.MediaSellerBannerFoto{
		ID: data.IdMediaBannerSellerFoto,
	}).Delete(&sot_models.MediaSellerBannerFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal menghapus coba lagi",
		}
	}

	go func(Dmbfs sot_models.MediaSellerBannerFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: Dmbfs.IdSeller,
		}

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.MediaSellerBannerFoto); err != nil {
			fmt.Println("Gagal decr banner foto ke threshold seller")
		}

		bannerFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbfs).SetTableName(Dmbfs.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahkanFotoTokoFisikSeller(ctx context.Context, data PayloadTambahkanFotoTokoFisikSeller, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahkanFotoTokoFisikSeller"

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

	var DataFotoSave []sot_models.MediaSellerTokoFisikFoto = make([]sot_models.MediaSellerTokoFisikFoto, 0, totalData)
	var DataUrlAndKeyResponse []response.UrlAndKey = make([]response.UrlAndKey, 0, totalData)

	for i := 0; i < totalData; i++ {
		ext := strings.ToLower(data.Ekstensi[i])
		if !media_ekstension.PhotoValidExt[ext] {
			return &response.ResponseMediaUpload{
				Status:   http.StatusBadRequest,
				Services: services,
			}
		}

		keyz := sot_models.MediaSellerTokoFisikFoto{}.PathName() +
			strconv.Itoa(int(data.IdentitasSeller.IdSeller)) + "/" +
			helper.GenerateMediaKeyPhoto() + "." + ext

		url, errPPO := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*10)
		if errPPO != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		DataFotoSave = append(DataFotoSave, sot_models.MediaSellerTokoFisikFoto{
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

	go func(Tffs []sot_models.MediaSellerTokoFisikFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if len(Tffs) > 0 {
			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(Tffs[0].IdSeller),
			}

			if err := thresholdSeller.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{FieldName: stsk_seller.MediaSellerTokoFisikFoto, Count: len(Tffs)}}); err != nil {
				fmt.Println("Gagal incr count media seller toko fisik foto ke threshold seller")
			}

			for _, foto := range Tffs {
				tokoFisikFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusFotoTokoFisikSeller(ctx context.Context, data PayloadHapusFotoTokoFisikSeller, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusFotoTokoFisikSeller"
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
	var dataFotoHapus []sot_models.MediaSellerTokoFisikFoto = make([]sot_models.MediaSellerTokoFisikFoto, 0, dataTotal)

	for i := 0; i < dataTotal; i++ {
		wg.Add(1)
		go func(angka int) {
			defer wg.Done()
			var data_media_foto_toko_fisik sot_models.MediaSellerTokoFisikFoto
			if err := db.Read.WithContext(ctx).Model(&sot_models.MediaSellerTokoFisikFoto{}).Where(&sot_models.MediaSellerTokoFisikFoto{
				ID:  data.DataMediaFotoTokoFisik[angka].IdMedia,
				Key: data.DataMediaFotoTokoFisik[angka].KeyMedia,
			}).Limit(1).Scan(&data_media_foto_toko_fisik).Error; err != nil {
				errChan <- fmt.Errorf("gagal mendapatkan id media dan key media ")
				return
			}

			if data_media_foto_toko_fisik.ID == 0 {
				errChan <- fmt.Errorf("gagal mendapatkan id media")
				return
			}

			ctxRemove, cancel := context.WithTimeout(ctx, settings.TimeoutContext)
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaSellerTokoFisikFoto{}).Where("id IN ?", idsHapus).Delete(&sot_models.MediaSellerTokoFisikFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dtfh []sot_models.MediaSellerTokoFisikFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if len(Dtfh) > 0 {
			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(Dtfh[0].IdSeller),
			}

			if err := thresholdSeller.CustomDecrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{FieldName: stsk_seller.MediaSellerTokoFisikFoto, Count: len(Dtfh)}}); err != nil {
				fmt.Println("Gagal decr foto toko fisik ke threshold seller")
			}

			for _, foto := range Dtfh {
				tokoFisikFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName()).SetRole(mb_cud_seeders.Seller)
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

func UbahFotoEtalaseSeller(ctx context.Context, data PayloadUbahFotoEtalase, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "UbahFotoEtalaseSeller"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.Etalase{}).Select("id").Where(&sot_models.Etalase{
		ID:       data.IdEtalase,
		SellerID: int64(data.IdentitasSeller.IdSeller),
	}).Limit(1).Take(&id_data_etalase).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_media_etalase_foto int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaEtalaseFoto{}).Select("id").Where(&sot_models.MediaEtalaseFoto{
		IdEtalase: id_data_etalase,
	}).Limit(1).Scan(&id_data_media_etalase_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaEtalaseFoto{}.PathName() + strconv.Itoa(int(id_data_etalase)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrl := url.String()

	if id_data_media_etalase_foto == 0 {
		newEtalaseFoto := sot_models.MediaEtalaseFoto{
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

		go func(Ef sot_models.MediaEtalaseFoto, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			etalaseFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ef).SetTableName(Ef.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish etalase foto create ke message broker")
			}
		}(newEtalaseFoto, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaEtalaseFoto{}).Where(&sot_models.MediaEtalaseFoto{
			ID: id_data_media_etalase_foto,
		}).Updates(&sot_models.MediaEtalaseFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdEf int64, Trh *gorm.DB, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataEtalaseFotoUpdated sot_models.MediaEtalaseFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaEtalaseFoto{}).Where(&sot_models.MediaEtalaseFoto{
				ID: IdEf,
			}).Limit(1).Take(&dataEtalaseFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data etalase foto")
			}

			var ThresEtalase sot_threshold.EtalaseThreshold = sot_threshold.EtalaseThreshold{
				ID: dataEtalaseFotoUpdated.IdEtalase,
			}

			if err := ThresEtalase.Increment(konteks, Trh, stsk_etalase.MediaEtalaseFoto); err != nil {
				fmt.Println("gagal increment data threshold etalase seller", err)
			}

			etalaseFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataEtalaseFotoUpdated).SetTableName(dataEtalaseFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, etalaseFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update etalase foto ke message broker")
			}
		}(id_data_media_etalase_foto, db.Write, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUrl,
		Key:       keyz,
	}
}

func HapusFotoEtalaseSeller(ctx context.Context, data PayloadHapusFotoEtalase, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusFotoEtalaseSeller"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_etalase_foto sot_models.MediaEtalaseFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaEtalaseFoto{}).Where(&sot_models.MediaEtalaseFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaEtalaseFoto{}).Where(&sot_models.MediaEtalaseFoto{
		ID: data_media_etalase_foto.ID,
	}).Delete(&sot_models.MediaEtalaseFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk, coba lagi lain waktu",
		}
	}

	go func(Dmef sot_models.MediaEtalaseFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var ThresEtalase sot_threshold.EtalaseThreshold = sot_threshold.EtalaseThreshold{
			ID: Dmef.IdEtalase,
		}

		if err := ThresEtalase.Decrement(konteks, Trh, stsk_etalase.MediaEtalaseFoto); err != nil {
			fmt.Println("gagall decrement threshold etalase", err)
		}

		etalaseFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmef).SetTableName(Dmef.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahkanMediaBarangIndukFoto(ctx context.Context, data PayloadTambahBarangIndukFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahkanMediaBarangIndukFoto"
	const Limit uint8 = 10

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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

	var DataMediaBarangIndukFoto []sot_models.MediaBarangIndukFoto = make([]sot_models.MediaBarangIndukFoto, 0, dataLength)
	var DataUrlAndKeyUpload []response.UrlAndKey = make([]response.UrlAndKey, 0, dataLength)

	var idsDataMediaBarangInduk []int64
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBarangIndukFoto{}).Select("id").Where(&sot_models.MediaBarangIndukFoto{
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

		keyz := sot_models.MediaBarangIndukFoto{}.PathName() + strconv.Itoa(int(id_data_barang_induk)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi[i]

		url, err := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
		if err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		DataMediaBarangIndukFoto = append(DataMediaBarangIndukFoto, sot_models.MediaBarangIndukFoto{
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

	go func(Mbif []sot_models.MediaBarangIndukFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
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
			barangIndukFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaBarangIndukFoto(ctx context.Context, data PayloadHapusBarangIndukFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaBarangIndukFoto"
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

	var dataMediaBarangIndukFotoHapus []sot_models.MediaBarangIndukFoto = make([]sot_models.MediaBarangIndukFoto, 0, dataLength)
	for i := range dataLength {
		wg.Add(1)
		go func(a int) {
			defer wg.Done()
			var data_media_barang_induk_foto sot_models.MediaBarangIndukFoto
			if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBarangIndukFoto{}).Where(&sot_models.MediaBarangIndukFoto{
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

	var LengthBarangIndukFotoDihapus = len(dataMediaBarangIndukFotoHapus)

	if LengthBarangIndukFotoDihapus == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	var idsHapus []int64 = make([]int64, 0, LengthBarangIndukFotoDihapus)
	for _, foto := range dataMediaBarangIndukFotoHapus {
		idsHapus = append(idsHapus, foto.ID)
	}

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBarangIndukFoto{}).Where("id IN ?", idsHapus).Delete(&sot_models.MediaBarangIndukFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbifh []sot_models.MediaBarangIndukFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Dmbifh[0].IdBarangInduk,
		}

		if err := barangIndukThreshold.CustomDecrement(konteks, Trh, []sot_threshold.CustomCounter{sot_threshold.CustomCounter{
			FieldName: stsk_baranginduk.MediaBarangIndukFoto,
			Count:     len(Dmbifh),
		}}); err != nil {
			fmt.Println("Gagal decr count media barang induk foto threshold Barang induk")
		}

		for _, foto := range Dmbifh {
			barangIndukFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(foto).SetTableName(foto.TableName()).SetRole(mb_cud_seeders.Seller)
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
} //  Tuned

func UbahBarangIndukVideo(ctx context.Context, data PayloadUbahVideoBarangInduk, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "UbahBarangIndukVideo"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBarangIndukVideo{}).Select("id").Where(&sot_models.MediaBarangIndukVideo{
		IdBarangInduk: id_barang_induk,
	}).Limit(1).Scan(&id_media_data_barang_induk_video).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBarangIndukVideo{}.PathName() + strconv.Itoa(int(id_barang_induk)) + "/" + helper.GenerateMediaKeyVideo() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketVideoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUploadUrl := url.String()
	if id_media_data_barang_induk_video == 0 {
		newBarangIndukVideo := sot_models.MediaBarangIndukVideo{
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

		go func(Biv sot_models.MediaBarangIndukVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
				IdBarangInduk: Biv.IdBarangInduk,
			}

			if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.MediaBarangIndukVideo); err != nil {
				fmt.Println("gagal publish create video barang induk ke message broker ")
			}

			barangIndukVideoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Biv).SetTableName(Biv.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, barangIndukVideoCreatePublish); err != nil {
				fmt.Println("Gagal publish barang induk video create ke message broker")
			}
		}(newBarangIndukVideo, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBarangIndukVideo{}).Where(&sot_models.MediaBarangIndukVideo{
			ID: id_media_data_barang_induk_video,
		}).Updates(&sot_models.MediaBarangIndukVideo{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdBiv int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBarangIndukVideoUpdated sot_models.MediaBarangIndukVideo
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBarangIndukVideo{}).Where(&sot_models.MediaBarangIndukVideo{
				ID: IdBiv,
			}).Limit(1).Take(&dataBarangIndukVideoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data barang induk video")
			}

			barangIndukVideoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBarangIndukVideoUpdated).SetTableName(dataBarangIndukVideoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusBarangIndukVideo(ctx context.Context, data PayloadHapusVideoBarangInduk, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusBarangIndukVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_barang_induk_video sot_models.MediaBarangIndukVideo
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBarangIndukVideo{}).Where(&sot_models.MediaBarangIndukVideo{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBarangIndukVideo{}).Where(&sot_models.MediaBarangIndukVideo{
		ID: data.IdMediaBarangIndukVideo,
	}).Delete(&sot_models.MediaBarangIndukVideo{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbiv sot_models.MediaBarangIndukVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: Dmbiv.IdBarangInduk,
		}

		if err := thresholdBarangInduk.Decrement(konteks, Trh, stsk_baranginduk.MediaBarangIndukVideo); err != nil {
			fmt.Println("gagal publish delete video barang induk ke message broker ")
		}

		barangIndukVideoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbiv).SetTableName(Dmbiv.TableName()).SetRole(mb_cud_seeders.Seller)
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

func UbahKategoriBarangFoto(ctx context.Context, data PayloadUbahKategoriBarangFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "UbahKategoriBarangFoto"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaKategoriBarangFoto{}).Select("id").Where(&sot_models.MediaKategoriBarangFoto{
		IdKategoriBarang: data.IdKategoriBarang,
		IdBarangInduk:    data.IdBarangInduk,
	}).Limit(1).Scan(&id_media_kategori_barang_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaKategoriBarangFoto{}.PathName() + strconv.Itoa(int(data.IdKategoriBarang)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, url_err := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if url_err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	minIOUrlUpload := url.String()

	if id_media_kategori_barang_foto == 0 {
		newKategoriBarangFoto := sot_models.MediaKategoriBarangFoto{
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

		go func(Mkbf sot_models.MediaKategoriBarangFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: Mkbf.IdKategoriBarang,
			}

			if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.MediaKategoriBarangFoto); err != nil {
				fmt.Println("Gagal incr count media kategori barang foto ke threshold kategori barang")
			}

			kategoriBarangFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mkbf).SetTableName(Mkbf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish kategori barang foto create ke message broker")
			}
		}(newKategoriBarangFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaKategoriBarangFoto{}).Where(&sot_models.MediaKategoriBarangFoto{
			ID: id_media_kategori_barang_foto,
		}).Updates(&sot_models.MediaKategoriBarangFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMkbf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataKategoriBarangFotoUpdated sot_models.MediaKategoriBarangFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaKategoriBarangFoto{}).Where(&sot_models.MediaKategoriBarangFoto{
				ID: IdMkbf,
			}).Limit(1).Take(&dataKategoriBarangFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data kategori barang foto")
			}

			kategoriBarangFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangFotoUpdated).SetTableName(dataKategoriBarangFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusKategoriBarangFoto(ctx context.Context, data PayloadHapusKategoriBarangFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusKategoriBarangFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var data_media_kategori_barang_foto sot_models.MediaKategoriBarangFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaKategoriBarangFoto{}).Where(&sot_models.MediaKategoriBarangFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaKategoriBarangFoto{}).Where(&sot_models.MediaKategoriBarangFoto{
		ID: data_media_kategori_barang_foto.ID,
	}).Delete(&sot_models.MediaKategoriBarangFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmkbf sot_models.MediaKategoriBarangFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Dmkbf.IdKategoriBarang,
		}

		if err := thresholdKategoriBarang.Decrement(konteks, Trh, stsk_kategori_barang.MediaKategoriBarangFoto); err != nil {
			fmt.Println("Gagal decr media kategori barang foto ke threshold kategori barang")
		}

		kategoriBarangFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmkbf).SetTableName(Dmkbf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahDistributorDataDokumen(ctx context.Context, data PayloadMediaDistributorDataDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahDistributorDataDokumen"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataDokumen{}).Select("id").Where(&sot_models.MediaDistributorDataDokumen{
		IdDistributorData: id_distributor_data,
	}).Limit(1).Scan(&id_media_distributor_data_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaDistributorDataDokumen{}.PathName() + strconv.Itoa(int(data.IdDistributorData)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_dokumen == 0 {
		newDistributorDataDokumen := sot_models.MediaDistributorDataDokumen{
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

		go func(Mddd sot_models.MediaDistributorDataDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddd.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataDokumen); err != nil {
				fmt.Println("Gagal incr count media distributor data dokumen ke threshold distributor data")
			}

			distributorDataDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddd).SetTableName(Mddd.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data dokumen create ke message broker")
			}
		}(newDistributorDataDokumen, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataDokumen{}).Where(&sot_models.MediaDistributorDataDokumen{
			ID: id_media_distributor_data_dokumen,
		}).Updates(&sot_models.MediaDistributorDataDokumen{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMddd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataDistributorDataDokumenUpdated sot_models.MediaDistributorDataDokumen
			if err := Read.WithContext(konteks).Model(&sot_models.MediaDistributorDataDokumen{}).Where(&sot_models.MediaDistributorDataDokumen{
				ID: IdMddd,
			}).Limit(1).Take(&dataDistributorDataDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data dokumen")
			}

			distributorDataDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataDokumenUpdated).SetTableName(dataDistributorDataDokumenUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaDistributorDataDokumen(ctx context.Context, data PayloadHapusMediaDistributorDataDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaDistributorDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_distributor_data int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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

	var data_media_distributor_data_dokumen sot_models.MediaDistributorDataDokumen
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataDokumen{}).Where(&sot_models.MediaDistributorDataDokumen{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataDokumen{}).Where(&sot_models.MediaDistributorDataDokumen{
		ID: data_media_distributor_data_dokumen.ID,
	}).Delete(&sot_models.MediaDistributorDataDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmddd sot_models.MediaDistributorDataDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmddd.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataDokumen); err != nil {
			fmt.Println("Gagal decr media distributor data dokumen ke threshold distributor data")
		}

		distributorDataDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmddd).SetTableName(Dmddd.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahMediaDistributorDataNPWPFoto(ctx context.Context, data PayloadTambahMediaDistributorDataNPWPFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahMediaDistributorDataNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataNPWPFoto{}).Select("id").Where(&sot_models.MediaDistributorDataNPWPFoto{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_npwp_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaDistributorDataNPWPFoto{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_npwp_foto == 0 {
		newDistributorDataNPWPFoto := sot_models.MediaDistributorDataNPWPFoto{
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

		go func(Mddnf sot_models.MediaDistributorDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddnf.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataNpwpFoto); err != nil {
				fmt.Println("Gagal incr count media distributor data npwp foto ke threshold distributor data")
			}

			distributorDataNPWPFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddnf).SetTableName(Mddnf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNPWPFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data npwp foto create ke message broker")
			}
		}(newDistributorDataNPWPFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataNPWPFoto{}).Where(&sot_models.MediaDistributorDataNPWPFoto{
			ID: id_media_distributor_data_npwp_foto,
		}).Updates(&sot_models.MediaDistributorDataNPWPFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMddnf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataDistributorDataNPWPFotoUpdated sot_models.MediaDistributorDataNPWPFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaDistributorDataNPWPFoto{}).Where(&sot_models.MediaDistributorDataNPWPFoto{
				ID: IdMddnf,
			}).Limit(1).Take(&dataDistributorDataNPWPFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data npwp foto")
			}

			distributorDataNPWPFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataNPWPFotoUpdated).SetTableName(dataDistributorDataNPWPFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaDistributorDataNPWPFoto(ctx context.Context, data PayloadHapusMediaDistributorDataNPWPFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaDistributorDataNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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

	var data_media_distributor_npwp_foto sot_models.MediaDistributorDataNPWPFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataNPWPFoto{}).Where(&sot_models.MediaDistributorDataNPWPFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataNPWPFoto{}).Where(&sot_models.MediaDistributorDataNPWPFoto{
		ID: data_media_distributor_npwp_foto.ID,
	}).Delete(&sot_models.MediaDistributorDataNPWPFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmdnf sot_models.MediaDistributorDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmdnf.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataNpwpFoto); err != nil {
			fmt.Println("Gagal decr media distributor data npwp foto ke threshold distributor data")
		}

		distributorDataNPWPFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmdnf).SetTableName(Dmdnf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahDistributorDataNIBFoto(ctx context.Context, data PayloadTambahDistributorDataNIBFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahDistributorDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataNIBFoto{}).Select("id").Where(&sot_models.MediaDistributorDataNIBFoto{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_nib_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaDistributorDataNIBFoto{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_nib_foto == 0 {
		newDistributorDataNIBFoto := sot_models.MediaDistributorDataNIBFoto{
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

		go func(Mddnibf sot_models.MediaDistributorDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddnibf.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataNibFoto); err != nil {
				fmt.Println("Gagal incr count media distributor data nib foto ke threshold distributor data")
			}

			distributorDataNIBFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddnibf).SetTableName(Mddnibf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataNIBFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data nib foto create ke message broker")
			}
		}(newDistributorDataNIBFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataNIBFoto{}).Where(&sot_models.MediaDistributorDataNIBFoto{
			ID: id_media_distributor_data_nib_foto,
		}).Updates(&sot_models.MediaDistributorDataNIBFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMddnibf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataDistributorDataNIBFotoUpdated sot_models.MediaDistributorDataNIBFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaDistributorDataNIBFoto{}).Where(&sot_models.MediaDistributorDataNIBFoto{
				ID: IdMddnibf,
			}).Limit(1).Take(&dataDistributorDataNIBFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data nib foto")
			}
			distributorDataNIBFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataNIBFotoUpdated).SetTableName(dataDistributorDataNIBFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusDistributorDataNIBFoto(ctx context.Context, data PayloadHapusDistributorDataNIBFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusDistributorDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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

	var data_media_distributor_nib_foto sot_models.MediaDistributorDataNIBFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataNIBFoto{}).Where(&sot_models.MediaDistributorDataNIBFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataNIBFoto{}).Where(&sot_models.MediaDistributorDataNIBFoto{
		ID: data_media_distributor_nib_foto.ID,
	}).Delete(&sot_models.MediaDistributorDataNIBFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmdnibf sot_models.MediaDistributorDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmdnibf.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataNibFoto); err != nil {
			fmt.Println("Gagal decr media distributor data nib foto ke threshold distributor data")
		}

		distributorDataNIBFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmdnibf).SetTableName(Dmdnibf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahDistributorDataSuratKerjasamaDokumen(ctx context.Context, data PayloadTambahDistributorDataSuratKerjasamaDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahDistributorDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{}).Select("id").Where(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{
		IdDistributorData: id_data_distributor,
	}).Limit(1).Scan(&id_media_distributor_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaDistributorDataSuratKerjasamaDokumen{}.PathName() + strconv.Itoa(int(id_data_distributor)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)
	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_distributor_data_surat_kerjasama_dokumen == 0 {
		newDistributorDataSuratKerjasamaDokumen := sot_models.MediaDistributorDataSuratKerjasamaDokumen{
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

		go func(Mddskd sot_models.MediaDistributorDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdDistributorData := sot_threshold.DistributorDataThreshold{
				IdDistributorData: Mddskd.IdDistributorData,
			}

			if err := thresholdDistributorData.Increment(konteks, Trh, stsk_distributor_data.MediaDistributorDataSuratKerjasamaDokumen); err != nil {
				fmt.Println("Gagal incr count media distributor data surat kerjasama dokumen ke threshold distributor data")
			}

			distributorDataSuratKerjasamaDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mddskd).SetTableName(Mddskd.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, distributorDataSuratKerjasamaDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish distributor data surat kerjasama dokumen create ke message broker")
			}
		}(newDistributorDataSuratKerjasamaDokumen, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{
			ID: id_media_distributor_data_surat_kerjasama_dokumen,
		}).Updates(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMddskd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataDistributorDataSuratKerjasamaDokumenUpdated sot_models.MediaDistributorDataSuratKerjasamaDokumen
			if err := Read.WithContext(konteks).Model(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{
				ID: IdMddskd,
			}).Limit(1).Take(&dataDistributorDataSuratKerjasamaDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data distributor data surat kerjasama dokumen")
			}

			distributorDataSuratKerjasamaDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataDistributorDataSuratKerjasamaDokumenUpdated).SetTableName(dataDistributorDataSuratKerjasamaDokumenUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusDistributorDataSuratKerjasamaDataDokumen(ctx context.Context, data PayloadHapusDistributorDataSuratKerjasamaDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusDistributorDataSuratKerjasamaDataDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_distributor int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.DistributorData{}).Select("id").Where(&sot_models.DistributorData{
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

	var data_media_distributor_data_surat_kerjasama_dokumen sot_models.MediaDistributorDataSuratKerjasamaDokumen
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{
		ID: data_media_distributor_data_surat_kerjasama_dokumen.ID,
	}).Delete(&sot_models.MediaDistributorDataSuratKerjasamaDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmdskd sot_models.MediaDistributorDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdDistributorData := sot_threshold.DistributorDataThreshold{
			IdDistributorData: Dmdskd.IdDistributorData,
		}

		if err := thresholdDistributorData.Decrement(konteks, Trh, stsk_distributor_data.MediaDistributorDataSuratKerjasamaDokumen); err != nil {
			fmt.Println("Gagal decr media distributor data surat kerjasama dokumen ke threshold distributor data")
		}

		distributorDataSuratKerjasamaDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmdskd).SetTableName(Dmdskd.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahBrandDataPerwakilanDokumen(ctx context.Context, data PayloadTambahBrandDataPerwakilanDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahBrandDataPerwakilanDokumen"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataPerwakilanDokumen{}).Select("id").Where(&sot_models.MediaBrandDataPerwakilanDokumen{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_perwakilan_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBrandDataPerwakilanDokumen{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_perwakilan_dokumen == 0 {
		newBrandDataPerwakilanDokumen := sot_models.MediaBrandDataPerwakilanDokumen{
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

		go func(Mbdpd sot_models.MediaBrandDataPerwakilanDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdpd.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataPerwakilanDokumen); err != nil {
				fmt.Println("Gagal incr count media brand data perwakilan dokumen ke threshold brand data")
			}

			brandDataPerwakilanDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdpd).SetTableName(Mbdpd.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataPerwakilanDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data perwakilan dokumen create ke message broker")
			}
		}(newBrandDataPerwakilanDokumen, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataPerwakilanDokumen{}).Where(&sot_models.MediaBrandDataPerwakilanDokumen{
			ID: id_media_brand_data_perwakilan_dokumen,
		}).Updates(&sot_models.MediaBrandDataPerwakilanDokumen{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdpd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBrandDataPerwakilanDokumenUpdated sot_models.MediaBrandDataPerwakilanDokumen
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBrandDataPerwakilanDokumen{}).Where(&sot_models.MediaBrandDataPerwakilanDokumen{
				ID: IdMbdpd,
			}).Limit(1).Take(&dataBrandDataPerwakilanDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data perwakilan dokumen")
			}

			brandDataPerwakilanDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataPerwakilanDokumenUpdated).SetTableName(dataBrandDataPerwakilanDokumenUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaBrandDataPerwakilanDokumen(ctx context.Context, data PayloadHapusBrandDataPerwakilanDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaBrandDataPerwakilanDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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

	var data_media_brand_data_perwakilan_dokumen sot_models.MediaBrandDataPerwakilanDokumen
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataPerwakilanDokumen{}).Where(&sot_models.MediaBrandDataPerwakilanDokumen{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataPerwakilanDokumen{}).Where(&sot_models.MediaBrandDataPerwakilanDokumen{
		ID: data_media_brand_data_perwakilan_dokumen.ID,
	}).Delete(&sot_models.MediaBrandDataPerwakilanDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdpd sot_models.MediaBrandDataPerwakilanDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdpd.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataPerwakilanDokumen); err != nil {
			fmt.Println("Gagal decr media brand data perwakilan dokumen ke threshold brand data")
		}

		brandDataPerwakilanDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdpd).SetTableName(Dmbdpd.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahMediaBrandDataSertifikatFoto(ctx context.Context, data PayloadTambahBrandDataSertifikatFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahMediaBrandDataSertifikatFoto"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataSertifikatFoto{}).Select("id").Where(&sot_models.MediaBrandDataSertifikatFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_sertifikat_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBrandDataSertifikatFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_sertifikat_foto == 0 {
		newBrandDataSertifikatFoto := sot_models.MediaBrandDataSertifikatFoto{
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

		go func(Mbdsf sot_models.MediaBrandDataSertifikatFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdsf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataSertifikatFoto); err != nil {
				fmt.Println("Gagal incr count media brand data sertifikat foto ke threshold brand data")
			}

			brandDataSertifikatFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdsf).SetTableName(Mbdsf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSertifikatFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data sertifikat foto create ke message broker")
			}
		}(newBrandDataSertifikatFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataSertifikatFoto{}).Where(&sot_models.MediaBrandDataSertifikatFoto{
			ID: id_media_brand_data_sertifikat_foto,
		}).Updates(&sot_models.MediaBrandDataSertifikatFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdsf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBrandDataSertifikatFotoUpdated sot_models.MediaBrandDataSertifikatFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBrandDataSertifikatFoto{}).Where(&sot_models.MediaBrandDataSertifikatFoto{
				ID: IdMbdsf,
			}).Limit(1).Take(&dataBrandDataSertifikatFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data sertifikat foto")
			}

			brandDataSertifikatFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataSertifikatFotoUpdated).SetTableName(dataBrandDataSertifikatFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaBrandDataSertifikatFoto(ctx context.Context, data PayloadHapusBrandDataSertifikatFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaBrandDataSertifikatFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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

	var data_media_brand_data_sertifikat_foto sot_models.MediaBrandDataSertifikatFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataSertifikatFoto{}).Where(&sot_models.MediaBrandDataSertifikatFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataSertifikatFoto{}).Where(&sot_models.MediaBrandDataSertifikatFoto{
		ID: data_media_brand_data_sertifikat_foto.ID,
	}).Delete(&sot_models.MediaBrandDataSertifikatFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdsf sot_models.MediaBrandDataSertifikatFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdsf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataSertifikatFoto); err != nil {
			fmt.Println("Gagal decr media brand data sertifikat foto ke threshold brand data")
		}

		brandDataSertifikatFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdsf).SetTableName(Dmbdsf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahMediaBrandDataNIBFoto(ctx context.Context, data PayloadTambahMediaBrandDataNIBFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahMediaBrandDataNIBFoto"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataNIBFoto{}).Select("id").Where(&sot_models.MediaBrandDataNIBFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_nib_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBrandDataNIBFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_nib_foto == 0 {
		newBrandDataNIBFoto := sot_models.MediaBrandDataNIBFoto{
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

		go func(Mbdnibf sot_models.MediaBrandDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdnibf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataNibFoto); err != nil {
				fmt.Println("Gagal incr count media brand data nib foto ke threshold brand data")
			}

			brandDataNIBFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdnibf).SetTableName(Mbdnibf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNIBFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data nib foto create ke message broker")
			}
		}(newBrandDataNIBFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataNIBFoto{}).Where(&sot_models.MediaBrandDataNIBFoto{
			ID: id_media_brand_data_nib_foto,
		}).Updates(&sot_models.MediaBrandDataNIBFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdnibf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBrandDataNIBFotoUpdated sot_models.MediaBrandDataNIBFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBrandDataNIBFoto{}).Where(&sot_models.MediaBrandDataNIBFoto{
				ID: IdMbdnibf,
			}).Limit(1).Take(&dataBrandDataNIBFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data nib foto")
			}

			brandDataNIBFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataNIBFotoUpdated).SetTableName(dataBrandDataNIBFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaBrandDataNIBFoto(ctx context.Context, data PayloadHapusMediaBrandDataNIBFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaBrandDataNIBFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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

	var data_media_brand_data_nib_foto sot_models.MediaBrandDataNIBFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataNIBFoto{}).Where(&sot_models.MediaBrandDataNIBFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataNIBFoto{}).Where(&sot_models.MediaBrandDataNIBFoto{
		ID: data_media_brand_data_nib_foto.ID,
	}).Delete(&sot_models.MediaBrandDataNIBFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdnibf sot_models.MediaBrandDataNIBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdnibf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataNibFoto); err != nil {
			fmt.Println("Gagal decr media brand data nib foto ke threshold brand data")
		}

		brandDataNIBFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdnibf).SetTableName(Dmbdnibf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahMediaBrandNPWPFoto(ctx context.Context, data PayloadTambahMediaBrandDataNPWPFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahMediaBrandNPWPFoto"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataNPWPFoto{}).Select("id").Where(&sot_models.MediaBrandDataNPWPFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_npwp_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBrandDataNPWPFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_npwp_foto == 0 {
		newBrandDataNPWPFoto := sot_models.MediaBrandDataNPWPFoto{
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

		go func(Mbdnf sot_models.MediaBrandDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {

			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdnf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataNpwpFoto); err != nil {
				fmt.Println("Gagal incr count media brand data npwp foto ke threshold brand data")
			}

			brandDataNPWPFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdnf).SetTableName(Mbdnf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataNPWPFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data npwp foto create ke message broker")
			}
		}(newBrandDataNPWPFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataNPWPFoto{}).Where(&sot_models.MediaBrandDataNPWPFoto{
			ID: id_media_brand_data_npwp_foto,
		}).Updates(&sot_models.MediaBrandDataNPWPFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdnf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBrandDataNPWPFotoUpdated sot_models.MediaBrandDataNPWPFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBrandDataNPWPFoto{}).Where(&sot_models.MediaBrandDataNPWPFoto{
				ID: IdMbdnf,
			}).Limit(1).Take(&dataBrandDataNPWPFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data npwp foto")
			}

			brandDataNPWPFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataNPWPFotoUpdated).SetTableName(dataBrandDataNPWPFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaBrandNPWPFoto(ctx context.Context, data PayloadHapusMediaBrandDataNPWPFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaBrandNPWPFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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

	var data_media_brand_data_npwp_foto sot_models.MediaBrandDataNPWPFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataNPWPFoto{}).Where(&sot_models.MediaBrandDataNPWPFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataNPWPFoto{}).Where(&sot_models.MediaBrandDataNPWPFoto{
		ID: data_media_brand_data_npwp_foto.ID,
	}).Delete(&sot_models.MediaBrandDataNPWPFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdnf sot_models.MediaBrandDataNPWPFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdnf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataNpwpFoto); err != nil {
			fmt.Println("Gagal decr media brand data npwp foto ke threshold brand data")
		}

		brandDataNPWPFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdnf).SetTableName(Dmbdnf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahMediaBrandDataLogoFoto(ctx context.Context, data PayloadTambahMediaBrandDataLogoFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahMediaBrandDataLogoBrandFoto"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataLogoFoto{}).Select("id").Where(&sot_models.MediaBrandDataLogoFoto{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_logo_foto).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBrandDataLogoFoto{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_logo_foto == 0 {
		newBrandDataLogoFoto := sot_models.MediaBrandDataLogoFoto{
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

		go func(Mbdlf sot_models.MediaBrandDataLogoFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdlf.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataLogoBrandFoto); err != nil {
				fmt.Println("Gagal incr count media brand data logo foto ke threshold brand data")
			}

			brandDataLogoFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdlf).SetTableName(Mbdlf.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataLogoFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data logo foto create ke message broker")
			}
		}(newBrandDataLogoFoto, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataLogoFoto{}).Where(&sot_models.MediaBrandDataLogoFoto{
			ID: id_media_brand_data_logo_foto,
		}).Updates(&sot_models.MediaBrandDataLogoFoto{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdlf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBrandDataLogoFotoUpdated sot_models.MediaBrandDataLogoFoto
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBrandDataLogoFoto{}).Where(&sot_models.MediaBrandDataLogoFoto{
				ID: IdMbdlf,
			}).Limit(1).Take(&dataBrandDataLogoFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data logo foto")
			}

			brandDataLogoFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataLogoFotoUpdated).SetTableName(dataBrandDataLogoFotoUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusMediaBrandDataLogo(ctx context.Context, data PayloadHapusMediaBrandDataLogoFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusMediaBrandDataLogo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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

	var data_media_brand_data_logo_foto sot_models.MediaBrandDataLogoFoto
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataLogoFoto{}).Where(&sot_models.MediaBrandDataLogoFoto{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataLogoFoto{}).Where(&sot_models.MediaBrandDataLogoFoto{
		ID: data_media_brand_data_logo_foto.ID,
	}).Delete(&sot_models.MediaBrandDataLogoFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdlf sot_models.MediaBrandDataLogoFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdlf.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataLogoBrandFoto); err != nil {
			fmt.Println("Gagal decr media brand data logo foto ke threshold brand data")
		}

		brandDataLogoFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdlf).SetTableName(Dmbdlf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahBrandDataSuratKerjasamaDokumen(ctx context.Context, data PayloadTambahBrandDataSuratKerjasamaDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahBrandDataSuratKerjasamaDokumen"

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataSuratKerjasamaDokumen{}).Select("id").Where(&sot_models.MediaBrandDataSuratKerjasamaDokumen{
		IdBrandData: id_data_brand,
	}).Limit(1).Scan(&id_media_brand_data_surat_kerjasama_dokumen).Error; err != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	keyz := sot_models.MediaBrandDataSuratKerjasamaDokumen{}.PathName() + strconv.Itoa(int(id_data_brand)) + "/" + helper.GenerateMediaKeyDokumen() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketDokumenName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	if id_media_brand_data_surat_kerjasama_dokumen == 0 {
		newBrandDataSuratKerjasamaDokumen := sot_models.MediaBrandDataSuratKerjasamaDokumen{
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

		go func(Mbdskd sot_models.MediaBrandDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			thresholdBrandData := sot_threshold.BrandDataThreshold{
				IdBrandData: Mbdskd.IdBrandData,
			}

			if err := thresholdBrandData.Increment(konteks, Trh, stsk_brand_data.MediaBrandDataSuratKerjasamaDokumen); err != nil {
				fmt.Println("Gagal incr count media brand data surat kerjasama dokumen ke threshold brand data")
			}

			brandDataSuratKerjasamaDokumenCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mbdskd).SetTableName(Mbdskd.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, brandDataSuratKerjasamaDokumenCreatePublish); err != nil {
				fmt.Println("Gagal publish brand data surat kerjasama dokumen create ke message broker")
			}
		}(newBrandDataSuratKerjasamaDokumen, db.Write, cud_publisher)
	} else {
		if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaBrandDataSuratKerjasamaDokumen{
			ID: id_media_brand_data_surat_kerjasama_dokumen,
		}).Updates(&sot_models.MediaBrandDataSuratKerjasamaDokumen{
			Key:    keyz,
			Format: data.Ekstensi,
		}).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(IdMbdskd int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			var dataBrandDataSuratKerjasamaDokumenUpdated sot_models.MediaBrandDataSuratKerjasamaDokumen
			if err := Read.WithContext(konteks).Model(&sot_models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaBrandDataSuratKerjasamaDokumen{
				ID: IdMbdskd,
			}).Limit(1).Take(&dataBrandDataSuratKerjasamaDokumenUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data brand data surat kerjasama dokumen")
			}

			brandDataSuratKerjasamaDokumenUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBrandDataSuratKerjasamaDokumenUpdated).SetTableName(dataBrandDataSuratKerjasamaDokumenUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusBrandDataSuratKerjasamaDokumen(ctx context.Context, data PayloadHapusBrandDataSuratKerjasamaDokumen, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusBrandDataSuratKerjasamaDokumen"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var id_data_brand int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BrandData{}).Select("id").Where(&sot_models.BrandData{
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

	var data_media_brand_data_surat_kerjasama_dokumen sot_models.MediaBrandDataSuratKerjasamaDokumen
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaBrandDataSuratKerjasamaDokumen{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.MediaBrandDataSuratKerjasamaDokumen{}).Where(&sot_models.MediaBrandDataSuratKerjasamaDokumen{
		ID: data_media_brand_data_surat_kerjasama_dokumen.ID,
	}).Delete(&sot_models.MediaBrandDataSuratKerjasamaDokumen{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmbdskd sot_models.MediaBrandDataSuratKerjasamaDokumen, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBrandData := sot_threshold.BrandDataThreshold{
			IdBrandData: Dmbdskd.IdBrandData,
		}

		if err := thresholdBrandData.Decrement(konteks, Trh, stsk_brand_data.MediaBrandDataSuratKerjasamaDokumen); err != nil {
			fmt.Println("Gagal decr media brand data surat kerjasama dokumen ke threshold brand data")
		}

		brandDataSuratKerjasamaDokumenDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmbdskd).SetTableName(Dmbdskd.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahMediaTransaksiApprovedFoto(ctx context.Context, data PayloadTambahMediaTransaksiApprovedFoto, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahMediaTransaksiApprovedFoto"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Transaksi{}).Select("id").Where(&sot_models.Transaksi{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaTransaksiApprovedFoto{}).Select("id").Where(&sot_models.MediaTransaksiApprovedFoto{
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

	keyz := sot_models.MediaTransaksiApprovedFoto{}.PathName() + strconv.Itoa(int(id_data_transaksi)) + "/" + helper.GenerateMediaKeyPhoto() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketFotoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	newTransaksiApprovedFoto := sot_models.MediaTransaksiApprovedFoto{
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

	go func(Mtaf sot_models.MediaTransaksiApprovedFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdTransaksi := sot_threshold.TransaksiThreshold{
			IdTransaksi: Mtaf.IdTransaksi,
		}

		if err := thresholdTransaksi.Increment(konteks, Trh, stsk_transaksi.MediaTransaksiApprovedFoto); err != nil {
			fmt.Println("Gagal incr count media transaksi approved foto ke threshold transaksi")
		}

		transaksiApprovedFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mtaf).SetTableName(Mtaf.TableName()).SetRole(mb_cud_seeders.Seller)
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

func TambahTransaksiApprovedVideo(ctx context.Context, data PayloadTambahMediaTransaksiApprovedVideo, db *environment.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	const services string = "TambahTransaksiApprovedVideo"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseMediaUpload{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	var id_data_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Transaksi{}).Select("id").Where(&sot_models.Transaksi{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.MediaTransaksiApprovedVideo{}).Select("id").Where(&sot_models.MediaTransaksiApprovedVideo{
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

	keyz := sot_models.MediaTransaksiApprovedVideo{}.PathName() + strconv.Itoa(int(id_data_transaksi)) + "/" + helper.GenerateMediaKeyVideo() + "." + data.Ekstensi

	url, err_url := ms.PresignedPutObject(ctx, media_storage_database_seeders.BucketVideoName, keyz, time.Minute*2)

	if err_url != nil {
		return &response.ResponseMediaUpload{
			Status:   http.StatusInternalServerError,
			Services: services,
		}
	}

	newTransaksiApprovedVideo := sot_models.MediaTransaksiApprovedVideo{
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

	go func(Mtav sot_models.MediaTransaksiApprovedVideo, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdTransaksi := sot_threshold.TransaksiThreshold{
			IdTransaksi: Mtav.IdTransaksi,
		}

		if err := thresholdTransaksi.Increment(konteks, Trh, stsk_transaksi.MediaTransaksiApprovedVideo); err != nil {
			fmt.Println("Gagal incr count media transaksi approved video ke threshold transaksi")
		}

		transaksiApprovedVideoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mtav).SetTableName(Mtav.TableName()).SetRole(mb_cud_seeders.Seller)
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
