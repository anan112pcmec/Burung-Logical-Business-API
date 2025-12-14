package particular_profiling_kurir

import (
	"fmt"
	"log"
	"time"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/service/emailservices"
)

func UbahNama(id_kurir int64, username_kurir, nama string, db *config.InternalDBReadWriteSystem) ResponseUbahNama {
	if nama == "" {
		log.Printf("[WARN] Nama kosong pada permintaan ubah nama kurir ID %d", id_kurir)
		return ResponseUbahNama{
			Message: "Gagal, nama tidak boleh kosong.",
		}
	}

	var nama_kurir string
	if err := db.Read.Model(&models.Kurir{}).Select("nama").Where(&models.Kurir{
		ID:       id_kurir,
		Username: username_kurir,
	}).Limit(1).Error; err != nil {
		return ResponseUbahNama{
			Message: "Gagal Koneksi terganggu",
		}
	}

	if nama_kurir == nama {
		return ResponseUbahNama{
			Message: "Nama masih sama",
		}
	}

	if err_ubah_nama := db.Write.Model(models.Kurir{}).Where(models.Kurir{
		ID:       id_kurir,
		Username: username_kurir,
	}).Update("nama", nama).Error; err_ubah_nama != nil {
		log.Printf("[ERROR] Gagal mengubah nama kurir ID %d: %v", id_kurir, err_ubah_nama)
		return ResponseUbahNama{
			Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
		}
	}

	log.Printf("[INFO] Nama kurir berhasil diubah untuk ID %d", id_kurir)
	return ResponseUbahNama{
		Message: "Berhasil",
	}
}

func UbahUsernameKurir(db *config.InternalDBReadWriteSystem, id_kurir int64, username string) ResponseUbahUsername {
	if id_kurir == 0 {
		log.Printf("[WARN] ID kurir tidak valid pada permintaan ubah username.")
		return ResponseUbahUsername{
			Message: "Gagal, ID Kurir Tidak Valid",
		}
	}

	var countUsername int64
	saran := make([]string, 0, 3)

	if err := db.Read.Model(&models.Kurir{}).
		Where("username = ?", username).
		Count(&countUsername).Error; err != nil {
		log.Printf("[ERROR] Gagal cek username pada database: %v", err)
		return ResponseUbahUsername{
			Message: "Server Bermasalah",
		}
	}

	if countUsername == 0 {
		if err_update := db.Write.Model(&models.Kurir{}).
			Where("id = ?", id_kurir).
			Update("username", username).Error; err_update == nil {
			log.Printf("[INFO] Username kurir berhasil diubah untuk ID %d", id_kurir)
			return ResponseUbahUsername{
				Message: "Berhasil",
			}
		} else {
			log.Printf("[ERROR] Gagal mengubah username kurir ID %d: %v", id_kurir, err_update)
			return ResponseUbahUsername{
				Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
			}
		}
	}

	// jika username sudah ada, buat saran
	if countUsername > 0 {
		for len(saran) < 4 {
			usernameBaru := username + helper.GenerateRandomDigits()
			var tmp int64

			if err := db.Read.Model(&models.Kurir{}).
				Where("username = ?", usernameBaru).
				Count(&tmp).Error; err != nil {
				continue
			}

			if tmp == 0 {
				duplicate := false
				for _, v := range saran {
					if v == usernameBaru {
						duplicate = true
						break
					}
				}
				if !duplicate {
					saran = append(saran, usernameBaru)
				}
			}
		}

		log.Printf("[WARN] Username '%s' sudah digunakan, memberikan saran username untuk ID %d", username, id_kurir)
		return ResponseUbahUsername{
			Message:       "Gagal, coba gunakan nama yang disarankan",
			SaranUsername: saran,
		}
	}

	log.Printf("[ERROR] Gagal mengubah username kurir ID %d karena server sibuk", id_kurir)
	return ResponseUbahUsername{
		Message: "Gagal, Server sedang sibuk coba lagi nanti",
	}
}

