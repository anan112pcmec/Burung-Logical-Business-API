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

	cache_db_entity_sessioning_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/cache_database/entity_sessioning/seeders"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	response_auth "github.com/anan112pcmec/Burung-backend-1/app/service/authservices/reponse_auth"
	"github.com/anan112pcmec/Burung-backend-1/app/service/emailservices"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"

)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Login Entity Function Procudure
// :Bertujuan Untuk menangani aksi Login Dari Pengguna atau seller atau kurir,
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PenggunaLogin(ctx context.Context, db *environment.InternalDBReadWriteSystem, email, password string, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	service := "PenggunaLogin"
	var user sot_models.Pengguna

	if err := db.Read.WithContext(ctx).Model(&sot_models.Pengguna{}).Where(&sot_models.Pengguna{Email: email}).Limit(1).Take(&user).Error; err != nil {
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
		if user.StatusPengguna == entity_enums.Offline {
			go func(u sot_models.Pengguna, Write gorm.DB, publisher *mb_cud_publisher.Publisher) {
				if err1 := Write.Model(sot_models.Pengguna{}).Where(sot_models.Pengguna{ID: u.ID}).Update("status", entity_enums.Online).Error; err1 != nil {
					fmt.Println("Gagal Ubah Status")
					return
				} else {
					u.StatusPengguna = entity_enums.Online
				}

				ctx_t := context.Background()
				konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
				defer cancel()

				updateOnlinePengguna := mb_cud_serializer.NewJsonPayload().SetPayload(u).SetTableName(service).SetRole(mb_cud_seeders.Pengguna)
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateOnlinePengguna); err != nil {
					fmt.Println("Gagal publish user online")
				}
			}(user, *db.Write, cud_publisher)
		} else {
			fmt.Println("user sudah login di tempat lain")
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
	}

	go func(u sot_models.Pengguna, rds_pass *redis.Client) {
		ctx_t, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		key := cache_db_entity_sessioning_seeders.SetSessionKey(&u)
		session := helper.StructToJSONMap(u)
		pipe := rds_pass.Pipeline()
		pipe.HSet(ctx_t, key, session)

		if _, err := pipe.Exec(ctx_t); err != nil {
			log.Printf("[UserLogin][Redis] %v", err)
		} else {
			log.Printf("[UserLogin][Redis] %v", "berhasil")
		}
	}(user, rds)

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
} //Tuned

func SellerLogin(ctx context.Context, db *environment.InternalDBReadWriteSystem, email, password string, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	service := "SellerLogin"
	var seller sot_models.Seller

	if err := db.Read.WithContext(ctx).Model(&sot_models.Seller{}).Where(&sot_models.Seller{Email: email}).Limit(1).Take(&seller).Error; err != nil {

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
	} else {
		if seller.StatusSeller == entity_enums.Offline {
			go func(u sot_models.Seller, Write gorm.DB, publisher *mb_cud_publisher.Publisher) {
				if err := Write.Model(&sot_models.Seller{}).
					Where(&sot_models.Seller{ID: u.ID}).
					Update("status", entity_enums.Online).Error; err != nil {
					fmt.Println("Gagal update status seller:", err)
					return
				} else {
					u.StatusSeller = entity_enums.Online
				}
				ctx_t := context.Background()
				konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
				defer cancel()

				updateOnlineSeller := mb_cud_serializer.NewJsonPayload().SetPayload(u).SetTableName(service).SetRole(mb_cud_seeders.Seller)
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateOnlineSeller); err != nil {
					fmt.Println("Gagal publish user online")
				}
			}(seller, *db.Write, cud_publisher)
		} else {
			fmt.Println("Seller sudah online di platform lain")
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

	}

	go func(u sot_models.Seller, rds_pass *redis.Client) {
		ctx_t, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		key := cache_db_entity_sessioning_seeders.SetSessionKey(&u)
		session := helper.StructToJSONMap(u)
		pipe := rds_pass.Pipeline()
		pipe.HSet(ctx_t, key, session)

		if _, err := pipe.Exec(ctx_t); err != nil {
			log.Printf("[SellerLogin][Redis] %v", err)
		}
	}(seller, rds)

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
} //Tuned

