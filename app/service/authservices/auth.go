package authservices

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_auth "github.com/anan112pcmec/Burung-backend-1/app/service/authservices/reponse_auth"
	"github.com/anan112pcmec/Burung-backend-1/app/service/emailservices"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Login Entity Function Procudure
// :Bertujuan Untuk menangani aksi Login Dari Pengguna atau seller atau kurir,
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UserLogin(db *config.InternalDBReadWriteSystem, email, password string) *response.ResponseForm {
	service := "UserLogin"
	var user models.Pengguna

	if err := db.Read.Where(models.Pengguna{Email: email}).Select("id", "nama", "username", "email", "password_hash", "status").Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &response.ResponseForm{
				Status:   http.StatusNotFound,
				Services: service,
				Payload: response_auth.LoginUserResp{
					Message: "Gagal Akun Belum Terdaftar, Coba Daftar kan dirimu dan bergabung bersama kami",
				},
			}
		}

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: service,
			Payload: response_auth.LoginUserResp{
				Message: "Coba Login Nanti Lagi Server sedang sibuk",
			},
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: service,
			Payload: response_auth.LoginUserResp{
				Message: "Password Salah",
			},
		}
	} else {
		go func() {
			if user.StatusPengguna == "Offline" {
				if err1 := db.Write.Model(models.Pengguna{}).Where(models.Pengguna{Email: email}).Update("status", "Online").Error; err1 != nil {
					fmt.Println("Gagal Ubah Status")
				}
			} else {
				fmt.Println("user sudah login di tempat lain")
			}
		}()
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: service,
		Payload: response_auth.LoginUserResp{
			Status:  "Berhasil",
			Message: "Kamu Berhasil Login Selamat datang",
			LoginResponse: response_auth.LoginResponse{
				ID:       user.ID,
				Nama:     user.Nama,
				Username: user.Username,
			},
		},
	}
}

func SellerLogin(db *config.InternalDBReadWriteSystem, email, password string) *response.ResponseForm {
	service := "SellerLogin"
	var seller models.Seller

	if err := db.Read.Where(&models.Seller{Email: email}).
		Select("id", "nama", "username", "email", "password_hash", "status").
		First(&seller).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &response.ResponseForm{
				Status:   http.StatusNotFound,
				Services: service,
				Payload: response_auth.LoginSellerResp{
					Message: "Gagal, akun belum terdaftar. Silakan daftar dulu.",
				},
			}
		}

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: service,
			Payload: response_auth.LoginSellerResp{
				Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
			},
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(seller.Password), []byte(password)); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: service,
			Payload: response_auth.LoginSellerResp{
				Message: "Password salah",
			},
		}
	}

	go func() {
		if seller.StatusSeller == "Offline" {
			if err := db.Write.Model(&models.Seller{}).
				Where(&models.Seller{Email: email}).
				Update("status", "Online").Error; err != nil {
				fmt.Println("Gagal update status seller:", err)
			} else {
				fmt.Println("Seller sudah login di tempat lain")
			}
		}
	}()

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: service,
		Payload: response_auth.LoginSellerResp{
			Status:  "Berhasil",
			Message: fmt.Sprintf("Kamu berhasil login %s, kembangkan koneksimu dan raih keuntungan di sini!", seller.Nama),
			LoginResponse: response_auth.LoginResponse{
				ID:       int64(seller.ID),
				Nama:     seller.Nama,
				Username: seller.Username,
			},
		},
	}
}

