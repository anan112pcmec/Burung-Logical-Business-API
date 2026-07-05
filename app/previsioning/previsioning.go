package previsioning

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	maintain_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/maintain"
	media_storage_database_migrate "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/migrate"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/migrate"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_exchange_provisioning "github.com/anan112pcmec/Burung-backend-1/app/message_broker/provisioning/cud_exchange"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
)

func Getenvi(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func PrevisioningEnvironment(db_system *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher, media_storage *minio.Client) {
	// Database Enums & Tables Migration
	if err := enums.UpEnumsEntity(db_system.Write); err != nil {
		log.Printf("ÃƒÂ¢Ã‚ÂÃ…â€™ Gagal UpEnumsEntity: %v", err)
	}
	if err := enums.UpBarangEnums(db_system.Write); err != nil {
		log.Printf("ÃƒÂ¢Ã‚ÂÃ…â€™ Gagal UpBarangEnums: %v", err)
	}
	if err := enums.UpEnumsTransaksi(db_system.Write); err != nil {
		log.Printf("ÃƒÂ¢Ã‚ÂÃ…â€™ Gagal UpEnumsTransaksi: %v", err)
	}

	migrate.UpEntity(db_system.Write)
	migrate.UpBarang(db_system.Write)
	migrate.UpTransaksi(db_system.Write)
	migrate.UpEngagementEntity(db_system.Write)
	migrate.UpSystemData(db_system.Write)
	migrate.UpTresholdData(db_system.Write)
	migrate.UpMediaData(db_system.Write)
	migrate.UpThresholdTable(db_system.Write)

	// Message Broker Provisioning (Exchange & Queues)
	if err := mb_cud_exchange_provisioning.ProvisionExchangeCUD(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat exchange create update delete: ", err)
		return
	}
	if err := cud_publisher.QueueCreate.ProvisioningQueues(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat Queue Create: ", err)
		return
	}
	if err := cud_publisher.QueueUpdate.ProvisioningQueues(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat update queue: ", err)
		return
	}
	if err := cud_publisher.QueueDelete.ProvisioningQueues(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat queue Delete: ", err)
		return
	}

	// Media Storage Bucket Provisioning
	media_storage_database_seeders.BucketFotoName = Getenvi("MINIO_PHOTOS_BUCKET", "NIL")
	media_storage_database_seeders.BucketVideoName = Getenvi("MINIO_VIDEOS_BUCKET", "NIL")
	media_storage_database_seeders.BucketDokumenName = Getenvi("MINIO_DOKUMENS_BUCKET", "NIL")
	media_storage_database_migrate.MigrateBucketMediaStorage(media_storage)

	// ==========================================
	// PHASE: DATA CACHING & SEEDING
	// ==========================================

	// Maintain Caches
	maintain_cache.DataAlamatEkspedisiUp(db_system.Write)

	// Database Seeding (JSON Location Data)

	var KebijakanSistemdata sot_models.KebijakanSistem

	if err := db_system.Read.Model(&sot_models.KebijakanSistem{}).
		Where(&sot_models.KebijakanSistem{StatusActive: true}).
		Limit(1).Take(&KebijakanSistemdata).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Data tidak ada di DB, ambil dari JSON (Fallback)
			fmt.Println("Data KebijakanSistem tidak ditemukan di DB. Membaca dari JSON...")

			byteValue, err := os.ReadFile("C:/Burung_App/Project_Source/Backend-1/app/operational_data/kebijakan_sistem.json")
			if err != nil {
				fmt.Println("Gagal baca file JSON KebijakanSistem:", err)
				return
			}

			if err := json.Unmarshal(byteValue, &KebijakanSistemdata); err != nil {
				fmt.Println("Gagal unmarshal JSON KebijakanSistem:", err)
				return
			}

			// Simpan data dari JSON ke DB Write
			if err := db_system.Write.Create(&KebijakanSistemdata).Error; err != nil {
				fmt.Println("Gagal memasukan data KebijakanSistem ke DB:", err)
			} else {
				fmt.Println("Berhasil sinkronisasi KebijakanSistem dari JSON ke DB.")
			}
		} else {
			// Error database lainnya (misal: koneksi putus)
			fmt.Println("Error DB saat read KebijakanSistemData:", err)
			return
		}
	}

	// ==========================================
	// 2. VERSI REKENING SISTEM (Sesuai Request)
	// ==========================================
	var RekeningSistemdata sot_models.RekeningSistem

	if err := db_system.Read.Model(&sot_models.RekeningSistem{}).
		Where(&sot_models.RekeningSistem{CurrentActive: true}). // Sesuaikan field status di modelmu jika berbeda
		Limit(1).Take(&RekeningSistemdata).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Data tidak ada di DB, ambil dari JSON (Fallback)
			fmt.Println("Data RekeningSistem tidak ditemukan di DB. Membaca dari JSON...")

			byteValue, err := os.ReadFile("C:/Burung_App/Project_Source/Backend-1/app/operational_data/rekening_sistem.json")
			if err != nil {
				fmt.Println("Gagal baca file JSON RekeningSistem:", err)
				return
			}

			if err := json.Unmarshal(byteValue, &RekeningSistemdata); err != nil {
				fmt.Println("Gagal unmarshal JSON RekeningSistem:", err)
				return
			}

			// Simpan data dari JSON ke DB Write
			if err := db_system.Write.Create(&RekeningSistemdata).Error; err != nil {
				fmt.Println("Gagal memasukan data RekeningSistem ke DB:", err)
			} else {
				fmt.Println("Berhasil sinkronisasi RekeningSistem dari JSON ke DB.")
			}
		} else {
			fmt.Println("Error DB saat read RekeningSistemData:", err)
			return
		}
	}
	var dump_ekspedisi sot_models.AlamatEkspedisi
	err := db_system.Read.Model(&sot_models.AlamatEkspedisi{}).Limit(1).Take(&dump_ekspedisi).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("Error DB saat read:", err)
		return
	}

	if dump_ekspedisi.ID == 0 {
		var dataAlamatEks []sot_models.AlamatEkspedisi

		byteValue, err := os.ReadFile("../jne_location.json")
		if err != nil {
			fmt.Println("Gagal baca file:", err)
			return
		}

		if err := json.Unmarshal(byteValue, &dataAlamatEks); err != nil {
			fmt.Println("Gagal unmarshal JSON:", err)
			return
		}

		if err := db_system.Write.CreateInBatches(&dataAlamatEks, 2000).Error; err != nil {
			fmt.Println("Gagal insert batches:", err)
		}
	}
}
