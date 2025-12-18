package pengguna_alamat_services

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
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Masukan Alamat Pengguna
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanAlamatPengguna(ctx context.Context, data PayloadMasukanAlamatPengguna, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanAlamatPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data Pengguna Tidak Valid",
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

	var id_data_alamats []int64
	if err := db.Read.WithContext(ctx).Select("id").Model(&models.AlamatPengguna{}).
		Where(models.AlamatPengguna{IDPengguna: data.IdentitasPengguna.ID}).
		Limit(5).Scan(&id_data_alamats).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi waktu",
		}
	}

	if len(id_data_alamats) == 5 {
		return &response.ResponseForm{
			Status:   http.StatusForbidden,
			Services: services,
			Message:  "Gagal batas alamat hanya 5",
		}
	}

	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatPengguna{}).Select("id").Where(&models.AlamatPengguna{
		IDPengguna:      data.IdentitasPengguna.ID,
		PanggilanAlamat: data.PanggilanAlamat,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_alamat != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah memiliki alamat dengan panggilan yang sama",
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longitude)

	Alamatpengguna := models.AlamatPengguna{
		IDPengguna:      data.IdentitasPengguna.ID,
		PanggilanAlamat: data.PanggilanAlamat,
		NamaAlamat:      data.NamaAlamat,
		Deskripsi:       data.Deskripsi,
		NomorTelephone:  data.NomorTelephone,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodePos:         data.KodePos,
		KodeNegara:      data.KodeNegara,
		Longitude:       data.Longitude,
		Latitude:        data.Latitude,
	}

	if err := db.Write.WithContext(ctx).Create(&Alamatpengguna).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Ap models.AlamatPengguna, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		threshold_pengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: Ap.IDPengguna,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := threshold_pengguna.Increment(konteks, Trh, stsk_pengguna.AlamatPengguna); err != nil {
			fmt.Println("gagal incr threshold pengguna")
		}

		threshold_alamat := sot_threshold.AlamatPenggunaThreshold{
			IdAlamatPengguna: Ap.ID,
		}

		if err := threshold_alamat.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold alamat pengguna")
		}

		publishNewAlamat := mb_cud_serializer.NewJsonPayload().SetPayload(Ap).SetTableName(Ap.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, cud_publisher, publishNewAlamat); err != nil {
			fmt.Println("gagal publish create alamat ke message broker")
		}
	}(Alamatpengguna, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditAlamatPengguna(ctx context.Context, data PayloadEditAlamatPengguna, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditAlamatPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Pengguna tidak ditemukan",
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

	var id_alamat_pengguna int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatPengguna{}).Select("id").Where(&models.AlamatPengguna{
		ID:         data.IdAlamatPengguna,
		IDPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_alamat_pengguna).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_alamat_pengguna == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data alamat tidak ditemukan",
		}
	}

	var idDataTransaksi int64 = 0

	if err := db.Read.WithContext(ctx).
		Model(&models.Transaksi{}).
		Select("id").
		Where("id_alamat_pengguna = ? AND status != ?", data.IdAlamatPengguna, "Selesai").
		Limit(1).
		Scan(&idDataTransaksi).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	// Jika ada transaksi yang menggunakan alamat ini
	if idDataTransaksi != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, alamat sedang digunakan sebagai acuan transaksi",
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longitude)

	if err := db.Write.WithContext(ctx).Model(&models.AlamatPengguna{}).Where(&models.AlamatPengguna{
		ID: data.IdAlamatPengguna,
	}).Updates(&models.AlamatPengguna{
		PanggilanAlamat: data.PanggilanAlamat,
		NamaAlamat:      data.NamaAlamat,
		Deskripsi:       data.Deskripsi,
		NomorTelephone:  data.NomorTelephone,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodePos:         data.KodePos,
		KodeNegara:      data.KodeNegara,
		Longitude:       data.Longitude,
		Latitude:        data.Latitude,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(idAlamat int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		ap := models.AlamatPengguna{}
		if err := Read.WithContext(konteks).Model(&models.AlamatPengguna{}).Where(&models.AlamatPengguna{
			ID: idAlamat,
		}).Limit(1).Take(&ap).Error; err != nil {
			fmt.Println("Gagal Mendapatkan alamat pengguna untuk publish")
			return
		}

		publishUpdateAlamat := mb_cud_serializer.NewJsonPayload().SetPayload(ap).SetTableName(ap.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, publishUpdateAlamat); err != nil {
			fmt.Println("gagal publish update alamat ke message broker")
			return
		}
	}(data.IdAlamatPengguna, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Hapus Alamat Pengguna
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusAlamatPengguna(ctx context.Context, data PayloadHapusAlamatPengguna, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusAlamatPengguna"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusOK,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var alamat_pengguna models.AlamatPengguna
	if err := db.Read.WithContext(ctx).Model(&models.AlamatPengguna{}).Where(&models.AlamatPengguna{
		ID:         data.IdAlamatPengguna,
		IDPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&alamat_pengguna).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if alamat_pengguna.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data alamat tidak ditemukan",
		}
	}

	var idDataTransaksi int64 = 0

	if err := db.Read.WithContext(ctx).
		Model(&models.Transaksi{}).
		Select("id").
		Where("id_alamat_pengguna = ? AND status != ?", data.IdAlamatPengguna, "Selesai").
		Limit(1).
		Scan(&idDataTransaksi).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	// Jika ada transaksi yang menggunakan alamat ini
	if idDataTransaksi != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, alamat sedang digunakan sebagai acuan transaksi",
		}
	}

	if err_hapus := db.Write.WithContext(ctx).Where(models.AlamatPengguna{
		ID: data.IdAlamatPengguna,
	}).Delete(&models.AlamatPengguna{}).Error; err_hapus != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Ap models.AlamatPengguna, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		threshold_pengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: Ap.IDPengguna,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := threshold_pengguna.Decrement(konteks, Trh, stsk_pengguna.AlamatPengguna); err != nil {
			fmt.Println("gagal decr threshold pengguna")
		}

		threshold_alamat := sot_threshold.AlamatPenggunaThreshold{
			IdAlamatPengguna: Ap.ID,
		}

		if err := db.Write.WithContext(konteks).Delete(&threshold_alamat).Error; err != nil {
			fmt.Println("gagal menghapus threshold alamat")
		}

		publishDeleteAlamat := mb_cud_serializer.NewJsonPayload().SetPayload(Ap).SetTableName(Ap.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, cud_publisher, publishDeleteAlamat); err != nil {
			fmt.Println("gagal publish hapus alamat ke message broker")
		}
	}(alamat_pengguna, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