func UbahEmail(id_kurir int64, username, email string, db *config.InternalDBReadWriteSystem) ResponseUbahGmail {
	if id_kurir == 0 && username == "" {
		log.Printf("[WARN] ID kurir dan username kosong pada permintaan ubah email.")
		return ResponseUbahGmail{
			Message: "Gagal, ID kurir dan username tidak valid.",
		}
	}

	var emailnya string = ""
	if err_ambil_email := db.Read.Model(models.Kurir{}).Select("email").Where(models.Kurir{
		ID:       id_kurir,
		Username: username,
	}).Limit(1).Take(&emailnya).Error; err_ambil_email != nil {
		log.Printf("[ERROR] Gagal mengambil email lama kurir ID %d: %v", id_kurir, err_ambil_email)
		return ResponseUbahGmail{
			Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
		}
	}

	if emailnya == email {
		return ResponseUbahGmail{
			Message: "Email masih sama",
		}
	}

	if err_ubah_email := db.Write.Model(models.Kurir{}).Where(models.Kurir{ID: id_kurir, Username: username}).Limit(1).Update("email", email).Error; err_ubah_email != nil {
		log.Printf("[ERROR] Gagal mengubah email kurir ID %d: %v", id_kurir, err_ubah_email)
		return ResponseUbahGmail{
			Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
		}
	}

	go func() {
		to := [...]string{
			emailnya,
		}
		cc := []string{}
		subject := "Pemberitahuan Pembaruan Gmail"
		message := fmt.Sprintf("Akun Burung Mu Telah Diubah Gmail nya Pada %s menjadi %s dan mulai sekarang semua pemberitahuan dan notif akan di berikan kepada email baru yang terkait", time.Now().Format("02-01-2006 15:04:05"), email)
		err := emailservices.SendMail(to[:], cc, subject, message)
		if err != nil {
			log.Printf("[ERROR] Gagal kirim email notifikasi perubahan email ke %s: %v", emailnya, err)
		} else {
			log.Printf("[INFO] Email notifikasi perubahan email berhasil dikirim ke %s", emailnya)
		}
	}()

	log.Printf("[INFO] Email kurir berhasil diubah untuk ID %d", id_kurir)
	return ResponseUbahGmail{
		Message: "Berhasil",
	}
}

func UbahDeskripsi(id_kurir int64, username, deskripsi string, db *config.InternalDBReadWriteSystem) ResponseUbahDeskripsi {
	if username == "" && id_kurir == 0 {
		log.Printf("[WARN] ID kurir dan username kosong pada permintaan ubah deskripsi.")
		return ResponseUbahDeskripsi{
			Message: "Gagal, ID kurir dan username tidak valid.",
		}
	}

	var deskripsi_kurir string
	if err := db.Read.Model(&models.Kurir{}).Select("deskripsi").Where(&models.Kurir{
		ID: id_kurir,
	}).Limit(1).Scan(&deskripsi_kurir).Error; err != nil {
		return ResponseUbahDeskripsi{
			Message: "Gagal koneksi server sedang terganggu",
		}
	}

	if deskripsi_kurir == deskripsi {
		return ResponseUbahDeskripsi{
			Message: "Deskripsi masih sama",
		}
	}

	if err_ubah_deskripsi := db.Write.Model(models.Kurir{}).Where(models.Kurir{
		ID:       id_kurir,
		Username: username,
	}).Limit(1).Update("deskripsi", deskripsi).Error; err_ubah_deskripsi != nil {
		log.Printf("[ERROR] Gagal mengubah deskripsi kurir ID %d: %v", id_kurir, err_ubah_deskripsi)
		return ResponseUbahDeskripsi{
			Message: "Gagal, server sedang sibuk. Coba lagi nanti.",
		}
	}

	log.Printf("[INFO] Deskripsi kurir berhasil diubah untuk ID %d", id_kurir)
	return ResponseUbahDeskripsi{
		Message: "Berhasil",
	}
}
