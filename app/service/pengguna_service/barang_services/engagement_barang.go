package pengguna_barang_services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_komentar "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/komentar"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_review "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/review"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

var fieldBarangViewed = "viewed_barang_induk"

const LIMITKERANJANG = 30

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Engagement Barang Level Uncritical
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

type BatchViewUpdate struct {
	UpdateTarget    map[int64]int64
	UpdateExecution chan map[int64]int64
	mutex           sync.RWMutex
	ticker          *time.Ticker
	duration        time.Duration
	Limit           int64
}

func NewBatchViewUpdate(interval int64) *BatchViewUpdate {
	dur := time.Duration(interval) * time.Second
	if dur == 0 {
		dur = 10 * time.Second
	}
	return &BatchViewUpdate{
		UpdateTarget:    make(map[int64]int64),
		UpdateExecution: make(chan map[int64]int64, 100),
		duration:        dur,
		ticker:          time.NewTicker(dur),
	}
}

// 1. Eksekusi update ke database secara asinkronus
func (b *BatchViewUpdate) UpdateToDB(db *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) {
	konteks, batal := context.WithTimeout(context.Background(), settings.TimeoutDatabaseQuery)
	defer batal()
	for dataToUpdate := range b.UpdateExecution {
		if len(dataToUpdate) == 0 {
			continue
		}

		fmt.Printf("[DB] Mengupdate %d data ke database...\n", len(dataToUpdate))
		for id, views := range dataToUpdate {
			if err := db.Write.WithContext(konteks).Model(&sot_models.BarangInduk{}).
				Where("id = ?", id).
				UpdateColumn("viewed", gorm.Expr(fmt.Sprintf("viewed + %v", views))).Error; err != nil {
				fmt.Printf("Gagal memperbarui vies barang ber id %v sebanyak %v view baru", id, views)

				go func(id_barang_induk int64, read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
					konteks, batal := context.WithTimeout(context.Background(), settings.TimeoutContext)
					defer batal()
					var updatedBarangInduk sot_models.BarangInduk = sot_models.BarangInduk{
						ID: 0,
					}

					if err := read.WithContext(konteks).Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{
						ID: int32(id_barang_induk),
					}).Limit(1).Take(&updatedBarangInduk).Error; err != nil {
						fmt.Println("Gagal mengambil data barang induk baru ", err)
					}

					updatedViewBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(updatedBarangInduk).SetTableName("UpdateViewBarangInduk").SetRole(mb_cud_seeders.Seller)
					if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedViewBarangIndukPublish); err != nil {
						fmt.Println("Gagal publish update view barang induk")
					}
				}(id, db.Read, cud_publisher)
			}

		}
	}
}

func (b *BatchViewUpdate) ResetInterval() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.ticker != nil {
		b.ticker.Reset(b.duration)
	}
}

func (b *BatchViewUpdate) WatchInterval() {
	for {
		// Mengambil channel ticker dengan aman
		b.mutex.RLock()
		ch := b.ticker.C
		b.mutex.RUnlock()

		select {
		case <-ch:
			b.mutex.Lock()
			if len(b.UpdateTarget) == 0 {
				b.mutex.Unlock()
				continue
			}

			oldTarget := b.UpdateTarget
			b.UpdateTarget = make(map[int64]int64)
			b.mutex.Unlock()

			// Kirim data ke channel execution
			b.UpdateExecution <- oldTarget
		}
	}
}

