package kurir_informasi_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	kurir_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity/kurir"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

func AjukanInformasiKendaraan(ctx context.Context, data PayloadInformasiDataKendaraan, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "AjukanInformasiKendaraanKurir"

	if !helper.JenisKendaraanKurirValidation(data.JenisKendaraan) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, jenis kendaraan kurir tidak valid",
		}
	}

	if !helper.JenisRodaKendaraanKurirValidation(data.RodaKendaraan) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, roda kendaraan kurir tidak valid",
		}
	}

	if strings.TrimSpace(data.NamaKendaraan) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, nama motor tak boleh kosong",
		}
	}

	if _, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial tidak valid.",
		}
	}

	var id_pengajuan_data_kendaraan int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.InformasiKendaraanKurir{}).Select("id").Where(&sot_models.InformasiKendaraanKurir{
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_pengajuan_data_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, Server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_pengajuan_data_kendaraan != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Kamu sudah membuat pengajuan",
		}
	}

	newInformasiKendaraan := sot_models.InformasiKendaraanKurir{
		IDkurir:        data.DataIdentitasKurir.IdKurir,
		JenisKendaraan: data.JenisKendaraan,
		NamaKendaraan:  data.NamaKendaraan,
		RodaKendaraan:  data.RodaKendaraan,
		STNK:           data.InformasiStnk,
		BPKB:           data.InformasiBpkb,
		NoRangka:       data.NomorRangka,
		NoMesin:        data.NomorMesin,
		Status:         kurir_enums.Pending,
	}

	if err := db.Write.WithContext(ctx).Create(&newInformasiKendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Ikk sot_models.InformasiKendaraanKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Ikk.IDkurir,
		}

		thresholdInformasiKendaraan := sot_threshold.InformasiKendaraanKurirThreshold{
			IdInformasiKendaraanKurir: Ikk.ID,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.InformasiKendaraanKurir); err != nil {
			fmt.Println("Gagal incr count informasi kendaraan kurir ke threshold kurir")
		}

		if err := thresholdInformasiKendaraan.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold informasi kendaraan kurir")
		}

		informasiKendaraanCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ikk).SetTableName(Ikk.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanCreatePublish); err != nil {
			fmt.Println("Gagal publish informasi kendaraan kurir create ke message broker")
		}
	}(newInformasiKendaraan, db.Write, cud_publisher)

	log.Printf("[INFO] Pengajuan informasi kendaraan berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil.",
	}
} // passed

func EditInformasiKendaraan(ctx context.Context, data PayloadEditInformasiDataKendaraan, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditInformasiKendaraanKurir"

	if data.IdInformasiKendaraan <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, id informais kendaraan tak boleh lebih kecil atau sama dengan 0",
		}
	}

	if !helper.JenisKendaraanKurirValidation(data.JenisKendaraan) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, jenis kendaraan kurir tidak valid",
		}
	}

	if !helper.JenisRodaKendaraanKurirValidation(data.RodaKendaraan) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, roda kendaraan kurir tidak valid",
		}
	}

	if strings.TrimSpace(data.NamaKendaraan) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, nama motor tak boleh kosong",
		}
	}

	if _, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial tidak valid.",
		}
	}

	var id_data_informasi_kendaraan int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.InformasiKendaraanKurir{}).Select("id").Where(&sot_models.InformasiKendaraanKurir{
		ID:      data.IdInformasiKendaraan,
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_informasi_kendaraan).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, Server Sedang Sibuk Coba Lagi Lain Waktu",
		}
	}

	if id_data_informasi_kendaraan == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, Data Tidak Valid",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&sot_models.InformasiKendaraanKurir{}).Where(&sot_models.InformasiKendaraanKurir{
		ID: data.IdInformasiKendaraan,
	}).Updates(&sot_models.InformasiKendaraanKurir{
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
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdIkk int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var dataInformasiKendaraanUpdated sot_models.InformasiKendaraanKurir
		if err := Read.WithContext(konteks).Model(&sot_models.InformasiKendaraanKurir{}).Where(&sot_models.InformasiKendaraanKurir{
			ID: IdIkk,
		}).Limit(1).Scan(&dataInformasiKendaraanUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data informasi kendaraan kurir")
			return
		}

		informasiKendaraanUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataInformasiKendaraanUpdated).SetTableName(dataInformasiKendaraanUpdated.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update informasi kendaraan kurir ke message broker")
		}
	}(data.IdInformasiKendaraan, db.Read, cud_publisher)

	log.Printf("[INFO] Edit informasi kendaraan berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil.",
	}
} // passed