func KurirLogin(ctx context.Context, db *environment.InternalDBReadWriteSystem, email, password string, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	service := "KurirLogin"

	var kurir sot_models.Kurir
	if err := db.Read.WithContext(ctx).Model(&sot_models.Kurir{}).Where(&sot_models.Kurir{Email: email}).Limit(1).Take(&kurir).Error; err != nil {

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
	} else {

		if kurir.StatusKurir == entity_enums.Offline {
			go func(u sot_models.Kurir, Write gorm.DB, publisher *mb_cud_publisher.Publisher) {
				if err := Write.Model(&sot_models.Kurir{}).
					Where(&sot_models.Kurir{ID: u.ID}).
					Update("status", entity_enums.Online).Error; err != nil {
					fmt.Println("Gagal update status kurir:", err)
				} else {
					u.StatusKurir = entity_enums.Online
				}

				ctx_t := context.Background()
				konteks, cancel := context.WithTimeout(ctx_t, settings.TimeoutContext)
				defer cancel()

				updateOnlineKurir := mb_cud_serializer.NewJsonPayload().SetPayload(u).SetTableName(service).SetRole(mb_cud_seeders.Kurir)
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateOnlineKurir); err != nil {
					fmt.Println("Gagal publish user online")
				}
			}(kurir, *db.Write, cud_publisher)
		} else {
			fmt.Println("Kurir sudah login di tempat lain")
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

	}

	go func(u sot_models.Kurir, rds_pass *redis.Client) {
		ctx_t, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		key := cache_db_entity_sessioning_seeders.SetSessionKey(&u)
		session := helper.StructToJSONMap(u)
		pipe := rds_pass.Pipeline()
		pipe.HSet(ctx_t, key, session)

		if _, err := pipe.Exec(ctx_t); err != nil {
			log.Printf("[KurirLogin][Redis] %v", err)
		}
	}(kurir, rds)

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
} //Tuned

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// PreRegistration Function Procedure
// :Bertujuan untuk melakukan Aksi pre registrasi untuk user seller dan kurir
// :Manfaatnya tidak akan banyak akun spam, semua akun yang terintegrasi valid dengan gmail nya dan identitas lain
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PreUserRegistration(ctx context.Context, db *environment.InternalDBReadWriteSystem, username, nama, email, password string, rds *redis.Client) *response.ResponseForm {
	services := "PreUserRegistration"

	var user int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Pengguna{}).Select("id").Where(&sot_models.Pengguna{Email: email}).Or(&sot_models.Pengguna{Username: username}).Limit(1).Scan(&user).Error; err != nil {
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

	go func(email_p, unique_otp string) {
		to := []string{email_p}
		cc := []string{}
		subject := "Kode OTP App Burung"
		message := fmt.Sprintf("Kode OTP Anda: %s\nMasa berlaku 3 menit.", unique_otp)

		err := emailservices.SendMail(to, cc, subject, message)

		if err != nil {
			fmt.Println("Gagal Kirim OTP")
		}
	}(email, Otp)

	go func(username_p, nama_p, email_p, pass_p, unique_otp string, redis_auth *redis.Client) {
		konteks, batal := context.WithTimeout(context.Background(), settings.TimeoutCacheOperation)
		defer batal()
		key := fmt.Sprintf("registration_user_pending:%s", unique_otp)

		fields := map[string]interface{}{
			"nama":          nama_p,
			"username":      username_p,
			"email":         email_p,
			"password_hash": pass_p,
		}

		for data, name := range fields {
			if err := redis_auth.HSet(konteks, key, data, name).Err(); err != nil {
				fmt.Println("Gagal Set Redis")
			}
		}

		if err := rds.Expire(konteks, key, 3*time.Minute).Err(); err != nil {
			fmt.Println("Gagal set expired redis")
		}
	}(username, nama, email, password, Otp, rds)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.PreRegistrationUserResp{
			Status:  "Berhasil",
			Message: "Silahkan Masukan Kode OTP yang sudah di kirimkan ke Gmail Anda",
		},
	}
} // Tuned

