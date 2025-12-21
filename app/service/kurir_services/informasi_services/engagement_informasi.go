package kurir_informasi_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_informasi_services_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/informasi_services/response_informasi_services"
)

func AjukanInformasiKendaraan(ctx context.Context, data PayloadInformasiDataKendaraan, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "AjukanInformasiKendaraanKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_pengajuan_data_kendaraan int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_pengajuan_data_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_pengajuan_data_kendaraan != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, Kamu sudah membuat pengajuan",
			},
		}
	}

	newInformasiKendaraan := models.InformasiKendaraanKurir{
		IDkurir:        data.DataIdentitasKurir.IdKurir,
		JenisKendaraan: data.JenisKendaraan,
		NamaKendaraan:  data.NamaKendaraan,
		RodaKendaraan:  data.RodaKendaraan,
		STNK:           data.InformasiStnk,
		BPKB:           data.InformasiBpkb,
		NoRangka:       data.NomorRangka,
		NoMesin:        data.NomorMesin,
		Status:         "Pending",
	}

	if err := db.Write.WithContext(ctx).Create(&newInformasiKendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
				Message: "Gagal, server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(Ikk models.InformasiKendaraanKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Ikk.IDkurir,
		}

		thresholdInformasiKendaraan := sot_threshold.InformasiKendaraanKurirThreshold{
			IdInformasiKendaraanKurir: Ikk.ID,
		}

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.InformasiKendaraanKurir); err != nil {
			fmt.Println("Gagal incr count informasi kendaraan kurir ke threshold kurir")
		}

		if err := thresholdInformasiKendaraan.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold informasi kendaraan kurir")
		}

		informasiKendaraanCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ikk).SetTableName(Ikk.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanCreatePublish); err != nil {
			fmt.Println("Gagal publish informasi kendaraan kurir create ke message broker")
		}
	}(newInformasiKendaraan, db.Write, cud_publisher)

	log.Printf("[INFO] Pengajuan informasi kendaraan berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseAjukanInformasiKendaraan{
			Message: "Berhasil.",
		},
	}
}

func EditInformasiKendaraan(ctx context.Context, data PayloadEditInformasiDataKendaraan, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditInformasiKendaraanKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_data_informasi_kendaraan int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Select("id").Where(&models.InformasiKendaraanKurir{
		ID:      data.IdInformasiKendaraan,
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal, Server Sedang Sibuk Coba Lagi Lain Waktu",
			},
		}
	}

	if id_data_informasi_kendaraan == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal, Data Tidak Valid",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.InformasiKendaraanKurir{}).Where(&models.InformasiKendaraanKurir{
		ID: data.IdInformasiKendaraan,
	}).Updates(&models.InformasiKendaraanKurir{
		JenisKendaraan: data.JenisKendaraan,
		NamaKendaraan:  data.NamaKendaraan,
		RodaKendaraan:  data.RodaKendaraan,
		STNK:           data.InformasiStnk,
		BPKB:           data.InformasiBpkb,
		NoRangka:       data.NomorRangka,
		NoMesin:        data.NomorMesin,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
				Message: "Gagal Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(IdIkk int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataInformasiKendaraanUpdated models.InformasiKendaraanKurir
		if err := Read.WithContext(konteks).Model(&models.InformasiKendaraanKurir{}).Where(&models.InformasiKendaraanKurir{
			ID: IdIkk,
		}).Limit(1).Take(&dataInformasiKendaraanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data informasi kendaraan kurir")
			return
		}

		informasiKendaraanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataInformasiKendaraanUpdated).SetTableName(dataInformasiKendaraanUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update informasi kendaraan kurir ke message broker")
		}
	}(data.IdInformasiKendaraan, db.Read, cud_publisher)

	log.Printf("[INFO] Edit informasi kendaraan berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseEditInformasiKendaraan{
			Message: "Berhasil.",
		},
	}
}

func AjukanInformasiKurir(ctx context.Context, data PayloadInformasiDataKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "AjukanInformasiKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_data_pengajuan_informasi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKurir{}).Select("id").Where(&models.InformasiKurir{
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_pengajuan_informasi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_pengajuan_informasi != 0 {
		log.Printf("[WARN] Sudah ada pengajuan data kurir yang belum diproses untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
				Message: "Gagal, tunggu pengajuan sebelumnya ditindak kami.",
			},
		}
	}

	newInformasiKurir := models.InformasiKurir{
		IDkurir:      data.DataIdentitasKurir.IdKurir,
		TanggalLahir: data.TanggalLahir,
		Alasan:       data.Alasan,
		Ktp:          data.InformasiKtp,
		InformasiSim: data.InformasiSim,
		Status:       "Pending",
	}

	if err := db.Write.WithContext(ctx).Create(&newInformasiKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
				Message: "Gagal Server Sedang Sibuk Coba Lagi Lain Waktu",
			},
		}
	}

	go func(Ik models.InformasiKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Ik.IDkurir,
		}

		thresholdInformasiKurir := sot_threshold.InformasiKurirThreshold{
			IdInformasiKurir: Ik.ID,
		}

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.InformasiKurir); err != nil {
			fmt.Println("Gagal incr count informasi kurir ke threshold kurir")
		}

		if err := thresholdInformasiKurir.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat informasi kurir threshold")
		}

		informasiKurirCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ik).SetTableName(Ik.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKurirCreatePublish); err != nil {
			fmt.Println("Gagal publish informasi kurir create ke message broker")
		}
	}(newInformasiKurir, db.Write, cud_publisher)

	log.Printf("[INFO] Pengajuan data kurir berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseAjukanInformasiKurir{
			Message: "Berhasil.",
		},
	}
}

func EditInformasiKurir(ctx context.Context, data PayloadEditInformasiDataKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditInformasiKurir"

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal, kredensial tidak valid.",
			},
		}
	}

	var id_data_pengajuan_informasi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.InformasiKurir{}).Select("id").Where(&models.InformasiKurir{
		ID:      data.IdInformasiKurir,
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_pengajuan_informasi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal, server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	if id_data_pengajuan_informasi == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal, data Tidak valid",
			},
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.InformasiKurir{}).Where(&models.InformasiKurir{
		ID: data.IdInformasiKurir,
	}).Updates(&models.InformasiKurir{
		TanggalLahir: data.TanggalLahir,
		Alasan:       data.Alasan,
		Ktp:          data.InformasiKtp,
		InformasiSim: data.InformasiSim,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
				Message: "Gagal Server sedang sibuk coba lagi lain waktu",
			},
		}
	}

	go func(IdIk int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataInformasiKurirUpdated models.InformasiKurir
		if err := Read.WithContext(konteks).Model(&models.InformasiKurir{}).Where(&models.InformasiKurir{
			ID: IdIk,
		}).Limit(1).Take(&dataInformasiKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data informasi kurir")
			return
		}

		informasiKurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataInformasiKurirUpdated).SetTableName(dataInformasiKurirUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKurirUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update informasi kurir ke message broker")
		}
	}(data.IdInformasiKurir, db.Read, cud_publisher)

	log.Printf("[INFO] Edit data kurir berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_informasi_services_kurir.ResponseEditInformasiKurir{
			Message: "Berhasil.",
		},
	}
}