func KurirLogin(db *config.InternalDBReadWriteSystem, email, password string) *response.ResponseForm {
	service := "KurirLogin"

	var kurir models.Kurir
	if err := db.Read.Where(&models.Kurir{Email: email}).
		Select("id", "nama", "email", "password_hash", "status").
		First(&kurir).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &response.ResponseForm{
				Status:   http.StatusNotFound,
				Services: service,
				Payload: response_auth.LoginKurirResp{
					Message: "Gagal, akun belum terdaftar. Silakan daftar dulu.",
				},
			}
		}

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: service,
			Payload: response_auth.LoginKurirResp{
				Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
			},
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(kurir.PasswordHash), []byte(password)); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: service,
			Payload: response_auth.LoginKurirResp{
				Message: "Password salah",
			},
		}
	}

	go func() {
		if kurir.StatusKurir == "Offline" {
			if err := db.Write.Model(&models.Kurir{}).
				Where(&models.Kurir{Email: email}).
				Update("status", "Online").Error; err != nil {
				fmt.Println("Gagal update status kurir:", err)
			} else {
				fmt.Println("Seller sudah login di tempat lain")
			}
		}
	}()

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: service,
		Payload: response_auth.LoginKurirResp{
			Status:  "Berhasil",
			Message: fmt.Sprintf("Kamu berhasil login %s, kembangkan koneksimu dan raih keuntungan di sini!", kurir.Nama),
			LoginResponse: response_auth.LoginResponse{
				ID:   kurir.ID,
				Nama: kurir.Nama,
			},
		},
	}

}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// PreRegistration Function Procedure
// :Bertujuan untuk melakukan Aksi pre registrasi untuk user seller dan kurir
// :Manfaatnya tidak akan banyak akun spam, semua akun yang terintegrasi valid dengan gmail nya dan identitas lain
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PreUserRegistration(db *config.InternalDBReadWriteSystem, username, nama, email, password string, rds *redis.Client) *response.ResponseForm {
	services := "PreUserRegistration"
	ctx := context.Background()

	var user int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Pengguna{}).Select("id").Where(&models.Pengguna{Email: email}).Or(&models.Pengguna{Username: username}).Limit(1).Scan(&user).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if user != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal pengguna dengan username atau email tersebut sudah ada",
		}
	}

	Otp := helper.GenerateOTP()

	go func() {
		to := []string{email}
		cc := []string{}
		subject := "Kode OTP App Burung"
		message := fmt.Sprintf("Kode OTP Anda: %s\nMasa berlaku 3 menit.", Otp)

		err := emailservices.SendMail(to, cc, subject, message)

		if err != nil {
			fmt.Println("Gagal Kirim OTP")
		}
	}()

	go func() {
		key := fmt.Sprintf("registration_user_pending:%s", Otp)

		fields := map[string]interface{}{
			"nama":          nama,
			"username":      username,
			"email":         email,
			"password_hash": password,
		}

		for data, name := range fields {
			if err := rds.HSet(ctx, key, data, name).Err(); err != nil {
				fmt.Println("Gagal Set Redis")
			}
		}

		if err := rds.Expire(ctx, key, 3*time.Minute).Err(); err != nil {
			fmt.Println("Gagal set expired redis")
		}
	}()

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.PreRegistrationUserResp{
			Status:  "Berhasil",
			Message: "Silahkan Masukan Kode OTP yang sudah di kirimkan ke Gmail Anda",
		},
	}
}

func PreSellerRegistration(db *config.InternalDBReadWriteSystem, username, nama, email string, jenis string, SellerDedication string, password string, rds *redis.Client) *response.ResponseForm {
	services := "PreSellerRegistration"

	var seller int64 = 0
	if err := db.Read.Model(&models.Seller{}).Select("id").
		Where(&models.Seller{Email: email}).
		Or(&models.Seller{Username: username}).
		Limit(1).Scan(&seller).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if seller != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal username atau gmail tersebut sudah digunakan",
		}
	}

	Otp := helper.GenerateOTP()

	go func() {
		to := []string{email}
		subject := "Kode OTP App Burung"
		message := fmt.Sprintf("Kode OTP Anda: %s\nMasa berlaku 3 menit.", Otp)

		if err := emailservices.SendMail(to, nil, subject, message); err != nil {
			fmt.Println("Gagal Kirim Email Untuk Otp:", Otp)
		}

		log.Println("[TRACE] Email sent successfully")
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		key := fmt.Sprintf("registration_seller_pending:%s", Otp)
		fields := map[string]interface{}{
			"nama":              nama,
			"username":          username,
			"email":             email,
			"jenis":             jenis,
			"seller_dedication": SellerDedication,
			"password_hash":     password,
		}

		pipe := rds.TxPipeline()
		hset := pipe.HSet(ctx, key, fields)
		exp := pipe.Expire(ctx, key, 3*time.Minute)

		res, err := pipe.Exec(ctx)
		if err != nil {
			log.Printf("[ERROR] Redis pipeline failed: %v\n", err)

		}

		_ = hset
		_ = exp
		_ = res
	}()

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.PreRegistrationSellerResp{
			Status:  "Berhasil",
			Message: "Silahkan Masukan Kode OTP yang sudah dikirimkan ke Gmail Anda",
		},
	}
}

