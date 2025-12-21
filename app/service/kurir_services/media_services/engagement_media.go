package kurir_media_services

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
	pengiriman_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/pengiriman"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_informasi_kendaraan_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/informasi_kendaraan"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func UbahKurirProfilFoto(ctx context.Context, data PayloadUbahKurirProfilFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "UbahKurirProfilFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
		newKurirProfilFoto := models.MediaKurirProfilFoto{
			IdKurir: data.IdentitasKurir.IdKurir,
			Key:     keyz,
			Format:  data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newKurirProfilFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mkpf models.MediaKurirProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdKurir := sot_threshold.KurirThreshold{
				IdKurir: Mkpf.IdKurir,
			}

			if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.MediaKurirProfilFoto); err != nil {
				fmt.Println("Gagal incr count media kurir profil foto ke threshold kurir")
			}

			kurirProfilFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mkpf).SetTableName(Mkpf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirProfilFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish kurir profil foto create ke message broker")
			}
		}(newKurirProfilFoto, db.Write, cud_publisher)
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

		go func(IdMkpf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataKurirProfilFotoUpdated models.MediaKurirProfilFoto
			if err := Read.WithContext(konteks).Model(&models.MediaKurirProfilFoto{}).Where(&models.MediaKurirProfilFoto{
				ID: IdMkpf,
			}).Limit(1).Take(&dataKurirProfilFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data kurir profil foto")
				return
			}

			kurirProfilFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKurirProfilFotoUpdated).SetTableName(dataKurirProfilFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirProfilFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update kurir profil foto ke message broker")
			}
		}(id_data_media_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		UrlUpload: minIOUrl,
		Key:       keyz,
	}
}

func HapusKurirProfilFoto(ctx context.Context, data PayloadHapusKurirProfilFoto, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKurirProfilFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var data_media_foto_profil_kurir models.MediaKurirProfilFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaKurirProfilFoto{}).Where(&models.MediaKurirProfilFoto{
		ID:  data.IdMediaKurirProfilFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_foto_profil_kurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_foto_profil_kurir.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data foto tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.MediaKurirProfilFoto{}).Where(&models.MediaKurirProfilFoto{
		ID: data.IdMediaKurirProfilFoto,
	}).Delete(&models.MediaKurirProfilFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmfpk models.MediaKurirProfilFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Dmfpk.IdKurir,
		}

		if err := thresholdKurir.Decrement(konteks, Trh, stsk_kurir.MediaKurirProfilFoto); err != nil {
			fmt.Println("Gagal decr media kurir profil foto ke threshold kurir")
		}

		kurirProfilFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmfpk).SetTableName(Dmfpk.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirProfilFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete kurir profil foto ke message broker")
		}
	}(data_media_foto_profil_kurir, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahMediaInformasiKendaraanKurirKendaraanFoto(ctx context.Context, data PayloadTambahMediaInformasiKurirKendaraanFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahMediaInformasiKendaraanKurirKendaraanFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
		newInformasiKendaraanKurirKendaraanFoto := models.MediaInformasiKendaraanKurirKendaraanFoto{
			IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
			Key:                       keyz,
			Format:                    data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newInformasiKendaraanKurirKendaraanFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mikkkf models.MediaInformasiKendaraanKurirKendaraanFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdInformasiKendaraanKurir := sot_threshold.InformasiKendaraanKurirThreshold{
				IdInformasiKendaraanKurir: Mikkkf.IdInformasiKendaraanKurir,
			}

			if err := thresholdInformasiKendaraanKurir.Increment(konteks, Trh, stsk_informasi_kendaraan_kurir.MediaInformasiKendaraanKurirKendaraanFoto); err != nil {
				fmt.Println("Gagal incr count media informasi kendaraan kurir kendaraan foto ke threshold informasi kendaraan kurir")
			}

			informasiKendaraanKurirKendaraanFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mikkkf).SetTableName(Mikkkf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanKurirKendaraanFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish informasi kendaraan kurir kendaraan foto create ke message broker")
			}
		}(newInformasiKendaraanKurirKendaraanFoto, db.Write, cud_publisher)
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

		go func(IdMikkkf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataInformasiKendaraanKurirKendaraanFotoUpdated models.MediaInformasiKendaraanKurirKendaraanFoto
			if err := Read.WithContext(konteks).Model(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Where(&models.MediaInformasiKendaraanKurirKendaraanFoto{
				ID: IdMikkkf,
			}).Limit(1).Take(&dataInformasiKendaraanKurirKendaraanFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data informasi kendaraan kurir kendaraan foto")
				return
			}

			informasiKendaraanKurirKendaraanFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataInformasiKendaraanKurirKendaraanFotoUpdated).SetTableName(dataInformasiKendaraanKurirKendaraanFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanKurirKendaraanFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update informasi kendaraan kurir kendaraan foto ke message broker")
			}
		}(id_media_informasi_kendaraan_kurir_kendaraan_foto, db.Read, cud_publisher)
	}

	return &response.ResponseMediaUpload{
		Status:    http.StatusOK,
		Services:  services,
		Key:       keyz,
		UrlUpload: url.String(),
	}
}

