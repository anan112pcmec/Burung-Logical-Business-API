package pengguna_wishlist_services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_wishlist "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/wishlist"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"

)

func TambahWishlist(ctx context.Context, data PayloadTambahWishlist, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahWishlistBaru"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Data sesi tidak valid",
		}
	}

	if data.Visibility != "Public" && data.Visibility != "Private" {
		return &response.ResponseForm{
			Status:   http.StatusUnsupportedMediaType,
			Services: services,
			Message:  "jenis visibility tidak valid",
		}
	}

	// validasi nama wishlist sama
	var id_wishlist int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Wishlist{}).Select("id").Where(&sot_models.Wishlist{
		IdPengguna: data.IdentitasPengguna.ID,
		Nama:       data.Nama,
	}).Limit(1).Take(&id_wishlist).Error; err != nil {
		fmt.Println("Gagal mendapatkan data error: ", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal coba lagi lain waktu",
		}
	}

	if id_wishlist != 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Kamu sudah memiliki wishlist dengan nama yang sama",
		}
	}

	newWishlist := sot_models.Wishlist{
		IdPengguna: data.IdentitasPengguna.ID,
		Nama:       data.Nama,
		Deskripsi:  data.Deskripsi,
		Visibility: data.Visibility,
	}

	if err := db.Write.WithContext(ctx).Create(&newWishlist).Error; err != nil {
		fmt.Println("Gagal membuat wishlist dengan error: ", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal coba lagi lain waktu",
		}
	}

	go func(W sot_models.Wishlist, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		penggunaThreshold := sot_threshold.PenggunaThreshold{
			IdPengguna: W.IdPengguna,
		}

		wishlistThreshold := sot_threshold.WishlistThreshold{
			IdWishlist: W.ID,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := penggunaThreshold.Increment(konteks, Trh, stsk_pengguna.Wishlist); err != nil {
			fmt.Println("Gagal increment wishlist counter ke pengguna threshold")
		}

		if err := wishlistThreshold.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal inisialisasi wishlist threshold")
		}

		createWishlistPublish := mb_cud_serializer.NewJsonPayload().SetPayload(W).SetTableName(W.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createWishlistPublish); err != nil {
			fmt.Println("Gagal publish create wishlist ke message broker")
		}
	}(newWishlist, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil menambahkan wishlist baru",
	}
}

func TambahBarangKeWishlist(ctx context.Context, data PayloadTambahBarangKeWishlist, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahBarangKeWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data user tidak valid",
		}
	}

	// Validasi kepemilikan wishlist
	var idWishlist int64 = 0
	if err := db.Read.WithContext(ctx).Select("id").Where(&sot_models.Wishlist{
		ID:         data.IdWishlist,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Take(&idWishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal wishlist tidak ditemukan",
		}
	}

	// Cek apakah barang sudah ada di dalam wishlist tersebut menggunakan struct gorm
	var id_wishlist_product int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.WishlistProduct{}).Select("id").Where(&sot_models.WishlistProduct{
		IdWishlist:    data.IdWishlist,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&id_wishlist_product).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_wishlist_product != 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal kamu sudah memasukan barang itu ke dalam wishlist ini",
		}
	}

	newWishlistProduct := sot_models.WishlistProduct{
		IdWishlist:    data.IdWishlist,
		IdBarangInduk: data.IdBarangInduk,
	}

	if err := db.Write.WithContext(ctx).Create(&newWishlistProduct).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(WP sot_models.WishlistProduct, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		wishlistThreshold := sot_threshold.WishlistThreshold{
			IdWishlist: WP.IdWishlist,
		}

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(WP.IdBarangInduk),
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := wishlistThreshold.Increment(konteks, Trh, stsk_wishlist.WishlistProduct); err != nil {
			fmt.Println("Gagal increment wishlist product counter ke wishlist threshold")
		}

		if err := barangIndukThreshold.Increment(konteks, Trh, stsk_baranginduk.Wishlist); err != nil {
			fmt.Println("Gagal increment wishlist counter ke barang induk threshold")
		}

		createWPPublish := mb_cud_serializer.NewJsonPayload().SetPayload(WP).SetTableName("wishlist_product").SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createWPPublish); err != nil {
			fmt.Println("Gagal publish create wishlist product ke message broker")
		}
	}(newWishlistProduct, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusBarangDariWishlist(ctx context.Context, data PayloadHapusBarangDariWishlist, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusBarangDariWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data user tidak valid",
		}
	}

	var dataWishlistProduct sot_models.WishlistProduct
	if err := db.Read.WithContext(ctx).Where(&sot_models.WishlistProduct{
		ID: data.IdWishlistProduct,
	}).Limit(1).Scan(&dataWishlistProduct).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if dataWishlistProduct.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data barang di wishlist tidak ditemukan",
		}
	}

	// Validasi kepemilikan lewat folder wishlist-nya menggunakan struct gorm
	var dataWishlist sot_models.Wishlist
	if err := db.Read.WithContext(ctx).Where(&sot_models.Wishlist{
		ID:         dataWishlistProduct.IdWishlist,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Take(&dataWishlist).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal akses ditolak",
		}
	}

	if err := db.Write.WithContext(ctx).Where(&sot_models.WishlistProduct{
		ID: data.IdWishlistProduct,
	}).Delete(&sot_models.WishlistProduct{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(WP sot_models.WishlistProduct, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		wishlistThreshold := sot_threshold.WishlistThreshold{
			IdWishlist: WP.IdWishlist,
		}

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(WP.IdBarangInduk),
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := wishlistThreshold.Decrement(konteks, Trh, stsk_wishlist.WishlistProduct); err != nil {
			fmt.Println("Gagal decrement wishlist product counter ke wishlist threshold")
		}

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Wishlist); err != nil {
			fmt.Println("Gagal decrement wishlist counter ke barang induk threshold")
		}

		deleteWPPublish := mb_cud_serializer.NewJsonPayload().SetPayload(WP).SetTableName("wishlist_product").SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteWPPublish); err != nil {
			fmt.Println("Gagal publish delete wishlist product ke message broker")
		}
	}(dataWishlistProduct, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusWishlist(ctx context.Context, data PayloadHapusWishlist, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services = "HapusWishlist"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data user tidak valid",
		}
	}

	var dataWishlist sot_models.Wishlist
	if err := db.Read.WithContext(ctx).Where(&sot_models.Wishlist{
		ID:         data.IdWishlist,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Take(&dataWishlist).Error; err != nil {
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

	if err := db.Write.WithContext(ctx).Where(&sot_models.Wishlist{
		ID: data.IdWishlist,
	}).Delete(&sot_models.Wishlist{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(W sot_models.Wishlist, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		penggunaThreshold := sot_threshold.PenggunaThreshold{
			IdPengguna: W.IdPengguna,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := penggunaThreshold.Decrement(konteks, Trh, stsk_pengguna.Wishlist); err != nil {
			fmt.Println("Gagal decrement wishlist counter ke pengguna threshold")
		}

		deleteWishlistPublish := mb_cud_serializer.NewJsonPayload().SetPayload(W).SetTableName(W.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteWishlistPublish); err != nil {
			fmt.Println("Gagal publish delete wishlist ke message broker")
		}
	}(dataWishlist, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
