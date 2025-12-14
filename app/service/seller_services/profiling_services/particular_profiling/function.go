package seller_particular_profiling

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/service/emailservices"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Particular Ubah Username Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahUsernameSeller(ctx context.Context, id_seller int32, username string, db *config.InternalDBReadWriteSystem) ResponseUbahUsername {
	var countUsername int64
	saran := make([]string, 0, 3)

	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).
		Where(&models.Seller{Username: username}).
		Count(&countUsername).Error; err != nil {
		return ResponseUbahUsername{
			Message: "Server Bermasalah",
		}
	}

	if countUsername == 0 {
		if err_seller := db.Write.WithContext(ctx).Model(&models.Seller{}).
			Where("id = ?", id_seller).
			Update("username", username).Error; err_seller == nil {
			return ResponseUbahUsername{
				Message: "Berhasil",
			}
		}
	}

	if countUsername > 0 {

		for len(saran) < 4 {

			usernameBaru := username + helper.GenerateRandomDigits()
			var tmp int64

			if err := db.Read.WithContext(ctx).Model(&models.Seller{}).
				Where("username = ?", usernameBaru).
				Count(&tmp).Error; err != nil {
				continue
			}

			if tmp == 0 {
				for _, v := range saran {
					if v == usernameBaru {
						continue
					}
				}
				saran = append(saran, usernameBaru)
			}

		}

		return ResponseUbahUsername{
			Message: "Gagal, coba gunakan nama yang disarankan",
			Saran:   saran,
		}
	}

	return ResponseUbahUsername{
		Message: "Gagal, Server sedang sibuk coba lagi nanti",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Particular Ubah Nama Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahNamaSeller(ctx context.Context, id_seller int32, nama string, db *config.InternalDBReadWriteSystem) ResponseUbahNama {

	var nama_seller string
	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Select("nama").Where(&models.Seller{
		ID: id_seller,
	}).Limit(1).Scan(&nama_seller).Error; err != nil {
		return ResponseUbahNama{
			Message: "Gagal kesalahan tidak terduga",
		}
	}

	if nama_seller == nama {
		return ResponseUbahNama{
			Message: "Nama nya masih sama",
		}
	}

	if err_db := db.Write.WithContext(ctx).Model(models.Seller{}).Where(models.Seller{ID: id_seller}).Update("nama", nama).Error; err_db == nil {
		return ResponseUbahNama{
			Message: "Berhasil",
		}
	}

	return ResponseUbahNama{
		Message: "Gagal, Server Sedang Sibuk Coba Lagi Nanti",
	}
}

func UbahEmailSeller(ctx context.Context, id_seller int32, email string, db *config.InternalDBReadWriteSystem) ResponseUbahEmail {
	sudah_ada := ""
	if err_sama := db.Read.WithContext(ctx).Model(models.Seller{}).Select("email").Where(models.Seller{ID: id_seller, Email: email}).First(&sudah_ada).Error; err_sama != nil {
		fmt.Println("Gak Ada")
	}

	if sudah_ada != "" {
		return ResponseUbahEmail{
			Message: "Gagal Mengubah Email, Coba Lagi Nanti",
		}
	}

	var email_lama string
	if err_email := db.Read.WithContext(ctx).Model(models.Seller{}).Select("email").Where(models.Seller{ID: id_seller}).First(&email_lama).Error; err_email != nil {
		return ResponseUbahEmail{
			Message: "Gagal Mengubah Email, Coba Lagi Nanti",
		}
	}

	if err_ubah_email := db.Write.WithContext(ctx).Model(models.Seller{}).Where(models.Seller{ID: id_seller}).Update("email", email).Error; err_ubah_email != nil {
		return ResponseUbahEmail{
			Message: "Gagal Mengubah Email, Coba Lagi Nanti",
		}
	}

	go func() {
		to := []string{email_lama}
		cc := []string{}
		subject := "Pemberitahuan Pembaruan Gmail"
		message := fmt.Sprintf("Akun Burung Mu Telah Diubah Gmail nya Pada %s menjadi %s dan mulai sekarang semua pemberitahuan dan notif akan di berikan kepada email baru yang terkait", time.Time{}, email)
		err := emailservices.SendMail(to, cc, subject, message)
		if err != nil {
			fmt.Println("Gagal Kirim Pesan")
		}
	}()

	return ResponseUbahEmail{
		Message: "Berhasil",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Particular Ubah Dedication Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahSellerDedicationSeller(id_seller int32, seller_dedication string, db *gorm.DB) {

}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Particular Ubah Jam Operasional Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahJamOperasionalSeller(ctx context.Context, id_seller int32, jam_operasional string, db *config.InternalDBReadWriteSystem) ResponseUbahJamOperasional {
	if id_seller == 0 {
		return ResponseUbahJamOperasional{
			Message: "Gagal Id seller tidak Valid",
		}
	}

	var jam_operasional_seller string
	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Select("jam_operasional").Where(&models.Seller{
		ID: id_seller,
	}).Limit(1).Scan(&jam_operasional_seller).Error; err != nil {
		return ResponseUbahJamOperasional{
			Message: "Gagal koneksi terganggu",
		}
	}

	if jam_operasional_seller == jam_operasional {
		return ResponseUbahJamOperasional{
			Message: "Jam operasional masih sama",
		}
	}

	if err_ubah := db.Write.WithContext(ctx).Model(models.Seller{}).Where(models.Seller{ID: id_seller}).Update("jam_operasional", jam_operasional).Error; err_ubah != nil {
		return ResponseUbahJamOperasional{
			Message: "Gagal Server Sedang sibuk, Coba Lagi lain waktu",
		}
	}

	return ResponseUbahJamOperasional{
		Message: "Berhasil",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Particular Ubah Punchline Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahPunchlineSeller(ctx context.Context, id_seller int32, punchline string, db *config.InternalDBReadWriteSystem) ResponseUbahPunchline {
	if id_seller == 0 {
		return ResponseUbahPunchline{
			Message: "Gagal, Seller Tidak Valid",
		}
	}

	var punchline_seller string
	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Select("punchline").Where(&models.Seller{
		ID: id_seller,
	}).Limit(1).Error; err != nil {
		return ResponseUbahPunchline{
			Message: "Gagal koneksi terganggu",
		}
	}

	if punchline_seller == punchline {
		return ResponseUbahPunchline{
			Message: "Punchline masih sama",
		}
	}

	if err_ubah := db.Write.WithContext(ctx).Model(models.Seller{}).Where(models.Seller{ID: id_seller}).Update("punchline", punchline).Error; err_ubah != nil {
		return ResponseUbahPunchline{
			Message: "Gagal Mungkin Server Sedang sibuk",
		}
	}

	return ResponseUbahPunchline{
		Message: "Berhasil",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Particular Ubah Deskripsi Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func UbahDeskripsiSeller(ctx context.Context, id_seller int32, deskripsi string, db *config.InternalDBReadWriteSystem) ResponseUbahDeskripsi {
	if id_seller == 0 {
		return ResponseUbahDeskripsi{
			Message: "Gagal, Seller kredensial tidak valid",
		}
	}

	var deskripsi_seller string
	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Select("deskripsi").Where(&models.Seller{
		ID: id_seller,
	}).Limit(1).Scan(&deskripsi_seller).Error; err != nil {
		return ResponseUbahDeskripsi{
			Message: "Gagal koneksi terganggu",
		}
	}

	if deskripsi_seller == deskripsi {
		return ResponseUbahDeskripsi{
			Message: "Deskripsi masih sama",
		}
	}

	if err_ubah := db.Write.WithContext(ctx).Model(models.Seller{}).Where(models.Seller{ID: id_seller}).Update("deskripsi", deskripsi).Error; err_ubah != nil {
		return ResponseUbahDeskripsi{
			Message: "Gagal Server Sedang Sibuk, Coba Lagi Lain Waktu",
		}
	}

	return ResponseUbahDeskripsi{
		Message: "Berhasil",
	}
}

func UbahSellerDedication(ctx context.Context, id_seller int32, dedication string, db *config.InternalDBReadWriteSystem) ResponseUbahDedication {
	if id_seller == 0 {
		return ResponseUbahDedication{
			Message: "Gagal id tidak valid",
		}
	}

	var dedication_seller string
	if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Select("seller_dedication").Where(&models.Seller{
		ID: id_seller,
	}).Limit(1).Scan(&dedication_seller).Error; err != nil {
		return ResponseUbahDedication{
			Message: "Gagal koneksi terganggu",
		}
	}

	if dedication_seller == dedication {
		return ResponseUbahDedication{
			Message: "Dedication masih sama",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Seller{}).Where(&models.Seller{
		ID: id_seller,
	}).Update("seller_dedication", dedication).Error; err != nil {
		return ResponseUbahDedication{
			Message: "Gagal mengubah dedication",
		}
	}

	return ResponseUbahDedication{
		Message: "Berhasil mengubah dedication",
	}
}