func PreKurirRegistration(db *config.InternalDBReadWriteSystem, nama, email, password, username string, rds *redis.Client) *response.ResponseForm {
	services := "PreKurirRegistration"

	var kurir int64 = 0
	if err := db.Read.Model(&models.Kurir{}).Select("id").Where(&models.Kurir{Email: email}).Or(&models.Kurir{
		Username: username,
	}).Limit(1).Scan(&kurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	Otp := helper.GenerateOTP()

	go func() {
		to := []string{email}
		subject := "Kode OTP App Burung"
		message := fmt.Sprintf("Kode OTP Anda: %s\nMasa berlaku 3 menit.", Otp)

		if err := emailservices.SendMail(to, nil, subject, message); err != nil {
			fmt.Println("Gagal Kirim Email Untuk Otp:", Otp)
		}

		log.Println("[TRACE] Email sent successfully")
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		key := fmt.Sprintf("registration_kurir_pending:%s", Otp)
		fields := map[string]interface{}{
			"nama":          nama,
			"email":         email,
			"username":      username,
			"password_hash": password,
		}

		pipe := rds.TxPipeline()
		hset := pipe.HSet(ctx, key, fields)
		exp := pipe.Expire(ctx, key, 3*time.Minute)

		res, err := pipe.Exec(ctx)
		if err != nil {
			log.Printf("[ERROR] Redis pipeline failed: %v\n", err)

		}

		_ = hset
		_ = exp
		_ = res
	}()

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.PreRegistrationKurirResp{
			Status:  "Berhasil",
			Message: "Silahkan Masukan Kode OTP yang sudah dikirimkan ke Gmail Anda",
		},
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ValidateRegistration
// :Bertujuan Untuk menangani aksi Validasi Preregister tadi,
// :Bermanfaat dalam memvalidasi sebuah pengguna (bukan orang iseng/bot/dll) supaya bisa dipertanggung jawabkan
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////

func ValidateUserRegistration(db *config.InternalDBReadWriteSystem, OTPkey string, rds *redis.Client) *response.ResponseForm {
	services := "ValidateUserRegistration"

	ctx := context.Background()

	key := fmt.Sprintf("registration_user_pending:%s", OTPkey)

	userData, err := rds.HGetAll(ctx, key).Result()
	if err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR getting data from Redis:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateUserResp{
				Message: "Gagal Kode Sudah Expired, Coba Registrasi Ulang",
			},
		}
	}

	if len(userData) == 0 {
		fmt.Println("[ValidateUserRegistration] Key not found or expired")
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_auth.ValidateUserResp{
				Message: "Gagal Kode Sudah Expired, Coba Registrasi Ulang",
			},
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData["password_hash"]), bcrypt.DefaultCost)

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateUserResp{
				Message: "Gagal Server Sedang Sibuk, Coba Registrasi Ulang",
			},
		}
	}

	user := models.Pengguna{
		Nama:         userData["nama"],
		Username:     userData["username"],
		Email:        userData["email"],
		PasswordHash: string(hashedPassword),
	}

	if err := db.Write.Create(&user).Error; err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR saving to DB:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateUserResp{
				Message: "Gagal Server Sedang Sibuk, Coba Registrasi Ulang",
			},
		}
	}

	if err := rds.Del(ctx, key).Err(); err != nil {
		fmt.Println("[ValidateUserRegistration] WARNING deleting Redis key:", err)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.ValidateUserResp{
			Status:  "Berhasil",
			Message: "Berhasil, Sekarang Kamu sudah memiliki akun di sistem Burung dan menjadi Bagian Dari Kami",
		},
	}
}

