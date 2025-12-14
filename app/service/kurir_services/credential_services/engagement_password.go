package kurir_credential_services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/emailservices"
	response_credential_kurir "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/credential_services/response_credential_services"
)

func PreUbahPasswordKurir(ctx context.Context, data PayloadPreUbahPassword, db *config.InternalDBReadWriteSystem, rds *redis.Client) *response.ResponseForm {
	services := "PreUbahPasswordKurir"

	kurir, status := data.DataIdentitas.Validating(ctx, db.Read)

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

func ValidateUbahPasswordKurir(ctx context.Context, data PayloadValidateUbahPassword, db *config.InternalDBReadWriteSystem, rds *redis.Client) *response.ResponseForm {
	services := "ValidateUbahPasswordKurir"

	_, status := data.DataIdentitas.Validating(ctx, db.Read)

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

	if err_change_pass := db.Write.WithContext(ctx).Model(models.Kurir{}).Where(models.Kurir{ID: data.DataIdentitas.IdKurir}).Update("password_hash", string(result["password_baru"])).Error; err_change_pass != nil {
		log.Printf("[ERROR] Gagal mengubah password kurir ID %d: %v", data.DataIdentitas.IdKurir, err_change_pass)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_credential_kurir.ResponseValidateUbahPassword{
				Message: "Terjadi kesalahan pada server saat mengubah password.",
			},
		}
	}

	log.Printf("[INFO] Password kurir ID %d berhasil diubah via OTP.", data.DataIdentitas.IdKurir)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_credential_kurir.ResponseValidateUbahPassword{
			Message: "Password berhasil diubah.",
		},
	}
}