func PreSellerRegistration(ctx context.Context, db *environment.InternalDBReadWriteSystem, username, nama, email string, jenis string, SellerDedication string, password string, rds *redis.Client) *response.ResponseForm {
	services := "PreSellerRegistration"

	var seller int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Seller{}).Select("id").
		Where(&sot_models.Seller{Email: email}).
		Or(&sot_models.Seller{Username: username}).
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

	go func(email_p, unique_otp string) {
		to := []string{email_p}
		subject := "Kode OTP App Burung"
		message := fmt.Sprintf("Kode OTP Anda: %s\nMasa berlaku 3 menit.", unique_otp)

		if err := emailservices.SendMail(to, nil, subject, message); err != nil {
			fmt.Println("Gagal Kirim Email Untuk Otp:", Otp)
		}

		log.Println("[TRACE] Email sent successfully")
	}(email, Otp)

	go func(username_p, nama_p, email_p, jenis_p, seller_dedic_p, pass_p, unique_otp string, redis_auth *redis.Client) {
		konteks, batal := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer batal()
		key := fmt.Sprintf("registration_seller_pending:%s", unique_otp)
		fields := map[string]interface{}{
			"nama":              nama_p,
			"username":          username_p,
			"email":             email_p,
			"jenis":             jenis_p,
			"seller_dedication": seller_dedic_p,
			"password_hash":     pass_p,
		}

		pipe := redis_auth.TxPipeline()
		hset := pipe.HSet(konteks, key, fields)
		exp := pipe.Expire(konteks, key, 3*time.Minute)

		res, err := pipe.Exec(konteks)
		if err != nil {
			log.Printf("[ERROR] Redis pipeline failed: %v\n", err)

		}

		_ = hset
		_ = exp
		_ = res
	}(username, nama, email, jenis, SellerDedication, password, Otp, rds)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.PreRegistrationSellerResp{
			Status:  "Berhasil",
			Message: "Silahkan Masukan Kode OTP yang sudah dikirimkan ke Gmail Anda",
		},
	}
} // Tuned

func PreKurirRegistration(ctx context.Context, db *environment.InternalDBReadWriteSystem, nama, email, password, username string, rds *redis.Client) *response.ResponseForm {
	services := "PreKurirRegistration"

	var kurir int64 = 0
	if err := db.Read.Model(&sot_models.Kurir{}).WithContext(ctx).Select("id").Where(&sot_models.Kurir{Email: email}).Or(&sot_models.Kurir{
		Username: username,
	}).Limit(1).Scan(&kurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	Otp := helper.GenerateOTP()

	go func(email_p, unique_otp string) {
		to := []string{email_p}
		subject := "Kode OTP App Burung"
		message := fmt.Sprintf("Kode OTP Anda: %s\nMasa berlaku 3 menit.", unique_otp)

		if err := emailservices.SendMail(to, nil, subject, message); err != nil {
			fmt.Println("Gagal Kirim Email Untuk Otp:", Otp)
		}

		log.Println("[TRACE] Email sent successfully")
	}(email, Otp)

	go func(username_p, nama_p, email_p, pass_p, unique_otp string, redis_auth *redis.Client) {
		konteks, batal := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer batal()
		key := fmt.Sprintf("registration_kurir_pending:%s", unique_otp)
		fields := map[string]interface{}{
			"nama":          nama_p,
			"email":         email_p,
			"username":      username_p,
			"password_hash": pass_p,
		}

		pipe := redis_auth.TxPipeline()
		hset := pipe.HSet(konteks, key, fields)
		exp := pipe.Expire(konteks, key, 3*time.Minute)

		res, err := pipe.Exec(konteks)
		if err != nil {
			log.Printf("[ERROR] Redis pipeline failed: %v\n", err)

		}

		_ = hset
		_ = exp
		_ = res
	}(username, nama, email, password, Otp, rds)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.PreRegistrationKurirResp{
			Status:  "Berhasil",
			Message: "Silahkan Masukan Kode OTP yang sudah dikirimkan ke Gmail Anda",
		},
	}
} // Tuned

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ValidateRegistration
// :Bertujuan Untuk menangani aksi Validasi Preregister tadi,
// :Bermanfaat dalam memvalidasi sebuah pengguna (bukan orang iseng/bot/dll) supaya bisa dipertanggung jawabkan
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////

