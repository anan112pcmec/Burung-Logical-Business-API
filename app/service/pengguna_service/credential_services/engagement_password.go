package pengguna_credential_services

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
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Engagement Credential Level Uncritical
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur PreUbahPasswordPengguna
// :Berfungsi untuk mengirimankan otp atau prsyarat sebelum perubahan di password terjadi supaya tidak ada hal
// yang tidak diinginkan dan penyalahgunaan pihak lain pada sebuah akun
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PreUbahPasswordPengguna(ctx context.Context, data PayloadPreUbahPasswordPengguna, db *config.InternalDBReadWriteSystem, rds_auth *redis.Client, rds_session *redis.Client) *response.ResponseForm {
	services := "PreUbahPasswordPengguna"

	if data.FaktorKedua != "OTP" && data.FaktorKedua != "PIN" {
		log.Printf("[WARN] Faktor kedua tidak valid: %s", data.FaktorKedua)
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Gagal faktor kedua tidak valid",
		}
	}

	user, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data pengguna tidak ditemukan",
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(data.PasswordSebelum)); err != nil {
		log.Println("[WARN] Password lama tidak sesuai.")
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal Password lama tidak sesuai",
		}
	} else {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.PasswordBaru), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[ERROR] Gagal mengenkripsi password baru: %v", err)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal terjadi kesalahan pada server coba lagi nanti",
			}
		}

		if data.FaktorKedua == "OTP" {

			otp := helper.GenerateOTP()
			key := fmt.Sprintf("user_ubah_password_by_otp:%s", otp)

			to := []string{user.Email}
			subject := "Kode Mengubah Password Burung"
			message := fmt.Sprintf("Kode Anda: %s\nMasa berlaku 3 menit.", otp)

			if err := emailservices.SendMail(to, nil, subject, message); err != nil {
				log.Printf("[ERROR] Gagal mengirim email OTP: %v", err)
			} else {
				log.Println("[INFO] Email OTP berhasil dikirim.")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			fields := map[string]interface{}{
				"id_user":       user.ID,
				"username":      user.Username,
				"password_baru": string(hashedPassword),
			}

			pipe := rds_auth.TxPipeline()
			hset := pipe.HSet(ctx, key, fields)
			exp := pipe.Expire(ctx, key, 3*time.Minute)

			_, err := pipe.Exec(ctx)
			if err != nil {
				log.Printf("[ERROR] Gagal menyimpan OTP ke Redis: %v", err)
			}

			_ = hset
			_ = exp
		} else {
			key := fmt.Sprintf("user_ubah_password_by_pin:%v", user.ID)

			to := []string{user.Email}
			subject := "Kode Mengubah Password Burung"
			message := fmt.Sprintf("Anda mengubah password akun Burung pada %s menggunakan faktor PIN.", time.Now().Format("02-01-2006 15:04:05"))

			if err := emailservices.SendMail(to, nil, subject, message); err != nil {
				log.Printf("[ERROR] Gagal mengirim notifikasi email PIN: %v", err)
			} else {
				log.Println("[INFO] Email notifikasi PIN berhasil dikirim.")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			fields := map[string]interface{}{
				"id_user":       user.ID,
				"username":      user.Username,
				"password_baru": string(hashedPassword),
			}

			if _, err := rds_auth.Del(ctx, key).Result(); err != nil {
				fmt.Println("gagal hapus", err)
			}

			pipe := rds_auth.TxPipeline()
			hset := pipe.HSet(ctx, key, fields)
			exp := pipe.Expire(ctx, key, 3*time.Minute)

			_, err := pipe.Exec(ctx)
			if err != nil {
				log.Printf("[ERROR] Gagal menyimpan data PIN ke Redis: %v", err)
			}

			_ = hset
			_ = exp
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  fmt.Sprintf("Berhasil, silakan masukkan kredensial berikutnya: %s.", data.FaktorKedua),
	}
}

func ValidateUbahPasswordPenggunaViaOtp(ctx context.Context, data PayloadValidateOTPPasswordPengguna, db *config.InternalDBReadWriteSystem, rds_auth, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ValidateUbahPasswordPenggunaViaOtp"

	var id_user int64
	if check_user := db.Read.Model(models.Pengguna{}).Select("id").Where(models.Pengguna{ID: data.IDPengguna}).First(&id_user).Error; check_user != nil {
		log.Printf("[WARN] Pengguna tidak ditemukan untuk validasi OTP: %v", check_user)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Pengguna tidak ditemukan.",
		}
	}

	key := fmt.Sprintf("user_ubah_password_by_otp:%s", data.OtpKey)

	result, err_rds := rds_auth.HGetAll(ctx, key).Result()

	if err_rds != nil || len(result) == 0 {
		log.Printf("[WARN] OTP tidak valid atau sudah kadaluarsa: %v", err_rds)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "OTP tidak valid atau sudah kadaluarsa.",
		}
	}

	if err_change_pass := db.Write.Model(models.Pengguna{}).Where(models.Pengguna{ID: data.IDPengguna}).Update("password_hash", string(result["password_baru"])).Error; err_change_pass != nil {
		log.Printf("[ERROR] Gagal mengubah password via OTP: %v", err_change_pass)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat mengubah password.",
		}
	}

	if _, err_del := rds_auth.Del(ctx, key).Result(); err_del != nil {
		log.Printf("[WARN] Gagal menghapus OTP key: %v", err_del)
	}

	go func(Ip int64, Read *gorm.DB, rds_session *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataPengguna models.Pengguna
		if err := Read.WithContext(konteks).Model(&models.Pengguna{}).Where(&models.Pengguna{
			ID: Ip,
		}).Limit(1).Take(&dataPengguna).Error; err != nil {
			fmt.Println("Gagal mengambil data pengguna")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Pengguna](konteks, &dataPengguna, rds_session); err != nil {
			fmt.Println("Gagal update data cache session")
		}

		penggunaUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengguna).SetTableName(dataPengguna.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, penggunaUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update pengguna password via otp ke message broker")
		}

	}(data.IDPengguna, db.Read, rds_session, cud_publisher)

	log.Println("[INFO] Password berhasil diubah via OTP.")
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Password berhasil diubah.",
	}

}

