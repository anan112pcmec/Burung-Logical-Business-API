package kurir_alamat_services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_kota"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_provinsi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/alamat_services/response_alamat_service_kurir"
)

func MasukanAlamatKurir(ctx context.Context, data PayloadMasukanAlamatKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanAlamatKurir"

	// Validasi identitas kurir
	_, valid := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Data kurir tidak valid",
			},
		}
	}

	if _, ok := nama_provinsi.JawaProvinsiMap[data.Provinsi]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Nama provinsi tidak valid",
		}
	}

	if _, ok := nama_kota.KotaJawaMap[data.Kota]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Nama kota tidak valid",
		}
	}

	// Cek apakah alamat sudah ada
	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatKurir{}).Select("id").
		Where(&models.AlamatKurir{IdKurir: data.IdentitasKurir.IdKurir}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Server sedang sibuk, coba lagi lain waktu",
			},
		}
	}

	if id_data_alamat != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Kamu sudah memasukan data alamat",
			},
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longtitude)

	// Simpan alamat baru
	newAlamatKurir := models.AlamatKurir{
		IdKurir:         data.IdentitasKurir.IdKurir,
		PanggilanAlamat: data.PanggilanAlamat,
		NomorTelephone:  data.NomorTelephone,
		NamaAlamat:      data.NamaAlamat,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodeNegara:      data.KodeNegara,
		KodePos:         data.KodePos,
		Deskripsi:       data.Deskripsi,
		Longitude:       data.Longtitude,
		Latitude:        data.Latitude,
	}
	if err := db.Write.WithContext(ctx).Create(&newAlamatKurir).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
				Message: "Gagal: Tidak dapat menyimpan data alamat, coba lagi lain waktu",
			},
		}
	}

	go func(Ak models.AlamatKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		kurirThreshold := sot_threshold.KurirThreshold{
			IdKurir: Ak.IdKurir,
		}

		if err := kurirThreshold.Increment(konteks, Trh, stsk_kurir.AlamatKurir); err != nil {
			fmt.Println("Gagal incr count alamat kurir ke kurir threshold")
		}

		alamatKurirCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ak).SetTableName(Ak.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, alamatKurirCreatePublish); err != nil {
			fmt.Println("Gagal publish create alamat kurir ke message broker")
		}
	}(newAlamatKurir, db.Write, cud_publisher)

	// Berhasil
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_alamat_service_kurir.ResponseMasukanAlamatKurir{
			Message: "Berhasil",
		},
	}
}

func EditAlamatKurir(ctx context.Context, data PayloadEditAlamatKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditAlamatKurir"

	// Validasi identitas kurir
	_, valid := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal: Data kurir tidak valid",
			},
		}
	}

	if _, ok := nama_provinsi.JawaProvinsiMap[data.Provinsi]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Nama provinsi tidak valid",
		}
	}

	if _, ok := nama_kota.KotaJawaMap[data.Kota]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Nama kota tidak valid",
		}
	}

	// Cek apakah alamat dengan ID dan kurir terkait benar-benar ada
	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatKurir{}).Select("id").Where(&models.AlamatKurir{
		ID:      data.IDAlamatKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal Data Alamat Tidak Valid",
			},
		}
	}

	if id_data_alamat == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal: Data alamat tidak ditemukan",
			},
		}
	}

	// Update data alamat
	if err := db.Write.WithContext(ctx).Model(&models.AlamatKurir{}).
		Where(&models.AlamatKurir{ID: data.IDAlamatKurir}).
		Updates(&models.AlamatKurir{
			PanggilanAlamat: data.PanggilanAlamat,
			NomorTelephone:  data.NomorTelephone,
			NamaAlamat:      data.NamaAlamat,
			Kota:            data.Kota,
			Provinsi:        data.Provinsi,
			KodeNegara:      data.KodeNegara,
			KodePos:         data.KodePos,
			Deskripsi:       data.Deskripsi,
			Longitude:       data.Longtitude,
			Latitude:        data.Latitude,
		}).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
				Message: "Gagal: Mengubah data alamat, coba lagi nanti",
			},
		}
	}

	go func(IdAk int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var alamatKurirUpdated models.AlamatKurir
		if err := Read.WithContext(konteks).Model(&models.AlamatKurir{}).Where(&models.AlamatKurir{
			ID: IdAk,
		}).Limit(1).Scan(&alamatKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data alamat kurir updated")
			return
		}

		alamatKurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(alamatKurirUpdated).SetTableName(alamatKurirUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, alamatKurirUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated alamat kurir ke messag broker")
		}
	}(data.IDAlamatKurir, db.Read, cud_publisher)

	// Berhasil
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_alamat_service_kurir.ResponseEditAlamatKurir{
			Message: "Berhasil",
		},
	}
}

func HapusAlamatKurir(ctx context.Context, data PayloadHapusAlamatKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusAlamatKurir"

	// Validasi identitas kurir
	_, valid := data.IdentitasKurir.Validating(ctx, db.Read, rds_session)
	if !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal: Data kurir tidak valid",
			},
		}
	}

	// Cek apakah alamat milik kurir tersebut ada
	var data_alamat models.AlamatKurir
	if err := db.Read.WithContext(ctx).Model(&models.AlamatKurir{}).Where(&models.AlamatKurir{
		ID:      data.IdAlamatKurir,
		IdKurir: data.IdentitasKurir.IdKurir,
	}).Limit(1).Scan(&data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal Data Alamat Tidak Valid",
			},
		}
	}

	if data_alamat.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal: Data alamat tidak ditemukan",
			},
		}
	}

	// Hapus alamat
	if err := db.Write.Model(&models.AlamatKurir{}).
		Where(&models.AlamatKurir{
			ID:      data.IdAlamatKurir,
			IdKurir: data.IdentitasKurir.IdKurir,
		}).
		Delete(&models.AlamatKurir{}).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
				Message: "Gagal: Server sedang sibuk, coba lagi lain waktu",
			},
		}
	}

	go func(Ak models.AlamatKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		kurirThreshold := sot_threshold.KurirThreshold{
			IdKurir: Ak.IdKurir,
		}

		if err := kurirThreshold.Decrement(konteks, Trh, stsk_kurir.AlamatKurir); err != nil {
			fmt.Println("Gagal decr count alamat kurir ke kurir threshold")
		}

		alamatKurirDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Ak).SetTableName(Ak.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, alamatKurirDeletePublish); err != nil {
			fmt.Println("Gagal publish delete alamat kurir ke message broker")
		}
	}(data_alamat, db.Write, cud_publisher)

	// Berhasil
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_alamat_service_kurir.ResponseHapusAlamatKurir{
			Message: "Berhasil",
		},
	}
}
