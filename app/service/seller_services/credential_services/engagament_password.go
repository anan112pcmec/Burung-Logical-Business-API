package seller_credential_services

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

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Pre Ubah Password Seller
// Berfungsi untuk mengirim kode otp ke gmail nantinya sebelum password benar benar diubah
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PreUbahPasswordSeller(ctx context.Context, data PayloadPreUbahPasswordSeller, db *config.InternalDBReadWriteSystem, rds_auth *redis.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "PreUbahPasswordSeller"

	seller, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(seller.Password), []byte(data.PasswordLama)); err != nil {
		log.Println("[WARN] Password lama yang dimasukkan salah.")
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Password lama yang dimasukkan salah.",
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.PasswordBaru), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Gagal mengenkripsi password baru: %v", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat mengenkripsi password.",
		}
	}

	otp := helper.GenerateOTP()
	key := fmt.Sprintf("seller_ubah_password_by_otp:%s", otp)

	if seller.Email == "" {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data tidak valid coba hubungi cs",
		}
	}

	to := []string{seller.Email}
	subject := "Kode Mengubah Password Burung"
	message := fmt.Sprintf("Kode Anda: %s\nMasa berlaku 3 menit.", otp)

	if err := emailservices.SendMail(to, nil, subject, message); err != nil {
		log.Printf("[ERROR] Gagal mengirim email OTP ke %s: %v", seller.Email, err)
	} else {
		log.Printf("[INFO] Email OTP berhasil dikirim ke %s", seller.Email)
	}

	fields := map[string]interface{}{
		"id_seller":     data.IdentitasSeller.IdSeller,
		"username":      data.IdentitasSeller.Username,
		"password_baru": string(hashedPassword),
	}

	pipe := rds_auth.TxPipeline()
	hset := pipe.HSet(ctx, key, fields)
	exp := pipe.Expire(ctx, key, 3*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[ERROR] Redis pipeline gagal: %v", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal menyimpan data OTP. Coba lagi nanti.",
		}
	}

	_ = hset
	_ = exp

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kode OTP telah dikirim ke email Anda. Silakan cek email untuk melanjutkan proses ubah password.",
	}
}

func ValidateUbahPasswordSeller(data PayloadValidateUbahPasswordSellerOTP, db *config.InternalDBReadWriteSystem, rds *redis.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ValidateUbahPasswordSeller"

	if data.OtpKeyValidateSeller == "" {
		log.Println("[WARN] OTP tidak ditemukan pada permintaan validasi OTP.")
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "OTP tidak ditemukan.",
		}
	}

	ctx := context.Background()
	key := fmt.Sprintf("seller_ubah_password_by_otp:%s", data.OtpKeyValidateSeller)

	result, err_rds := rds.HGetAll(ctx, key).Result()

	if err_rds != nil || len(result) == 0 {
		log.Printf("[WARN] OTP tidak valid atau sudah kadaluarsa: %v", err_rds)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "OTP tidak valid atau sudah kadaluarsa.",
		}
	}

	if errDel := rds.Del(ctx, key).Err(); errDel != nil {
		log.Printf("[WARN] Gagal menghapus OTP key dari Redis: %v", errDel)
	} else {
		log.Printf("[INFO] OTP key %s berhasil dihapus dari Redis.", key)
	}

	if err_change_pass := db.Write.WithContext(ctx).Model(&models.Seller{}).Where(&models.Seller{
		ID: data.IdentitasSeller.IdSeller,
	}).Update("password_hash", string(result["password_baru"])).Error; err_change_pass != nil {
		log.Printf("[ERROR] Gagal mengubah password seller ID %d: %v", data.IdentitasSeller.IdSeller, err_change_pass)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada server saat mengubah password.",
		}
	}

	go func(IdSeller int32, Read *gorm.DB, RdsSession *redis.Client, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataSellerUpdated models.Seller
		if err := Read.WithContext(konteks).Model(&models.Seller{}).Where(&models.Seller{
			ID: IdSeller,
		}).Limit(1).Take(&dataSellerUpdated).Error; err != nil {
			fmt.Println("Gagal mengambil data seller")
			return
		}

		if err := cache_db_entity_sessioning_seeders.UpdateCacheSessionData[*models.Seller](konteks, &dataSellerUpdated, RdsSession); err != nil {
			fmt.Println("Gagal update data cache session")
		}

		sellerUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataSellerUpdated).SetTableName(dataSellerUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, sellerUpdatedPublish); err != nil {
			fmt.Println("Gagal publish update seller ke message broker")
		}
	}(data.IdentitasSeller.IdSeller, db.Read, rds_session, cud_publisher)

	log.Printf("[INFO] Password seller ID %d berhasil diubah via OTP.", data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Password berhasil diubah.",
	}
}