func ValidateUserRegistration(ctx context.Context, db *environment.InternalDBReadWriteSystem, OTPkey string, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ValidateUserRegistration"

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

	if err := rds.Del(ctx, key).Err(); err != nil {
		fmt.Println("[ValidateUserRegistration] WARNING deleting Redis key:", err)
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

	user := sot_models.Pengguna{
		Nama:         userData["nama"],
		Username:     userData["username"],
		Email:        userData["email"],
		PasswordHash: string(hashedPassword),
	}

	if err := db.Write.WithContext(ctx).Create(&user).Error; err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR saving to DB:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateUserResp{
				Message: "Gagal Server Sedang Sibuk, Coba Registrasi Ulang",
			},
		}
	} else {
		go func(u sot_models.Pengguna, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			newPenggunaCreated := mb_cud_serializer.NewJsonPayload().SetPayload(u).SetTableName(u.TableName()).SetRole(mb_cud_seeders.Pengguna)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newPenggunaCreated); err != nil {
				fmt.Println("Gagal mempublish create pengguna ber id: ", u.ID)
			}

		}(user, cud_publisher)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.ValidateUserResp{
			Status:  "Berhasil",
			Message: "Berhasil, Sekarang Kamu sudah memiliki akun di sistem Burung dan menjadi Bagian Dari Kami",
		},
	}
} //Tuned

func ValidateSellerRegistration(ctx context.Context, db *environment.InternalDBReadWriteSystem, OTPkey string, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "ValidateSellerRegistration"

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

	if err := rds.Del(ctx, key).Err(); err != nil {
		fmt.Println("[ValidateUserRegistration] WARNING deleting Redis key:", err)
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

	seller := sot_models.Seller{
		Nama:             userData["nama"],
		Username:         userData["username"],
		Email:            userData["email"],
		Jenis:            "Personal",
		SellerDedication: userData["seller_dedication"],
		Password:         string(hashedPassword),
	}

	if err := db.Write.WithContext(ctx).Create(&seller).Error; err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR saving to DB:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateSellerResp{
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti Ya",
			},
		}
	} else {
		go func(s sot_models.Seller, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			newSellerCreated := mb_cud_serializer.NewJsonPayload().SetPayload(s).SetTableName(s.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newSellerCreated); err != nil {
				fmt.Println("Gagal mempublish create seller ber id: ", s.ID)
			}

		}(seller, cud_publisher)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.ValidateSellerResp{
			Status:  "Berhasil",
			Message: "Berhasil, Akun mu sudah terdaftar dan Kamu Siap Berjualan Bersama Kami",
		},
	}
} //Tuned

func ValidateKurirRegistration(ctx context.Context, db *environment.InternalDBReadWriteSystem, OTPkey string, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "ValidateKurirRegistration"

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

	if err := rds.Del(ctx, key).Err(); err != nil {
		fmt.Println("[ValidateUserRegistration] WARNING deleting Redis key:", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData["password_hash"]), bcrypt.DefaultCost)

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateSellerResp{ // Tetap mempertahankan tipe payload bawaan Anda
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti",
			},
		}
	}

	kurir := sot_models.Kurir{
		Nama:         userData["nama"],
		Email:        userData["email"],
		Username:     userData["username"],
		PasswordHash: string(hashedPassword),
	}

	if err := db.Write.WithContext(ctx).Create(&kurir).Error; err != nil {
		fmt.Println("[ValidateUserRegistration] ERROR saving to DB:", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_auth.ValidateKurirResp{
				Message: "Gagal Server Sedang Sibuk, Coba Lagi Nanti Ya",
			},
		}
	} else {
		// POLA SAMA PERSIS: Menjalankan background task secara async untuk kurir
		go func(k sot_models.Kurir, publisher *mb_cud_publisher.Publisher) {
			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			// Catatan: Pastikan k.TableName() dan mb_cud_seeders.Kurir sudah terdefinisi di project Anda
			newKurirCreated := mb_cud_serializer.NewJsonPayload().SetPayload(k).SetTableName(k.TableName()).SetRole(mb_cud_seeders.Kurir)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKurirCreated); err != nil {
				fmt.Println("Gagal mempublish create kurir ber id: ", k.ID)
			}

		}(kurir, cud_publisher)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_auth.ValidateKurirResp{
			Status:  "Berhasil",
			Message: "Berhasil, Akun mu sudah terdaftar dan Kamu Siap Menjadi Bagian Dari Kami",
		},
	}
} // Tuned
