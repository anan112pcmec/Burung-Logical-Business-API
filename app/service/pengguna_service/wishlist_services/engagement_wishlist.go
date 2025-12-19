package pengguna_wishlist_services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

func TambahBarangKeWishlist(ctx context.Context, data PayloadTambahBarangKeWishlist, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahBarangKeWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	var id_data_wishlist = 0
	if err := db.Read.WithContext(ctx).Model(&models.Wishlist{}).Select("id").Where(&models.Wishlist{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&id_data_wishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_wishlist != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah memasukan barang itu ke dalam wishlist",
		}
	}

	newWishlist := models.Wishlist{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IdBarangInduk,
	}

	if err := db.Write.WithContext(ctx).Create(&newWishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(W models.Wishlist, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		penggunaThreshold := sot_threshold.PenggunaThreshold{
			IdPengguna: W.IdPengguna,
		}

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(W.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := penggunaThreshold.Increment(konteks, Trh, stsk_pengguna.Wishlist); err != nil {
			fmt.Println("Gagal increment wishlist counter ke pengguna threshold")
		}

		if err := barangIndukThreshold.Increment(konteks, Trh, stsk_baranginduk.Wishlist); err != nil {
			fmt.Println("Gagal increment wishlist counter ke barang induk threshold")
		}

		createWishlistPublish := mb_cud_serializer.NewJsonPayload().SetPayload(W).SetTableName(W.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createWishlistPublish); err != nil {
			fmt.Println("Gagal publish create wishlist ke message broker")
		}
	}(newWishlist, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusBarangDariWishlist(ctx context.Context, data PayloadHapusBarangDariWishlist, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusBarangDariWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	var dataWishlist models.Wishlist
	if err := db.Read.WithContext(ctx).Model(&models.Wishlist{}).Where(&models.Wishlist{
		ID:         data.IdWishlist,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&dataWishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if dataWishlist.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data wishlist tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Wishlist{}).Where(&models.Wishlist{
		ID: data.IdWishlist,
	}).Delete(&models.Wishlist{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(W models.Wishlist, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		penggunaThreshold := sot_threshold.PenggunaThreshold{
			IdPengguna: W.IdPengguna,
		}

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(W.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := penggunaThreshold.Decrement(konteks, Trh, stsk_pengguna.Wishlist); err != nil {
			fmt.Println("Gagal decrement wishlist counter ke pengguna threshold")
		}

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Wishlist); err != nil {
			fmt.Println("Gagal decrement wishlist counter ke barang induk threshold")
		}

		createWishlistPublish := mb_cud_serializer.NewJsonPayload().SetPayload(W).SetTableName(W.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createWishlistPublish); err != nil {
			fmt.Println("Gagal publish delete wishlist ke message broker")
		}
	}(dataWishlist, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
