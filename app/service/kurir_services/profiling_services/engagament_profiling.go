package kurir_profiling_service

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
	particular_profiling_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/profiling_services/particular_profiling"
	response_profiling_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/profiling_services/response_profiling"
)

func PersonalProfilingKurir(ctx context.Context, data PayloadPersonalProfilingKurir, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	var wg sync.WaitGroup
	services := "GeneralProfilingKurir"

	if _, status := data.IdentitasKurir.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kurir tidak ditemukan",
		}
	}

	var hasilresponsenama particular_profiling_kurir.ResponseUbahNama
	var hasilresponseusername particular_profiling_kurir.ResponseUbahUsername
	var hasilresponseemail particular_profiling_kurir.ResponseUbahGmail

	if data.Nama != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasilresponsenama = particular_profiling_kurir.UbahNama(data.IdentitasKurir.IdKurir, data.IdentitasKurir.UsernameKurir, data.Nama, db)
		}()
	}

	if data.Username != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasilresponseusername = particular_profiling_kurir.UbahUsernameKurir(db, data.IdentitasKurir.IdKurir, data.Username)
		}()
	}

	if data.Email != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hasilresponseemail = particular_profiling_kurir.UbahEmail(data.IdentitasKurir.IdKurir, data.Username, data.Email, db)
		}()
	}

	wg.Wait()

	go func(IdKurir int64, Read *gorm.DB, rds_sesi *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataUpdatedKurir models.Kurir
		if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
			ID: IdKurir,
		}).Limit(1).Take(&dataUpdatedKurir).Error; err != nil {
			fmt.Println("Gagal mendapatakan data updated kurir")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionKey[*models.Kurir](konteks, &dataUpdatedKurir, cache_db_entity_sessioning_seeders.SetSessionKey[*models.Kurir](&dataUpdatedKurir), rds_sesi); err != nil {
			fmt.Println("Gagal memperbarui session kurir data ke cache database")
		}

		kurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedKurir).SetTableName(dataUpdatedKurir.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirDataUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated kurir data ke message broker")
		}

	}(data.IdentitasKurir.IdKurir, db.Read, rds_session, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_profiling_kurir.ResponseProfilingPersonalKurir{
			UpdateNama:     hasilresponsenama,
			UpdateUsername: hasilresponseusername,
			UpdateEmail:    hasilresponseemail,
		},
	}
}

func GeneralProfilingKurir(ctx context.Context, data PayloadGeneralProfiling, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "GeneralProfilingKurir"
	var hasil_update_deskripsi particular_profiling_kurir.ResponseUbahDeskripsi

	_, status := data.DataIdentitas.Validating(ctx, db.Read, rds_session)

	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
		}
	}

	if data.Deskripsi != "" {
		hasil_update_deskripsi = particular_profiling_kurir.UbahDeskripsi(data.DataIdentitas.IdKurir, data.DataIdentitas.UsernameKurir, data.Deskripsi, db)
	}

	go func(IdKurir int64, Read *gorm.DB, rds_sesi *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataUpdatedKurir models.Kurir
		if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
			ID: IdKurir,
		}).Limit(1).Take(&dataUpdatedKurir).Error; err != nil {
			fmt.Println("Gagal mendapatakan data updated kurir")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Kurir](konteks, &dataUpdatedKurir, rds_sesi); err != nil {
			fmt.Println("Gagal memperbarui session kurir data ke cache database")
		}

		kurirDataUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataUpdatedKurir).SetTableName(dataUpdatedKurir.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirDataUpdatedPublish); err != nil {
			fmt.Println("Gagal publish updated kurir data ke message broker")
		}

	}(data.DataIdentitas.IdKurir, db.Read, rds_session, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_profiling_kurir.ResponseProfilingGeneralKurir{
			UpdateDeskripsi: hasil_update_deskripsi,
		},
	}

}