func ValidateUbahPasswordPenggunaViaPin(ctx context.Context, data PayloadValidatePinPasswordPengguna, db *config.InternalDBReadWriteSystem, rds_auth, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ValidateUbahPasswordPenggunaViaPin"

	var pin_user string
	if check_pin := db.Read.WithContext(ctx).Model(models.Pengguna{}).Select("pin_hash").Where(models.Pengguna{ID: data.IDPengguna}).Limit(1).Scan(&pin_user).Error; check_pin != nil {
		log.Printf("[WARN] PIN pengguna tidak ditemukan: %v", check_pin)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "PIN pengguna tidak ditemukan.",
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pin_user), []byte(data.Pin)); err != nil {
		log.Println("[WARN] PIN yang dimasukkan salah.")
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "PIN yang dimasukkan salah.",
		}
	} else {
		key := fmt.Sprintf("user_ubah_password_by_pin:%v", data.IDPengguna)
		result, err_validate := rds_auth.HGetAll(ctx, key).Result()
		if err_validate != nil || len(result) == 0 {
			log.Printf("[WARN] Data perubahan password via PIN tidak ditemukan atau sudah kadaluarsa: %v", err_validate)
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Data perubahan password via PIN tidak ditemukan atau sudah kadaluarsa.",
			}
		}

		if _, err_del := rds_auth.Del(ctx, key).Result(); err_del != nil {
			log.Printf("[WARN] Gagal menghapus OTP key: %v", err_del)
		}

		if err_change_pass := db.Write.WithContext(ctx).Model(models.Pengguna{}).Where(models.Pengguna{ID: data.IDPengguna}).Update("password_hash", string(result["password_baru"])).Error; err_change_pass != nil {
			log.Printf("[ERROR] Gagal mengubah password via PIN: %v", err_change_pass)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Terjadi kesalahan pada server saat mengubah password.",
			}
		}
	}

	go func(Ip int64, Read *gorm.DB, rds_session *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataPengguna models.Pengguna
		if err := Read.WithContext(konteks).Model(&models.Pengguna{}).Where(&models.Pengguna{
			ID: Ip,
		}).Limit(1).Take(&dataPengguna).Error; err != nil {
			fmt.Println("Gagal mengambil data pengguna")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Pengguna](konteks, &dataPengguna, rds_session); err != nil {
			fmt.Println("Gagal update data cache session")
		}

		penggunaUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengguna).SetTableName(dataPengguna.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, penggunaUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update pengguna password via pin ke message broker")
		}

	}(data.IDPengguna, db.Read, rds_session, cud_publisher)

	log.Println("[INFO] Password berhasil diubah via PIN.")
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Password berhasil diubah.",
	}
}