func HapusMediaInformasiKendaraanKurirKendaraanFoto(ctx context.Context, data PayloadHapusMediaInformasiKurirKendaraanFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusMediaInformasiKendaraanKurirKendaraanFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var data_media_informasi_kendaraan_kurir_kendaraan_foto models.MediaInformasiKendaraanKurirKendaraanFoto
	if err := db.Read.WithContext(ctx).Model(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Where(&models.MediaInformasiKendaraanKurirKendaraanFoto{
		ID:  data.IdMediaKendaraanKurirKendaraanFoto,
		Key: data.KeyFoto,
	}).Limit(1).Scan(&data_media_informasi_kendaraan_kurir_kendaraan_foto).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_media_informasi_kendaraan_kurir_kendaraan_foto.ID == 0 {
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
		ID: data_media_informasi_kendaraan_kurir_kendaraan_foto.ID,
	}).Delete(&models.MediaInformasiKendaraanKurirKendaraanFoto{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dmikkkf models.MediaInformasiKendaraanKurirKendaraanFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdInformasiKendaraanKurir := sot_threshold.InformasiKendaraanKurirThreshold{
			IdInformasiKendaraanKurir: Dmikkkf.IdInformasiKendaraanKurir,
		}

		if err := thresholdInformasiKendaraanKurir.Decrement(konteks, Trh, stsk_informasi_kendaraan_kurir.MediaInformasiKendaraanKurirKendaraanFoto); err != nil {
			fmt.Println("Gagal decr media informasi kendaraan kurir kendaraan foto ke threshold informasi kendaraan kurir")
		}

		informasiKendaraanKurirKendaraanFotoDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dmikkkf).SetTableName(Dmikkkf.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanKurirKendaraanFotoDeletePublish); err != nil {
			fmt.Println("Gagal publish delete informasi kendaraan kurir kendaraan foto ke message broker")
		}
	}(data_media_informasi_kendaraan_kurir_kendaraan_foto, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func TambahInformasiKendaraanKurirBPKBFoto(ctx context.Context, data PayloadTambahMediaInformasiKendaraanKurirBPKBFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseMediaUpload {
	services := "TambahInformasiKendaraanKurirBPKBFoto"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
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
		newInformasiKendaraanKurirBPKBFoto := models.MediaInformasiKendaraanKurirBPKBFoto{
			IdInformasiKendaraanKurir: id_data_informasi_kendaraan_kurir,
			Key:                       keyz,
			Format:                    data.Ekstensi,
		}
		if err := db.Write.WithContext(ctx).Create(&newInformasiKendaraanKurirBPKBFoto).Error; err != nil {
			return &response.ResponseMediaUpload{
				Status:   http.StatusInternalServerError,
				Services: services,
			}
		}

		go func(Mikkbf models.MediaInformasiKendaraanKurirBPKBFoto, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			thresholdInformasiKendaraanKurir := sot_threshold.InformasiKendaraanKurirThreshold{
				IdInformasiKendaraanKurir: Mikkbf.IdInformasiKendaraanKurir,
			}

			if err := thresholdInformasiKendaraanKurir.Increment(konteks, Trh, stsk_informasi_kendaraan_kurir.MediaInformasiKendaraanKurirBpkbFoto); err != nil {
				fmt.Println("Gagal incr count media informasi kendaraan kurir bpkb foto ke threshold informasi kendaraan kurir")
			}

			informasiKendaraanKurirBPKBFotoCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Mikkbf).SetTableName(Mikkbf.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanKurirBPKBFotoCreatePublish); err != nil {
				fmt.Println("Gagal publish informasi kendaraan kurir bpkb foto create ke message broker")
			}
		}(newInformasiKendaraanKurirBPKBFoto, db.Write, cud_publisher)
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

		go func(IdMikkbf int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			ctx_t := context.Background()
			konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
			defer cancel()

			var dataInformasiKendaraanKurirBPKBFotoUpdated models.MediaInformasiKendaraanKurirBPKBFoto
			if err := Read.WithContext(konteks).Model(&models.MediaInformasiKendaraanKurirBPKBFoto{}).Where(&models.MediaInformasiKendaraanKurirBPKBFoto{
				ID: IdMikkbf,
			}).Limit(1).Take(&dataInformasiKendaraanKurirBPKBFotoUpdated).Error; err != nil {
				fmt.Println("Gagal mengambil data informasi kendaraan kurir bpkb foto")
				return
			}

			informasiKendaraanKurirBPKBFotoUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataInformasiKendaraanKurirBPKBFotoUpdated).SetTableName(dataInformasiKendaraanKurirBPKBFotoUpdated.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanKurirBPKBFotoUpdatedPublish); err != nil {
				fmt.Println("Gagal publish update informasi kendaraan kurir bpkb foto ke message broker")
			}
		}(id_media_informasi_kendaraan_kurir_bpkb_foto, db.Read, cud_publisher)
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

func TambahMediaPengirimanEkspedisiSampaiAgentFoto(ctx context.Context, data PayloadTambahPengirimanEkspedisiSampaiAgentFoto, db *config.InternalDBReadWriteSystem, ms *minio.Client) *response.ResponseMediaUpload {
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
