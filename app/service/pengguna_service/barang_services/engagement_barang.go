package pengguna_service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_komentar "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/komentar"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_review "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/review"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
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

func LikesBarang(ctx context.Context, data PayloadLikesBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "LikesBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	newLikeBarang := models.BarangDisukai{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IDBarangInduk,
	}

	if err := db.Write.WithContext(ctx).Create(&newLikeBarang).Error; err != nil {
		fmt.Println("Gagal likes:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Lb models.BarangDisukai, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			ID: Lb.IdPengguna,
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Lb.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.BarangDisukai); err != nil {
			fmt.Println("Gagal increment barang disukai pengguna threshold")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.BarangDisukai); err != nil {
			fmt.Println("Gagal increment barang disukai barang induk threshold")
		}

		publishNewBarangDisukai := mb_cud_serializer.NewJsonPayload().SetPayload(Lb).SetTableName(Lb.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, publishNewBarangDisukai); err != nil {
			fmt.Println("Gagal publish create barang disukai ke message broker")
		}

	}(newLikeBarang, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UnlikeBarang(ctx context.Context, data PayloadUnlikeBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UnlikeBarang"

	var barang_disukai models.BarangDisukai
	if err := db.Read.WithContext(ctx).Model(&models.BarangDisukai{}).Where(&models.BarangDisukai{
		ID:            data.IdBarangDisukai,
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(1).Scan(&barang_disukai).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if barang_disukai.ID == 0 {
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

	go func(Bs models.BarangDisukai, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: Bs.IdPengguna,
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Bs.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdPengguna.Decrement(konteks, Trh, stsk_pengguna.BarangDisukai); err != nil {
			fmt.Println("Gagal decrement threshold pengguna likes barang")
		}

		if err := thresholdBarangInduk.Decrement(konteks, Trh, stsk_baranginduk.BarangDiDiskon); err != nil {
			fmt.Println("Gagal decrement threshold barang induk likes barang")
		}

		publishDeleteBarangDisukai := mb_cud_serializer.NewJsonPayload().SetPayload(Bs).SetTableName(Bs.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, publishDeleteBarangDisukai); err != nil {
			fmt.Println("Gagal publish delete barang disukai ke message broker")
		}

	}(barang_disukai, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Engagement Barang Level Critical
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanKomentarBarang(ctx context.Context, data PayloadMasukanKomentarBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahKomentarBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	NewKomentar := models.Komentar{
		IdBarangInduk: data.IdBarangInduk,
		IdEntity:      data.IdentitasPengguna.ID,
		JenisEntity:   entity_enums.Pengguna,
		Komentar:      data.Komentar,
		IsSeller:      false,
	}

	if err := db.Write.WithContext(ctx).Create(&NewKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal memposting komentar",
		}
	}

	go func(K models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(K.IdBarangInduk),
		}

		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: K.ID,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdKomentar.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold komentar")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal increment total komentar barang induk ke threshold barang induk")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar baru barang induk ke message broker")
		}

	}(NewKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditKomentarBarang(ctx context.Context, data PayloadEditKomentarBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditKomentarBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

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

	go func(idKomen int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarData := models.Komentar{}
		if err := Read.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
			ID: idKomen,
		}).Limit(1).Take(&komentarData); err != nil {
			return
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		newUpdateKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(komentarData).SetTableName(komentarData.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newUpdateKomentarPublish); err != nil {
			fmt.Println("Gagal publish update komentar barang ke message broker")
		}

	}(id_komentar, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusKomentarBarang(ctx context.Context, data PayloadHapusKomentarBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKomentarBarang"

	var Komentar models.Komentar
	if err := db.Read.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
	}).Limit(1).Scan(&Komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if Komentar.ID == 0 {
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

	go func(K models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			ID: int64(K.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal decr komentar barang induk ke threshold barang induk")
		}

		newDeleteKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newDeleteKomentarPublish); err != nil {
			fmt.Println("Gagal publish delete komentar ke message broker")
		}

	}(Komentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func MasukanChildKomentar(ctx context.Context, data PayloadMasukanChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanChildKomentar"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data user tidak ditemukan",
		}
	}

	newKomentar := models.KomentarChild{
		IdKomentar:  data.IdKomentarBarang,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
		IsiKomentar: data.Komentar,
		IsSeller:    false,
	}

	if err := db.Write.WithContext(ctx).Create(&newKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Kc models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar reply ke message broker")
		}
	}(newKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func MentionChildKomentar(ctx context.Context, data PayloadMentionChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MentionChildKomentar"

	newKomentar := models.KomentarChild{
		IdKomentar:  data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
		IsiKomentar: data.Komentar,
		IsSeller:    false,
	}

	if err := db.Write.WithContext(ctx).Create(&newKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Kc models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar reply ke message broker")
		}
	}(newKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditChildKomentar(ctx context.Context, data PayloadEditChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditChildKomentar"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data user tidak ditemukan",
		}
	}

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

	go func(IdKc int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKomentarChild models.KomentarChild
		if err := Read.WithContext(konteks).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
			ID: IdKc,
		}).Limit(1).Take(&dataKomentarChild).Error; err != nil {
			return
		}

		updateKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKomentarChild).SetTableName(dataKomentarChild.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateKomentarChildPublish); err != nil {
			fmt.Println("Gagal publish update child komentar ke message broker")
		}
	}(id_edit_child_komentar, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusChildKomentar(ctx context.Context, data PayloadHapusChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusChildKomentar"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data user tidak ditemukan",
		}
	}

	var childKomentar models.KomentarChild
	if err := db.Read.WithContext(ctx).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    data.IdentitasPengguna.ID,
		JenisEntity: entity_enums.Pengguna,
	}).Limit(1).Scan(&childKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if childKomentar.ID == 0 {
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

	go func(Kc models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarThreshold := sot_threshold.KomentarThreshold{
			ID: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := komentarThreshold.Decrement(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal decrement komentar child ke threshold komentar")
		}

		deleteKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteKomentarChildPublish); err != nil {
			fmt.Println("Gagal publish delete komentar child ke message broker")
		}
	}(childKomentar, db.Write, cud_publisher)
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

func TambahKeranjangBarang(ctx context.Context, data PayloadTambahDataKeranjangBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	newKeranjang := models.Keranjang{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdSeller:      data.IdSeller,
		IdBarangInduk: data.IdBarangInduk,
		IdKategori:    data.IdKategori,
		Status:        "Ready",
		Jumlah:        0,
	}

	if err := db.Write.WithContext(ctx).Create(&newKeranjang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(K models.Keranjang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: K.IdPengguna,
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(K.IdBarangInduk),
		}

		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: K.IdKategori,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.Keranjang); err != nil {
			fmt.Println("Gagal increment Keranjang count pada pengguna threshold")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Keranjang); err != nil {
			fmt.Println("Gagal increment Keranjang count pada barang induk threshold")
		}

		if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.Keranjang); err != nil {
			fmt.Println("Gagal increment Keranjang count pada kategori barang threshold")
		}

		newKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKeranjangPublish); err != nil {
			fmt.Println("Gagal publish create new keranjang ke message broker")
		}
	}(newKeranjang, db.Write, cud_publisher)

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

