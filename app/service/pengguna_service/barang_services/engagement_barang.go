package pengguna_service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/entity"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

var fieldBarangViewed = "viewed_barang_induk"

const LIMITKERANJANG = 30

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Engagement Barang Level Uncritical
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur View Barang
// Berfungsi Untuk Menambah View Barang Setiap kali di klik akan menjalankan fungsi ini
// Hanya bersifat menaikan view (increment)
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func ViewBarang(data PayloadViewBarang, rds *redis.Client, db *gorm.DB) {
	ctx := context.Background()
	key := fmt.Sprintf("barang:%d", data.ID)

	// Jika gagal increment di Redis -> fallback update ke DB (asynchronous)
	if err := rds.HIncrBy(ctx, key, fieldBarangViewed, 1).Err(); err != nil {
		go func() {
			_ = db.Model(&models.BarangInduk{}).
				Where("id = ?", data.ID).
				UpdateColumn("viewed", gorm.Expr("viewed + 1")).Error
		}()
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Likes Barang
// :Berfungsi Untuk Menambah Dan Mengurangi Likes Barang induk dan mencatat barangdisukai
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func LikesBarang(ctx context.Context, data PayloadLikesBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "LikesBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Pengguna tidak ditemukan",
		}
	}

	var id_pengguna_disukai int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangDisukai{}).Select("id").Where(&models.BarangDisukai{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IDBarangInduk,
	}).Limit(1).Scan(&id_pengguna_disukai).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_pengguna_disukai != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah menyukai barang itu",
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.BarangDisukai{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IDBarangInduk,
	}).Error; err != nil {
		fmt.Println("Gagal likes:", err)
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

func UnlikeBarang(ctx context.Context, data PayloadUnlikeBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "UnlikeBarang"

	var id_data_barang_disukai int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangDisukai{}).Select("id").Where(&models.BarangDisukai{
		ID:            data.IdBarangDisukai,
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&id_data_barang_disukai).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_barang_disukai == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BarangDisukai{}).Where(&models.BarangDisukai{
		ID: data.IdBarangDisukai,
	}).Delete(&models.BarangDisukai{}).Error; err != nil {
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

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Engagement Barang Level Critical
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanKomentarBarang(ctx context.Context, data PayloadMasukanKomentarBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahKomentarBarang"

	if err := db.Write.WithContext(ctx).Create(&models.Komentar{
		IdBarangInduk: data.IdBarangInduk,
		IdEntity:      data.IdentitasPengguna.ID,
		JenisEntity:   entity_enums.Pengguna,
		Komentar:      data.Komentar,
		IsSeller:      false,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal memposting komentar",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditKomentarBarang(ctx context.Context, data PayloadEditKomentarBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditKomentarBarang"

	var id_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Komentar{}).Select("id").Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
	}).Limit(1).Scan(&id_komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_komentar == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
		ID: data.IdKomentar,
	}).Update("komentar", data.Komentar).Error; err != nil {
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

func HapusKomentarBarang(ctx context.Context, data PayloadHapusKomentarBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusKomentarBarang"

	var id_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Komentar{}).Select("id").Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
	}).Limit(1).Scan(&id_komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_komentar == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
		ID: data.IdKomentar,
	}).Delete(&models.Komentar{}).Error; err != nil {
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

func MasukanChildKomentar(ctx context.Context, data PayloadMasukanChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "MasukanChildKomentar"

	if err := db.Write.WithContext(ctx).Create(&models.KomentarChild{
		IdKomentar:  data.IdKomentarBarang,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
		IsiKomentar: data.Komentar,
		IsSeller:    false,
	}).Error; err != nil {
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

func MentionChildKomentar(ctx context.Context, data PayloadMentionChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "MentionChildKomentar"

	if err := db.Write.WithContext(ctx).Create(&models.KomentarChild{
		IdKomentar:  data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
		IsiKomentar: data.Komentar,
		IsSeller:    false,
		Mention:     data.UsernameMentioned,
	}).Error; err != nil {
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

func EditChildKomentar(ctx context.Context, data PayloadEditChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditChildKomentar"

	var id_edit_child_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KomentarChild{}).Select("id").Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
	}).Limit(1).Scan(&id_edit_child_komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_edit_child_komentar == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
		ID: data.IdKomentar,
	}).Update("komentar", data.Komentar).Error; err != nil {
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

func HapusChildKomentar(ctx context.Context, data PayloadHapusChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusChildKomentar"

	var id_edit_child_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KomentarChild{}).Select("id").Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
	}).Limit(1).Scan(&id_edit_child_komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_edit_child_komentar == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}
	if err := db.Write.Model(&models.KomentarChild{}).Where(&models.KomentarChild{
		ID: data.IdKomentar,
	}).Delete(&models.KomentarChild{}).Error; err != nil {
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

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Tambah Keranjang Barang
// :Berfungsi Untuk menambahkan sebuah barang ke keranjang pengguna tertentu
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TambahKeranjangBarang(ctx context.Context, data PayloadTambahDataKeranjangBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_total []int64
	if err := db.Read.WithContext(ctx).Model(&models.Keranjang{}).Select("id").Where(models.Keranjang{
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(LIMITKERANJANG).Scan(&id_total).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if len(id_total) >= LIMITKERANJANG {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  fmt.Sprintf("Gagal keranjang sudah penuh max sebanyak %v barang", LIMITKERANJANG),
		}
	}

	var id_data_keranjang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Keranjang{}).Select("id").Where(&models.Keranjang{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdSeller:      data.IdSeller,
		IdBarangInduk: data.IdBarangInduk,
		IdKategori:    data.IdKategori,
	}).Limit(1).Scan(&id_data_keranjang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_keranjang != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah memiliki barang itu di keranjang mu",
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.Keranjang{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdSeller:      data.IdSeller,
		IdBarangInduk: data.IdBarangInduk,
		IdKategori:    data.IdKategori,
		Status:        "Ready",
		Jumlah:        0,
	}).Error; err != nil {
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

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Edit Keranjang Barang
// :Berfungsi Untuk mengedit sebuah count dari keranjang pengguna
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditKeranjangBarang(ctx context.Context, data PayloadEditDataKeranjangBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_data_keranjang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Keranjang{}).Select("id").Where(&models.Keranjang{
		ID:         data.IdKeranjang,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_data_keranjang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_keranjang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data keranjang tidak ditemukan",
		}
	}

	var id_stok []int64
	if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").Where(&models.VarianBarang{
		IdKategori:    data.IdKategori,
		IdBarangInduk: data.IdBarangInduk,
		Status:        "Ready",
	}).Limit(int(data.Jumlah)).Scan(&id_stok).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if int64(len(id_stok)) < data.Jumlah {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Gagal barang melebihi stok yang tersedia",
		}
	}

	if err := db.Write.WithContext(ctx).
		Model(&models.Keranjang{}).
		Where(&models.Keranjang{
			ID: data.IdKeranjang,
		}).
		Update("jumlah", data.Jumlah).Error; err != nil {
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

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Hapus Keranjang Barang
// :Berfungsi Untuk menghapus suatu barang dari keranjang pengguna tertentu
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusKeranjangBarang(ctx context.Context, data PayloadHapusDataKeranjangBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_data_keranjang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Keranjang{}).Select("id").Where(&models.Keranjang{
		ID:         data.IdKeranjang,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&id_data_keranjang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_keranjang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data keranjang tidak ditemukan",
		}
	}

	if err_hapus := db.Write.WithContext(ctx).Model(&models.Keranjang{}).Where(&models.Keranjang{
		ID:         data.IdKeranjang,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Delete(&models.Keranjang{}).Error; err_hapus != nil {
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

func BerikanReviewBarang(ctx context.Context, data PayloadBerikanReviewBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "BerikanReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var id_transaksi_data_selesai int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Transaksi{}).Select("id").Where(&models.Transaksi{
		IdBarangInduk: data.IdBarangInduk,
		IdPengguna:    data.IdentitasPengguna.ID,
		Status:        transaksi_enums.Selesai,
		Reviewed:      false,
	}).Limit(1).Scan(&id_transaksi_data_selesai).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_transaksi_data_selesai == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal data transaksi tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.Review{
			IdPengguna:    data.IdentitasPengguna.ID,
			IdBarangInduk: int32(data.IdBarangInduk),
			Rating:        data.Rating,
			Ulasan:        data.Ulasan,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Transaksi{}).Where(&models.Transaksi{
			ID: id_transaksi_data_selesai,
		}).Updates(&models.Transaksi{
			Reviewed: true,
		}).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
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

func LikeReviewBarang(ctx context.Context, data PayloadLikeReviewBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "LikeReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var id_review_like int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.ReviewLike{}).
		Select("id").
		Where(&models.ReviewLike{
			IdPengguna: data.IdentitasPengguna.ID,
			IdReview:   data.IdReview,
		}).
		Limit(1).
		Scan(&id_review_like).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_review_like != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah like review itu",
		}
	}

	if err := db.Write.WithContext(ctx).Create(&models.ReviewLike{
		IdPengguna: data.IdentitasPengguna.ID,
		IdReview:   data.IdReview,
	}).Error; err != nil {
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

func UnlikeReviewBarang(ctx context.Context, data PayloadUnlikeReviewBarang, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "UnlikeReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var id_review_like int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.ReviewLike{}).
		Select("id").
		Where(&models.ReviewLike{
			IdPengguna: data.IdentitasPengguna.ID,
			IdReview:   data.IdReview,
		}).
		Limit(1).
		Scan(&id_review_like).Error; err != nil {

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_review_like == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data like tidak ditemukan",
		}
	}

	// Hapus like
	if err := db.Write.WithContext(ctx).Delete(&models.ReviewLike{}, id_review_like).Error; err != nil {
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
