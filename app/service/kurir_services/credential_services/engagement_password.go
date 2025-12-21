package kurir_credential_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	cache_db_entity_sessioning_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/cache_database/entity_sessioning/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/emailservices"
	response_credential_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/credential_services/response_credential_services"
)

func PreUbahPasswordKurir(ctx context.Context, data PayloadPreUbahPassword, db *config.InternalDBReadWriteSystem, rds *redis.Client, rds_session *redis.Client) *response.ResponseForm {
	services := "PreUbahPasswordKurir"

	kurir, status := data.DataIdentitas.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Identitas kurir tidak valid untuk ID %d", data.DataIdentitas.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_credential_kurir.ResponsePreUbahPassword{
				Message: "Gagal, identitas kurir tidak valid.",
			},
		}
	}

	if data.PasswordBaru == "" || data.PasswordLama == "" {
		log.Println("[WARN] Password lama atau baru kosong pada permintaan ubah password kurir.")
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Payload: response_credential_kurir.ResponsePreUbahPassword{
				Message: "Gagal, isi password lama dan baru dengan benar.",
			},
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(kurir.PasswordHash), []byte(data.PasswordLama)); err != nil {
		log.Printf("[WARN] Password lama salah untuk kurir ID %d", data.DataIdentitas.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_credential_kurir.ResponsePreUbahPassword{
				Message: "Gagal, password lama yang dimasukkan salah.",
			},
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.PasswordBaru), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Gagal mengenkripsi password baru untuk kurir ID %d: %v", data.DataIdentitas.IdKurir, err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_credential_kurir.ResponsePreUbahPassword{
				Message: "Gagal, server sedang sibuk. Coba lagi lain waktu.",
			},
		}
	}

	go func() {
		otp := helper.GenerateOTP()
		key := fmt.Sprintf("kurir_ubah_password_by_otp:%s", otp)

		to := []string{data.DataIdentitas.EmailKurir}
		subject := "Kode Mengubah Password Burung"
		message := fmt.Sprintf("Kode Anda: %s\nMasa berlaku 3 menit.", otp)

		if err := emailservices.SendMail(to, nil, subject, message); err != nil {
			log.Printf("[ERROR] Gagal mengirim email OTP ke %s: %v", data.DataIdentitas.EmailKurir, err)
		} else {
			log.Printf("[INFO] Email OTP berhasil dikirim ke %s", data.DataIdentitas.EmailKurir)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		fields := map[string]interface{}{
			"id_kurir":      data.DataIdentitas.IdKurir,
			"username":      data.DataIdentitas.UsernameKurir,
			"password_baru": string(hashedPassword),
		}

		pipe := rds.TxPipeline()
		hset := pipe.HSet(ctx, key, fields)
		exp := pipe.Expire(ctx, key, 3*time.Minute)

		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("[ERROR] Redis pipeline gagal: %v", err)
		}

		_ = hset
		_ = exp
	}()

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_credential_kurir.ResponsePreUbahPassword{
			Message: fmt.Sprintf("Berhasil, silakan input kode OTP yang dikirim ke email Anda: %s", data.DataIdentitas.EmailKurir),
		},
	}
}

func ValidateUbahPasswordKurir(ctx context.Context, data PayloadValidateUbahPassword, db *config.InternalDBReadWriteSystem, rds *redis.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ValidateUbahPasswordKurir"

	_, status := data.DataIdentitas.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Identitas kurir tidak valid untuk ID %d", data.DataIdentitas.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_credential_kurir.ResponseValidateUbahPassword{
				Message: "Gagal, identitas kurir tidak valid.",
			},
		}
	}

	key := fmt.Sprintf("kurir_ubah_password_by_otp:%s", data.OtpKey)

	result, err_rds := rds.HGetAll(ctx, key).Result()

	if err_rds != nil || len(result) == 0 {
		log.Printf("[WARN] OTP tidak valid atau sudah kadaluarsa untuk kurir ID %d", data.DataIdentitas.IdKurir)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Payload: response_credential_kurir.ResponseValidateUbahPassword{
				Message: "OTP tidak valid atau sudah kadaluarsa.",
			},
		}
	}

	if errDel := rds.Del(ctx, key).Err(); errDel != nil {
		log.Printf("[WARN] Gagal menghapus OTP key dari Redis: %v", errDel)
	} else {
		log.Printf("[INFO] OTP key %s berhasil dihapus dari Redis.", key)
	}

	if err_change_pass := db.Write.WithContext(ctx).Model(&models.Kurir{}).Where(&models.Kurir{
		ID: data.DataIdentitas.IdKurir,
	}).Update("password_hash", string(result["password_baru"])).Error; err_change_pass != nil {
		log.Printf("[ERROR] Gagal mengubah password kurir ID %d: %v", data.DataIdentitas.IdKurir, err_change_pass)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_credential_kurir.ResponseValidateUbahPassword{
				Message: "Terjadi kesalahan pada server saat mengubah password.",
			},
		}
	}

	go func(IdKurir int64, Read *gorm.DB, RdsSession *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKurirUpdated models.Kurir
		if err := Read.WithContext(konteks).Model(&models.Kurir{}).Where(&models.Kurir{
			ID: IdKurir,
		}).Limit(1).Take(&dataKurirUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data kurir")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Kurir](konteks, &dataKurirUpdated, RdsSession); err != nil {
			fmt.Println("Gagal update data cache session")
		}

		kurirUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKurirUpdated).SetTableName(dataKurirUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kurirUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update kurir ke message broker")
		}
	}(data.DataIdentitas.IdKurir, db.Read, rds_session, cud_publisher)

	log.Printf("[INFO] Password kurir ID %d berhasil diubah via OTP.", data.DataIdentitas.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_credential_kurir.ResponseValidateUbahPassword{
			Message: "Password berhasil diubah.",
		},
	}
}
