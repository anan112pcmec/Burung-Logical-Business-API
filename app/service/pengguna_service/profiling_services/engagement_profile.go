package pengguna_profiling_services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	cache_db_entity_sessioning_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/cache_database/entity_sessioning/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	particular_profiling_pengguna "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/profiling_services/particular_profiling"
	response_profiling_pengguna "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/profiling_services/response_profiling"
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Ubah Personal Profiling Pengguna
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahPersonalProfilingPengguna(ctx context.Context, data PayloadPersonalProfilingPengguna, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	var wg sync.WaitGroup

	services := "UbahPersonalProfilingPengguna"
	var hasil_update_gmail particular_profiling_pengguna.ResponseUbahEmail
	var hasil_update_username particular_profiling_pengguna.ResponseUbahUsername
	var hasil_update_nama particular_profiling_pengguna.ResponseUbahNama

	pengguna, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if data.EmailUpdate != "" && data.EmailUpdate != pengguna.Email && data.EmailUpdate != "not" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasil_update_gmail = particular_profiling_pengguna.UbahEmailPengguna(ctx, data.IdentitasPengguna.ID, data.EmailUpdate, db)
		}()
	}

	if data.UsernameUpdate != "" && data.UsernameUpdate != pengguna.Username && data.UsernameUpdate != "not" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasil_update_username = particular_profiling_pengguna.UbahUsernamePengguna(ctx, db, data.IdentitasPengguna.ID, data.UsernameUpdate)
		}()
	}

	if data.NamaUpdate != "" && data.NamaUpdate != pengguna.Nama && data.NamaUpdate != "not" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasil_update_nama = particular_profiling_pengguna.UbahNamaPengguna(ctx, data.IdentitasPengguna.ID, data.NamaUpdate, db)
		}()
	}

	wg.Wait()

	go func(Bpengguna models.Pengguna, Read *gorm.DB, rds_sesi *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		KeyLama := cache_db_entity_sessioning_seeders.SetSessionKey[*models.Pengguna](&Bpengguna)
		var uptodatePenggunaData models.Pengguna
		if err := Read.WithContext(konteks).Model(&models.Pengguna{}).Where(&models.Pengguna{
			ID: Bpengguna.ID,
		}).Limit(1).Take(&uptodatePenggunaData); err != nil {
			fmt.Println("Gagal mengambil data pengguna terbaru")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionKey(konteks, &uptodatePenggunaData, KeyLama, rds_sesi); err != nil {
			fmt.Println("Gagal update cache key dan data")
			return
		}

		updatedPenggunaDataPublish := mb_cud_serializer.NewJsonPayload().SetPayload(uptodatePenggunaData).SetTableName(uptodatePenggunaData.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedPenggunaDataPublish); err != nil {
			fmt.Println("Gagal publish update data pengguna ke message broker")
		}
	}(pengguna, db.Read, rds_session, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil Memperbarui",
		Payload: response_profiling_pengguna.ResponsePersonalProfilingPengguna{
			UpdateNama:     hasil_update_nama,
			UpdateUsername: hasil_update_username,
			UpdateGmail:    hasil_update_gmail,
		},
	}
}