func MembuatSecretPinPengguna(ctx context.Context, data PayloadMembuatPinPengguna, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MembuatSecretPinPengguna"

	user, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session)

	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Data kredensial tidak valid.",
		}
	}

	if check_pass := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(data.Password)); check_pass != nil {
		log.Println("[WARN] Password yang dimasukkan salah.")
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Password yang dimasukkan salah.",
		}
	}

	hashed_pin, err := bcrypt.GenerateFromPassword([]byte(data.Pin), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Gagal mengenkripsi PIN: %v", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat membuat PIN.",
		}
	}

	if errUpdatePin := db.Write.WithContext(ctx).Model(models.Pengguna{}).
		Where(models.Pengguna{ID: user.ID}).
		Update("pin_hash", string(hashed_pin)).Error; errUpdatePin != nil {
		log.Printf("[ERROR] Gagal menyimpan PIN ke database: %v", errUpdatePin)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat menyimpan PIN.",
		}
	}

	go func(Ip int64, Read *gorm.DB, rds_session *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataPengguna models.Pengguna
		if err := Read.WithContext(konteks).Model(&models.Pengguna{}).Where(&models.Pengguna{
			ID: Ip,
		}).Limit(1).Take(&dataPengguna).Error; err != nil {
			fmt.Println("Gagal mengambil data pengguna")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Pengguna](konteks, &dataPengguna, rds_session); err != nil {
			fmt.Println("Gagal update data cache session")
		}

		penggunaUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengguna).SetTableName(dataPengguna.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, penggunaUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update pengguna membuat pin ke message broker")
		}

	}(data.IdentitasPengguna.ID, db.Read, rds_session, cud_publisher)

	log.Println("[INFO] PIN berhasil dibuat.")
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "PIN berhasil dibuat.",
	}
}

func UpdateSecretPinPengguna(ctx context.Context, data PayloadUpdatePinPengguna, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "UpdateSecretPinPengguna"

	user, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session)

	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Data kredensial tidak valid.",
		}
	}

	if check_pin := bcrypt.CompareHashAndPassword([]byte(user.PinHash), []byte(data.PinLama)); check_pin != nil {
		log.Println("[WARN] PIN lama yang dimasukkan salah.")
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "PIN lama yang dimasukkan salah.",
		}
	}

	hashed_pin, err := bcrypt.GenerateFromPassword([]byte(data.PinBaru), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Gagal mengenkripsi PIN baru: %v", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat membuat PIN baru.",
		}
	}

	if errUpdatePin := db.Write.WithContext(ctx).Model(models.Pengguna{}).
		Where(models.Pengguna{ID: user.ID}).
		Update("pin_hash", string(hashed_pin)).Error; errUpdatePin != nil {
		log.Printf("[ERROR] Gagal menyimpan PIN baru ke database: %v", errUpdatePin)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat menyimpan PIN baru.",
		}
	}

	go func(Ip int64, Read *gorm.DB, rds_session *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataPengguna models.Pengguna
		if err := Read.WithContext(konteks).Model(&models.Pengguna{}).Where(&models.Pengguna{
			ID: Ip,
		}).Limit(1).Take(&dataPengguna).Error; err != nil {
			fmt.Println("Gagal mengambil data pengguna")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Pengguna](konteks, &dataPengguna, rds_session); err != nil {
			fmt.Println("Gagal update data cache session")
		}

		penggunaUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPengguna).SetTableName(dataPengguna.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, penggunaUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update pengguna pin ke message broker")
		}

	}(data.IdentitasPengguna.ID, db.Read, rds_session, cud_publisher)

	log.Println("[INFO] PIN berhasil diubah.")
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "PIN berhasil diubah.",
	}
}