func EditKeranjangBarang(ctx context.Context, data PayloadEditDataKeranjangBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdKeranjang int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKeranjang models.Keranjang
		if err := Read.WithContext(konteks).Model(&models.Keranjang{}).Where(&models.Keranjang{
			ID: IdKeranjang,
		}).Limit(1).Take(&dataKeranjang).Error; err != nil {
			return
		}

		updateKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKeranjang).SetTableName(dataKeranjang.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateKeranjangPublish); err != nil {
			fmt.Println("Gagal publish update keranjang ke message broker")
		}
	}(id_data_keranjang, db.Read, cud_publisher)

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

func HapusKeranjangBarang(ctx context.Context, data PayloadHapusDataKeranjangBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var dataKeranjang models.Keranjang
	if err := db.Read.WithContext(ctx).Model(&models.Keranjang{}).Where(&models.Keranjang{
		ID:         data.IdKeranjang,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Limit(1).Scan(&dataKeranjang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if dataKeranjang.ID == 0 {
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

	go func(Dk models.Keranjang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		penggunaThreshold := sot_threshold.PenggunaThreshold{
			IdPengguna: Dk.IdPengguna,
		}

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Dk.IdBarangInduk),
		}

		kategoriBarangThreshold := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Dk.IdKategori,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := penggunaThreshold.Decrement(konteks, Trh, stsk_pengguna.Keranjang); err != nil {
			fmt.Println("Gagal decr count keranjang ke threshold pengguna")
		}

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Keranjang); err != nil {
			fmt.Println("Gagal decr count keranjang ke threshold barang induk")
		}

		if err := kategoriBarangThreshold.Decrement(konteks, Trh, stsk_kategori_barang.Keranjang); err != nil {
			fmt.Println("Gagal decr count keranjang ke threshold kategori barang")
		}

		deleteKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dk).SetTableName(Dk.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteKeranjangPublish); err != nil {
			fmt.Println("Gagal publish delete keranjang ke message broker")
		}
	}(dataKeranjang, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func BerikanReviewBarang(ctx context.Context, data PayloadBerikanReviewBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "BerikanReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	newReview := models.Review{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: int32(data.IdBarangInduk),
		Rating:        data.Rating,
		Ulasan:        data.Ulasan,
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newReview).Error; err != nil {
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

	go func(R models.Review, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		penggunaThreshold := sot_threshold.PenggunaThreshold{
			IdPengguna: R.IdPengguna,
		}

		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(R.IdBarangInduk),
		}

		reviewThreshold := sot_threshold.ReviewThreshold{
			IdReview: R.ID,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := penggunaThreshold.Increment(konteks, Trh, stsk_pengguna.Review); err != nil {
			fmt.Println("Gagal increment count review ke threshold pengguna")
		}

		if err := barangIndukThreshold.Increment(konteks, Trh, stsk_baranginduk.Review); err != nil {
			fmt.Println("Gagal increment count review ke threshold barang induk")
		}

		if err := reviewThreshold.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold review")
		}

		createReviewPublish := mb_cud_serializer.NewJsonPayload().SetPayload(R).SetTableName(R.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createReviewPublish); err != nil {
			fmt.Println("Gagal publish create review ke message broker")
		}

	}(newReview, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func LikeReviewBarang(ctx context.Context, data PayloadLikeReviewBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client) *response.ResponseForm {
	services := "LikeReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdReview int64, Trh *gorm.DB) {
		reviewThreshold := sot_threshold.ReviewThreshold{
			IdReview: IdReview,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := reviewThreshold.Increment(konteks, Trh, stsk_review.ReviewLike); err != nil {
			fmt.Println("Gagal increment count like review ke review threshold")
		}
	}(id_review_like, db.Write)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UnlikeReviewBarang(ctx context.Context, data PayloadUnlikeReviewBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client) *response.ResponseForm {
	services := "UnlikeReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
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

	go func(IdReview int64, Trh *gorm.DB) {
		reviewThreshold := sot_threshold.ReviewThreshold{
			IdReview: IdReview,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := reviewThreshold.Increment(konteks, Trh, stsk_review.ReviewDislike); err != nil {
			fmt.Println("Gagal increment count like review dislike ke review threshold")
		}
	}(id_review_like, db.Write)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