// 4. Increment view
func (b *BatchViewUpdate) IncrUpdateViewColumn(id_column int64) {
	b.mutex.Lock()
	if b.UpdateTarget == nil {
		b.UpdateTarget = make(map[int64]int64)
	}

	b.UpdateTarget[id_column]++
	b.Limit++

	// JIKA LIMIT TERCAPAI, LANGSUNG FLUSH KE DB
	if b.Limit >= 2000 {
		oldTarget := b.UpdateTarget
		b.UpdateTarget = make(map[int64]int64) // Kosongkan map untuk batch berikutnya
		b.Limit = 0                            // Reset limit ke 0

		b.mutex.Unlock()

		// Kirim data ke worker DB secara non-blocking
		b.UpdateExecution <- oldTarget
		b.ResetInterval() // Reset ticker juga agar waktu 10 detiknya mulai dari 0 lagi
		return
	}

	// Jangan lupa unlock untuk kondisi normal (limit belum 2000)
	b.mutex.Unlock()

	// Reset waktu tunggu jika limit belum tercapai
	b.ResetInterval()
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur View Barang
// Berfungsi Untuk Menambah View Barang Setiap kali di klik akan menjalankan fungsi ini
// Hanya bersifat menaikan view (increment)
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func ViewBarang(ctx context.Context, b *BatchViewUpdate, data PayloadViewBarang) *response.ResponseForm {
	const services string = "ViewBarang"
	b.IncrUpdateViewColumn(int64(data.ID))

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Likes Barang
// :Berfungsi Untuk Menambah Dan Mengurangi Likes Barang induk dan mencatat barangdisukai
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func LikesBarang(ctx context.Context, data PayloadLikesBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "LikesBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Pengguna tidak ditemukan",
		}
	}

	var id_pengguna_disukai int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangDisukai{}).Select("id").Where(&sot_models.BarangDisukai{
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

	newLikeBarang := sot_models.BarangDisukai{
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

	go func(Lb sot_models.BarangDisukai, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			ID: Lb.IdPengguna,
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Lb.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.BarangDisukai); err != nil {
			fmt.Println("Gagal increment barang disukai pengguna threshold")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.BarangDisukai); err != nil {
			fmt.Println("Gagal increment barang disukai barang induk threshold")
		}

		publishNewBarangDisukai := mb_cud_serializer.NewJsonPayload().SetPayload(Lb).SetTableName(Lb.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func UnlikeBarang(ctx context.Context, data PayloadUnlikeBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UnlikeBarang"

	var barang_disukai sot_models.BarangDisukai
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangDisukai{}).Where(&sot_models.BarangDisukai{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.BarangDisukai{}).Where(&sot_models.BarangDisukai{
		ID: data.IdBarangDisukai,
	}).Delete(&sot_models.BarangDisukai{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bs sot_models.BarangDisukai, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdPengguna := sot_threshold.PenggunaThreshold{
			IdPengguna: Bs.IdPengguna,
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Bs.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := thresholdPengguna.Decrement(konteks, Trh, stsk_pengguna.BarangDisukai); err != nil {
			fmt.Println("Gagal decrement threshold pengguna likes barang")
		}

		if err := thresholdBarangInduk.Decrement(konteks, Trh, stsk_baranginduk.BarangDiDiskon); err != nil {
			fmt.Println("Gagal decrement threshold barang induk likes barang")
		}

		publishDeleteBarangDisukai := mb_cud_serializer.NewJsonPayload().SetPayload(Bs).SetTableName(Bs.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func MasukanKomentarBarang(ctx context.Context, data PayloadMasukanKomentarBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahKomentarBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	NewKomentar := sot_models.Komentar{
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

	go func(K sot_models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(K.IdBarangInduk),
		}

		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: K.ID,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := thresholdKomentar.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold komentar")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal increment total komentar barang induk ke threshold barang induk")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func EditKomentarBarang(ctx context.Context, data PayloadEditKomentarBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditKomentarBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Komentar{}).Select("id").Where(&sot_models.Komentar{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
		ID: data.IdKomentar,
	}).Update("komentar", data.Komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(idKomen int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarData := sot_models.Komentar{}
		if err := Read.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
			ID: idKomen,
		}).Limit(1).Take(&komentarData); err != nil {
			return
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		newUpdateKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(komentarData).SetTableName(komentarData.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func HapusKomentarBarang(ctx context.Context, data PayloadHapusKomentarBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKomentarBarang"

	var Komentar sot_models.Komentar
	if err := db.Read.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
		ID: data.IdKomentar,
	}).Delete(&sot_models.Komentar{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(K sot_models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			ID: int64(K.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal decr komentar barang induk ke threshold barang induk")
		}

		newDeleteKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func MasukanChildKomentar(ctx context.Context, data PayloadMasukanChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanChildKomentar"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data user tidak ditemukan",
		}
	}

	newKomentar := sot_models.KomentarChild{
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

	go func(Kc sot_models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func MentionChildKomentar(ctx context.Context, data PayloadMentionChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MentionChildKomentar"

	newKomentar := sot_models.KomentarChild{
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

	go func(Kc sot_models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func EditChildKomentar(ctx context.Context, data PayloadEditChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditChildKomentar"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data user tidak ditemukan",
		}
	}

	var id_edit_child_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.KomentarChild{}).Select("id").Where(&sot_models.KomentarChild{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
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
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		var dataKomentarChild sot_models.KomentarChild
		if err := Read.WithContext(konteks).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
			ID: IdKc,
		}).Limit(1).Take(&dataKomentarChild).Error; err != nil {
			return
		}

		updateKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKomentarChild).SetTableName(dataKomentarChild.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func HapusChildKomentar(ctx context.Context, data PayloadHapusChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusChildKomentar"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data user tidak ditemukan",
		}
	}

	var childKomentar sot_models.KomentarChild
	if err := db.Read.WithContext(ctx).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
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
	if err := db.Write.Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
		ID: data.IdKomentar,
	}).Delete(&sot_models.KomentarChild{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Kc sot_models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarThreshold := sot_threshold.KomentarThreshold{
			ID: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := komentarThreshold.Decrement(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal decrement komentar child ke threshold komentar")
		}

		deleteKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func TambahKeranjangBarang(ctx context.Context, data PayloadTambahDataKeranjangBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_total []int64
	if err := db.Read.WithContext(ctx).Model(&sot_models.Keranjang{}).Select("id").Where(sot_models.Keranjang{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.Keranjang{}).Select("id").Where(&sot_models.Keranjang{
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

	newKeranjang := sot_models.Keranjang{
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

	go func(K sot_models.Keranjang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
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
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
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

		newKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func EditKeranjangBarang(ctx context.Context, data PayloadEditDataKeranjangBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var id_data_keranjang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Keranjang{}).Select("id").Where(&sot_models.Keranjang{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").Where(&sot_models.VarianBarang{
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
		Model(&sot_models.Keranjang{}).
		Where(&sot_models.Keranjang{
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
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		var dataKeranjang sot_models.Keranjang
		if err := Read.WithContext(konteks).Model(&sot_models.Keranjang{}).Where(&sot_models.Keranjang{
			ID: IdKeranjang,
		}).Limit(1).Take(&dataKeranjang).Error; err != nil {
			return
		}

		updateKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKeranjang).SetTableName(dataKeranjang.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func HapusKeranjangBarang(ctx context.Context, data PayloadHapusDataKeranjangBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKeranjangBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	var dataKeranjang sot_models.Keranjang
	if err := db.Read.WithContext(ctx).Model(&sot_models.Keranjang{}).Where(&sot_models.Keranjang{
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

	if err_hapus := db.Write.WithContext(ctx).Model(&sot_models.Keranjang{}).Where(&sot_models.Keranjang{
		ID:         data.IdKeranjang,
		IdPengguna: data.IdentitasPengguna.ID,
	}).Delete(&sot_models.Keranjang{}).Error; err_hapus != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Dk sot_models.Keranjang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
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
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
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

		deleteKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Dk).SetTableName(Dk.TableName()).SetRole(entity_enums.Pengguna)
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

func BerikanReviewBarang(ctx context.Context, data PayloadBerikanReviewBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "BerikanReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var id_transaksi_data_selesai int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Transaksi{}).Select("id").Where(&sot_models.Transaksi{
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

	newReview := sot_models.Review{
		IdPengguna:    data.IdentitasPengguna.ID,
		IdBarangInduk: int32(data.IdBarangInduk),
		Rating:        data.Rating,
		Ulasan:        data.Ulasan,
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newReview).Error; err != nil {
			return err
		}

		if err := tx.Model(&sot_models.Transaksi{}).Where(&sot_models.Transaksi{
			ID: id_transaksi_data_selesai,
		}).Updates(&sot_models.Transaksi{
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

	go func(IdTransaksi int64, R sot_models.Review, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
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
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
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

		createReviewPublish := mb_cud_serializer.NewJsonPayload().SetPayload(R).SetTableName(R.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createReviewPublish); err != nil {
			fmt.Println("Gagal publish create review ke message broker")
		}

		var DTU sot_models.Transaksi
		if err := Trh.WithContext(ctx).Model(&sot_models.Transaksi{}).Where(&sot_models.Transaksi{
			ID: IdTransaksi,
		}).Limit(1).Scan(&DTU).Error; err != nil {
			fmt.Println("Gagal mendapatkan data transaksi", err)
		}

		updateTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(DTU).SetTableName("UpdateBerikanReviewBarangIIUpdateTransaksi").SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateTransaksiPublish); err != nil {
			fmt.Println("Gagal publish update transaksi ke message broker")
		}

	}(id_transaksi_data_selesai, newReview, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func LikeReviewBarang(ctx context.Context, data PayloadLikeReviewBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "LikeReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var id_review_like int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.ReviewLike{}).
		Select("id").
		Where(&sot_models.ReviewLike{
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

	if err := db.Write.WithContext(ctx).Create(&sot_models.ReviewLike{
		IdPengguna: data.IdentitasPengguna.ID,
		IdReview:   data.IdReview,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdReview int64, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		reviewThreshold := sot_threshold.ReviewThreshold{
			IdReview: IdReview,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := reviewThreshold.Increment(konteks, Trh, stsk_review.ReviewLike); err != nil {
			fmt.Println("Gagal increment count like review ke review threshold")
		}

		var R sot_models.ReviewLike
		if err := Trh.WithContext(ctx).Model(&sot_models.ReviewLike{}).Where(&sot_models.ReviewLike{
			ID: IdReview,
		}).Limit(1).Scan(&R).Error; err != nil {
			fmt.Println("Gagal mendapatkan data review")
		}

		CreateLikeReviewBarang := mb_cud_serializer.NewJsonPayload().SetPayload(R).SetTableName(R.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](ctx_t, publisher, CreateLikeReviewBarang); err != nil {
			fmt.Println("Gagal publish create review ke message broker")
		}
	}(id_review_like, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UnlikeReviewBarang(ctx context.Context, data PayloadUnlikeReviewBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UnlikeReviewBarang"

	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var id_review_like int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.ReviewLike{}).
		Select("id").
		Where(&sot_models.ReviewLike{
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
	if err := db.Write.WithContext(ctx).Delete(&sot_models.ReviewLike{}, id_review_like).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdReview int64, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		reviewThreshold := sot_threshold.ReviewThreshold{
			IdReview: IdReview,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
		defer cancel()

		if err := reviewThreshold.Increment(konteks, Trh, stsk_review.ReviewDislike); err != nil {
			fmt.Println("Gagal increment count like review dislike ke review threshold")
		}

		var R sot_models.ReviewLike
		if err := Trh.WithContext(ctx).Model(&sot_models.ReviewLike{}).Where(&sot_models.ReviewLike{
			ID: IdReview,
		}).Limit(1).Scan(&R).Error; err != nil {
			fmt.Println("Gagal mendapatkan data review")
		}

		DeleteUnLikeReviewBarang := mb_cud_serializer.NewJsonPayload().SetPayload(R).SetTableName(R.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](ctx_t, publisher, DeleteUnLikeReviewBarang); err != nil {
			fmt.Println("Gagal publish create review ke message broker")
		}
	}(id_review_like, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