func HapusInformasiKendaraan(ctx context.Context, data PayloadHapusInformasiDataKendaraan, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusInformasiKendaraan"

	if data.IdInformasiKendaraan <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, id informasi kendaraan tidak boleh lebih kecil atau sama dengan 0",
		}
	}

	kurir, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kredensial kurir tidak valid",
		}
	}

	var ubah_verified bool = kurir.VerifiedKurir

	var data_informasi_kendaraan_kurir sot_models.InformasiKendaraanKurir
	if err := db.Read.WithContext(ctx).Model(&sot_models.InformasiKendaraanKurir{}).Where(&sot_models.InformasiKendaraanKurir{
		ID:      data.IdInformasiKendaraan,
		IDkurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&data_informasi_kendaraan_kurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_informasi_kendaraan_kurir.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, data informasi kendaraan kurir tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sot_models.InformasiKendaraanKurir{}).Where(&sot_models.InformasiKendaraanKurir{
			ID: data.IdInformasiKendaraan,
		}).Delete(&sot_models.InformasiKendaraanKurir{}).Error; err != nil {
			return err
		}

		if ubah_verified {
			if err := tx.Model(&sot_models.Kurir{}).Where(&sot_models.Kurir{
				ID: data.IdentitasKurir.IdKurir,
			}).Updates(&sot_models.Kurir{
				VerifiedKurir: false,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(id_kurir int64, verif_ubah bool, IKK sot_models.InformasiKendaraanKurir, RW *environment.InternalDBReadWriteSystem, publisher *mb_cud_publisher.Publisher) {
		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: id_kurir,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := RW.Write.Unscoped().WithContext(konteks).Model(&sot_threshold.InformasiKendaraanKurirThreshold{}).Where(&sot_threshold.InformasiKendaraanKurirThreshold{
			IdInformasiKendaraanKurir: IKK.ID,
		}).Delete(&sot_threshold.InformasiKendaraanKurirThreshold{}).Error; err != nil {
			fmt.Println("Gagal delete threshold informasi kurir")
		}

		if err := thresholdKurir.Decrement(konteks, RW.Write, stsk_kurir.InformasiKendaraanKurir); err != nil {
			fmt.Println("gagal decrement threshold kurir kolom informasi kendaraan kurir")
		}

		informasiKendaraanKurirDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(IKK).SetTableName(IKK.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKendaraanKurirDeletePublish); err != nil {
			fmt.Println("gagal publish delete informasi kendaraan kurir ke message broker")
		}

		if verif_ubah {
			var K sot_models.Kurir
			if err := RW.Read.WithContext(konteks).Model(&sot_models.Kurir{}).Where(&sot_models.Kurir{
				ID: id_kurir,
			}).Limit(1).Scan(&K).Error; err != nil {
				fmt.Println("Gagal mengambil data kurir")
			}

			if K.ID == 0 {
				return
			}

			kurirUpdatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName() + services).SetRole(mb_cud_seeders.Kurir)
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatePublish); err != nil {
				fmt.Println("gagal publish update kurir ke message broker")
			}
		}

	}(data.IdentitasKurir.IdKurir, ubah_verified, data_informasi_kendaraan_kurir, db, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}

}

func AjukanInformasiKurir(ctx context.Context, data PayloadInformasiDataKurir, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "AjukanInformasiKurir"

	if parsedTime, err := time.Parse("2006-01-02", data.TanggalLahir); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, data tanggal lahir tidak berupa tanggal valid",
		}
	} else {
		data.TanggalLahir = parsedTime.Format("2006-01-02")
	}

	if _, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial tidak valid.",
		}
	}

	var id_data_pengajuan_informasi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.InformasiKurir{}).Select("id").Where(&sot_models.InformasiKurir{
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
			Message:  "Gagal, tunggu pengajuan sebelumnya ditindak kami.",
		}
	}

	newInformasiKurir := sot_models.InformasiKurir{
		IDkurir:      data.DataIdentitasKurir.IdKurir,
		TanggalLahir: data.TanggalLahir,
		Alasan:       data.Alasan,
		Ktp:          data.InformasiKtp,
		InformasiSim: data.InformasiSim,
		Status:       kurir_enums.Pending,
	}

	if err := db.Write.WithContext(ctx).Create(&newInformasiKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server Sedang Sibuk Coba Lagi Lain Waktu",
		}
	}

	go func(Ik sot_models.InformasiKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: Ik.IDkurir,
		}

		thresholdInformasiKurir := sot_threshold.InformasiKurirThreshold{
			IdInformasiKurir: Ik.ID,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdKurir.Increment(konteks, Trh, stsk_kurir.InformasiKurir); err != nil {
			fmt.Println("Gagal incr count informasi kurir ke threshold kurir")
		}

		if err := thresholdInformasiKurir.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat informasi kurir threshold")
		}

		informasiKurirCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ik).SetTableName(Ik.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKurirCreatePublish); err != nil {
			fmt.Println("Gagal publish informasi kurir create ke message broker")
		}
	}(newInformasiKurir, db.Write, cud_publisher)

	log.Printf("[INFO] Pengajuan data kurir berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil.",
	}
} // passed