func ValidateSellerRegistration(db *config.InternalDBReadWriteSystem, OTPkey string, rds *redis.Client) *response.ResponseForm {
	services := "ValidateSellerRegistration"
	ctx := context.Background()

	key := fmt.Sprintf("registration_seller_pending:%s", OTPkey)

	userData, err := rds.HGetAll(ctx, key).Result()
	if err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR getting data from Redis:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateSellerResp{
				Message: "Gagal Kode Sudah Expired, Coba Registrasi Ulang",
			},
		}
	}

	if len(userData) == 0 {
		fmt.Println("[ValidateUserRegistration] Key not found or expired")
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_auth.ValidateSellerResp{
				Message: "Gagal Kode Sudah Expired, Coba Registrasi Ulang",
			},
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData["password_hash"]), bcrypt.DefaultCost)

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateSellerResp{
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti",
			},
		}
	}

	seller := models.Seller{
		Nama:             userData["nama"],
		Username:         userData["username"],
		Email:            userData["email"],
		Jenis:            "Personal",
		SellerDedication: userData["seller_dedication"],
		Password:         string(hashedPassword),
	}

	if err := db.Write.Create(&seller).Error; err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR saving to DB:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateSellerResp{
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti Ya",
			},
		}
	}

	if err := rds.Del(ctx, key).Err(); err != nil {
		fmt.Println("[ValidateUserRegistration] WARNING deleting Redis key:", err)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.ValidateSellerResp{
			Status:  "Berhasil",
			Message: "Berhasil, Akun mu sudah terdaftar dan Kamu Siap Berjualan Bersama Kami",
		},
	}
}

func ValidateKurirRegistration(db *config.InternalDBReadWriteSystem, OTPkey string, rds *redis.Client) *response.ResponseForm {

	services := "ValidateKurirRegistration"
	ctx := context.Background()

	key := fmt.Sprintf("registration_kurir_pending:%s", OTPkey)

	userData, err := rds.HGetAll(ctx, key).Result()
	if err != nil {
		fmt.Println("[ValidateKurirRegistration] ERROR getting data from Redis:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateKurirResp{
				Message: "Gagal Kode Sudah Expired, Coba Registrasi Ulang",
			},
		}
	}

	if len(userData) == 0 {
		fmt.Println("[ValidateUserRegistration] Key not found or expired")
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload: response_auth.ValidateKurirResp{
				Message: "Gagal Kode Sudah Expired, Coba Registrasi Ulang",
			},
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData["password_hash"]), bcrypt.DefaultCost)

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateSellerResp{
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti",
			},
		}
	}

	seller := models.Kurir{
		Nama:         userData["nama"],
		Email:        userData["email"],
		Username:     userData["username"],
		PasswordHash: string(hashedPassword),
	}

	if err := db.Write.Create(&seller).Error; err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR saving to DB:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateKurirResp{
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti Ya",
			},
		}
	}

	if err := rds.Del(ctx, key).Err(); err != nil {
		fmt.Println("[ValidateUserRegistration] WARNING deleting Redis key:", err)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.ValidateKurirResp{
			Status:  "Berhasil",
			Message: "Berhasil, Akun mu sudah terdaftar dan Kamu Siap Menjadi Bagian Dari Kami",
		},
	}
}
