package seller_alamat_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_kota"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_provinsi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Alamat Gudang Seller
// Berfungsi Untuk Menulis Ke table alamat_gudang tentang alamat gudang seller tersebut, tidak ada batasan maksimal
// gudang yang boleh dilampirkan alamat nya
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TambahAlamatGudang(ctx context.Context, data PayloadTambahAlamatGudang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahAlamatGudang"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial seller tidak valid untuk ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Kredensial seller tidak valid.",
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

	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Select("id").Where(&models.AlamatGudang{
		IDSeller:   data.IdentitasSeller.IdSeller,
		NamaAlamat: data.NamaAlamat,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_alamat != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal Kamu sudah memiliki alamat dengan nama yang sama",
		}
	}

	helper.SanitasiKoordinat(&data.Latitude, &data.Longitude)

	newAlamatGudang := models.AlamatGudang{
		IDSeller:        data.IdentitasSeller.IdSeller,
		PanggilanAlamat: data.PanggilanAlamat,
		NomorTelephone:  data.NomorTelefon,
		NamaAlamat:      data.NamaAlamat,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodePos:         data.KodePos,
		KodeNegara:      data.KodeNegara,
		Deskripsi:       data.Deskripsi,
		Longitude:       data.Longitude,
		Latitude:        data.Latitude,
	}

	if err := db.Write.WithContext(ctx).Create(&newAlamatGudang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(A models.AlamatGudang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(A.IDSeller),
		}

		thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
			IdAlamatGudang: A.ID,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.AlamatGudang); err != nil {
			fmt.Println("gagal increment count alamat gudang ke threshold seller")
		}

		if err := thresholdAlamatGudang.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("gagal membuat alamat gudang threshold")
		}

		createnewAlamatGudangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(A).SetTableName(A.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createnewAlamatGudangPublish); err != nil {
			fmt.Print("Gagal publish create alamat gudang new ke message broker")
		}
	}(newAlamatGudang, db.Write, cud_publisher)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil ditambahkan.",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Edit Alamat Gudang
// Berfungsi Untuk Seller manakala mereka ingin mengedit gudang mereka entah perubahan titik, nama dll
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditAlamatGudang(ctx context.Context, data PayloadEditAlamatGudang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditAlamatGudang"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial seller tidak valid untuk ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Kredensial seller tidak valid.",
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

	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Select("id").Where(&models.AlamatGudang{
		ID:       data.IdAlamatGudang,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_alamat == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data alamat tidak valid",
		}
	}

	var idDataTransaksi int64 = 0

	if err := db.Read.WithContext(ctx).
		Model(&models.Transaksi{}).
		Select("id").
		Where("id_alamat_gudang = ? AND status != ?", data.IdAlamatGudang, "Selesai").
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

	if err := db.Write.WithContext(ctx).Model(&models.AlamatGudang{}).Where(&models.AlamatGudang{
		ID: data.IdAlamatGudang,
	}).Updates(&models.AlamatGudang{
		PanggilanAlamat: data.PanggilanAlamat,
		NomorTelephone:  data.NomorTelefon,
		NamaAlamat:      data.NamaAlamat,
		Provinsi:        data.Provinsi,
		Kota:            data.Kota,
		KodePos:         data.KodePos,
		KodeNegara:      data.KodeNegara,
		Deskripsi:       data.Deskripsi,
		Longitude:       data.Longitude,
		Latitude:        data.Latitude,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdAlamatGudang int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataUpdatedAlamatGudang models.AlamatGudang
		if err := Read.WithContext(konteks).Model(&models.AlamatGudang{}).Where(&models.AlamatGudang{
			ID: IdAlamatGudang,
		}).Limit(1).Take(&dataUpdatedAlamatGudang).Error; err != nil {
			fmt.Println("Gagal mendapatkan data alamat gudang terbaru")
		}

		updatedAlamatGudangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedAlamatGudang).SetTableName(dataUpdatedAlamatGudang.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedAlamatGudangPublish); err != nil {
			fmt.Println("Gagal publish updated alamat gudang ke message broker")
		}
	}(data.IdAlamatGudang, db.Read, cud_publisher)

	log.Printf("[INFO] Alamat gudang berhasil diubah untuk seller ID %d", data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil diubah.",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Alamat Gudang Seller
// Berfungsi Untuk Menghapus Suatu Alamat Gudang Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusAlamatGudang(ctx context.Context, data PayloadHapusAlamatGudang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusAlamatGudang"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial seller tidak valid untuk ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Kredensial seller tidak valid.",
		}
	}

	var data_alamat models.AlamatGudang

	if err := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Where(&models.AlamatGudang{
		ID:       data.IdGudang,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_alamat.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal masukan data alamat tidak valid",
		}
	}

	var total int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
		IDAlamat: data.IdGudang,
	}).Limit(1).Scan(&total).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if total != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu tidak bisa menghapus alamat ini karna masih digunakan oleh beberapa barangmu alihkan terlebih dahulu",
		}
	}

	var idDataTransaksi int64 = 0

	if err := db.Read.WithContext(ctx).
		Model(&models.Transaksi{}).
		Select("id").
		Where("id_alamat_gudang = ? AND status != ?", data.IdGudang, "Selesai").
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

	if err_hapus := db.Write.WithContext(ctx).Model(&models.AlamatGudang{}).Where(models.AlamatGudang{
		ID:       data.IdGudang,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Delete(&models.AlamatGudang{}).Error; err_hapus != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
		}
	}

	go func(A models.AlamatGudang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(A.IDSeller),
		}
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.AlamatGudang); err != nil {
			fmt.Println("Gagal decr alamat gudang count ke threshold seller")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.AlamatGudangThreshold{}).Where(&sot_threshold.AlamatGudangThreshold{
			IdAlamatGudang: A.ID,
		}).Delete(&sot_threshold.AlamatGudangThreshold{}).Error; err != nil {
			fmt.Printf("Gagal hapus threshold alamat gudang ber id: %v", A.ID)
		}

		deleteAlamatGudangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(A).SetTableName(A.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteAlamatGudangPublish); err != nil {
			fmt.Println("Gagal publish delete alamat gudang ke message broker")
		}

	}(data_alamat, db.Write, cud_publisher)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil dihapus.",
	}
}