func EditInformasiKurir(ctx context.Context, data PayloadEditInformasiDataKurir, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditInformasiKurir"

	if parsedTime, err := time.Parse("2006-01-02", data.TanggalLahir); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, data tanggal lahir tidak berupa tanggal valid",
		}
	} else {
		data.TanggalLahir = parsedTime.Format("2006-01-02")
	}

	_, status := data.DataIdentitasKurir.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial kurir tidak valid untuk ID %d", data.DataIdentitasKurir.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial tidak valid.",
		}
	}

	var id_data_pengajuan_informasi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.InformasiKurir{}).Select("id").Where(&sot_models.InformasiKurir{
		ID:      data.IdInformasiKurir,
		IDkurir: data.DataIdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_pengajuan_informasi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_pengajuan_informasi == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, data Tidak valid",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&sot_models.InformasiKurir{}).Where(&sot_models.InformasiKurir{
		ID: data.IdInformasiKurir,
	}).Updates(&sot_models.InformasiKurir{
		TanggalLahir: data.TanggalLahir,
		Alasan:       data.Alasan,
		Ktp:          data.InformasiKtp,
		InformasiSim: data.InformasiSim,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdIk int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var dataInformasiKurirUpdated sot_models.InformasiKurir
		if err := Read.WithContext(konteks).Model(&sot_models.InformasiKurir{}).Where(&sot_models.InformasiKurir{
			ID: IdIk,
		}).Limit(1).Scan(&dataInformasiKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data informasi kurir")
			return
		}

		informasiKurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataInformasiKurirUpdated).SetTableName(dataInformasiKurirUpdated.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKurirUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update informasi kurir ke message broker")
		}
	}(data.IdInformasiKurir, db.Read, cud_publisher)

	log.Printf("[INFO] Edit data kurir berhasil untuk kurir ID %d", data.DataIdentitasKurir.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil.",
	}
} // passed

func HapusInformasiKurir(ctx context.Context, data PayloadHapusInformasiDataKurir, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusInformasiKurir"

	if data.IdInformasiKurir <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, id informasi kurir tidak boleh lebih kecil atau sama dengan 0",
		}
	}

	kurir, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kredensial kurir tidak valid",
		}
	}

	var ubah_verified bool = kurir.VerifiedKurir

	var data_informasi_kurir sot_models.InformasiKurir
	if err := db.Read.WithContext(ctx).Model(&sot_models.InformasiKurir{}).Where(&sot_models.InformasiKurir{
		ID:      data.IdInformasiKurir,
		IDkurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&data_informasi_kurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_informasi_kurir.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, data informasi kurir tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sot_models.InformasiKurir{}).Where(&sot_models.InformasiKurir{
			ID: data.IdInformasiKurir,
		}).Delete(&sot_models.InformasiKurir{}).Error; err != nil {
			return err
		}

		if ubah_verified {
			if err := tx.Model(&sot_models.Kurir{}).Where(&sot_models.Kurir{
				ID: data.IdentitasKurir.IdKurir,
			}).Updates(&sot_models.Kurir{
				VerifiedKurir: false,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(id_kurir int64, verif_ubah bool, IK sot_models.InformasiKurir, RW *environment.InternalDBReadWriteSystem, publisher *mb_cud_publisher.Publisher) {
		thresholdKurir := sot_threshold.KurirThreshold{
			IdKurir: id_kurir,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := RW.Write.Unscoped().WithContext(konteks).Model(&sot_threshold.InformasiKurirThreshold{}).Where(&sot_threshold.InformasiKurirThreshold{
			IdInformasiKurir: IK.ID,
		}).Delete(&sot_threshold.InformasiKurirThreshold{}).Error; err != nil {
			fmt.Println("Gagal delete threshold informasi kurir")
		}

		if err := thresholdKurir.Decrement(konteks, RW.Write, stsk_kurir.InformasiKurir); err != nil {
			fmt.Println("gagal decrement threshold kurir kolom informasi kurir")
		}

		informasiKurirDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(IK).SetTableName(IK.TableName()).SetRole(mb_cud_seeders.Kurir)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, informasiKurirDeletePublish); err != nil {
			fmt.Println("gagal publish delete informasi kurir ke message broker")
		}

		if verif_ubah {
			var K sot_models.Kurir
			if err := RW.Read.WithContext(konteks).Model(&sot_models.Kurir{}).Where(&sot_models.Kurir{
				ID: id_kurir,
			}).Limit(1).Scan(&K).Error; err != nil {
				fmt.Println("Gagal mengambil data kurir")
			}

			if K.ID == 0 {
				return
			}

			kurirUpdatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName() + services).SetRole(mb_cud_seeders.Kurir)
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatePublish); err != nil {
				fmt.Println("gagal publish update kurir ke message broker")
			}
		}

	}(data.IdentitasKurir.IdKurir, ubah_verified, data_informasi_kurir, db, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}

}
