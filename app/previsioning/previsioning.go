package previsioning

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	maintain_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/maintain"
	media_storage_database_migrate "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/migrate"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/nama_kota"
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

// Helper struct untuk membaca format asli JSON
type JNELocationJSON struct {
	Kota        string `json:"kota"`
	Location    string `json:"location"`
	Coordinates string `json:"coordinates"`
	Address     string `json:"address"`
}

func ParseJNELocations() ([]sot_models.AlamatEkspedisi, error) {
	fmt.Println("\n================ [TRACE START: ParseJNELocations] ================")

	// 1. Baca file JSON
	pathFile := "C:/Burung_App/Project_Source/Backend-1/app/operational_data/jne_locations.json"
	fmt.Printf("[TRACE 1] Membaca file dari path: %s\n", pathFile)

	byteValue, err := os.ReadFile(pathFile)
	if err != nil {
		fmt.Printf("[TRACE ERROR 1] Gagal membaca file: %v\n", err)
		return nil, fmt.Errorf("gagal baca file: %w", err)
	}
	fmt.Printf("[TRACE 1 SUCCESS] Berhasil membaca file. Ukuran: %d bytes\n", len(byteValue))

	// Nyalakan ini kalau mau intip contoh isi string mentah JSON-nya (misal 200 karakter pertama)
	if len(byteValue) > 200 {
		fmt.Printf("[TRACE 1.1] Cuplikan data JSON mentah: %s...\n", string(byteValue[:200]))
	} else {
		fmt.Printf("[TRACE 1.1] Data JSON mentah: %s\n", string(byteValue))
	}

	// 2. Unmarshal ke helper struct (DIKOREKSI: Menggunakan slice/array []JNELocationJSON)
	var rawLocations [][]JNELocationJSON
	fmt.Println("[TRACE 2] Mencoba Unmarshal ke slice [][]JNELocationJSON (Double Array)...")

	err = json.Unmarshal(byteValue, &rawLocations)
	if err != nil {
		fmt.Printf("[TRACE ERROR 2] Gagal Unmarshal! Error: %v\n", err)
		fmt.Println("================ [TRACE END: FAILED] ================\n")
		return nil, fmt.Errorf("gagal unmarshal JSON: %w", err)
	}
	fmt.Printf("[TRACE 2 SUCCESS] Berhasil Unmarshal. Menemukan %d kelompok array di dalam JSON.\n", len(rawLocations))

	// 3. Mapping & Konversi koordinat ke struct AlamatEkspedisi
	var daftarAlamat []sot_models.AlamatEkspedisi
	fmt.Println("[TRACE 3] Memulai proses nested looping & splitting data koordinat...")

	// Loop tingkat 1: Membuka bungkus array pertama
	for _, subArray := range rawLocations {
		// Loop tingkat 2: Memproses objek JNELocationJSON di dalamnya
		for index, raw := range subArray {
			fmt.Printf("  -> Memproses Kota: %s, Location: %s, Coords: %s\n", raw.Kota, raw.Location, raw.Coordinates)

			var lat, long float64

			if _, exist := nama_kota.KotaJawaMap[raw.Kota]; !exist {
				continue
			}

			// Pecah string coordinates berdasarkan koma (",")
			coords := strings.Split(raw.Coordinates, ",")
			if len(coords) == 2 {
				var errLat, errLong error
				lat, errLat = strconv.ParseFloat(strings.TrimSpace(coords[0]), 64)
				long, errLong = strconv.ParseFloat(strings.TrimSpace(coords[1]), 64)

				if errLat == nil && errLong == nil {
					// [AUTO FIX] Jika angka terlalu besar (ribuan/jutaan tanpa desimal)
					// Koordinat normal Indonesia itu Lat sekitar -6 s/d -11 dan Long 95 s/d 141
					if lat > 90 || lat < -90 {
						lat = lat / 10000000.0 // sesuaikan jumlah angka 0 dengan format data asli JSON-mu
					}
					if long > 180 || long < -180 {
						long = long / 10000000.0
					}

					// [SANITY CHECK] Jika setelah diperbaiki masih ngaco, skip data ini supaya GORM ga crash
					if lat < -90 || lat > 90 || long < -180 || long > 180 {
						fmt.Printf("[⚠️ SKIP DATA CORRUPT] Koordinat %s (%s) di luar batas bumi: Lat=%f, Long=%f\n", raw.Location, raw.Kota, lat, long)
						continue
					}
				} else {
					fmt.Printf("[⚠️ SKIP DATA ERROR] Gagal parse string ke float pada %s\n", raw.Location)
					continue
				}
			} else {
				fmt.Printf("[⚠️ SKIP DATA INVALID] Format coordinates tidak valid pada %s\n", raw.Location)
				continue
			}

			// Masukkan ke struct tujuan
			alamat := sot_models.AlamatEkspedisi{
				Kota:       raw.Kota,
				NamaAlamat: raw.Address,
				Lokasi:     raw.Location,
				Latitude:   lat,
				Longitude:  long,
			}

			daftarAlamat = append(daftarAlamat, alamat)

			if index > 2000 {
				break
			}
		}
	}

	fmt.Printf("[TRACE 3 SUCCESS] Selesai me-mapping %d data ke struct AlamatEkspedisi.\n", len(daftarAlamat))
	fmt.Println("================ [TRACE END: SUCCESS] ================\n")
	return daftarAlamat, nil
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
	var dump_ekspedisi sot_models.AlamatEkspedisi = sot_models.AlamatEkspedisi{
		ID: 0,
	}
	err := db_system.Read.Model(&sot_models.AlamatEkspedisi{}).Limit(1).Take(&dump_ekspedisi).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("Error DB saat read:", err)
		return
	}

	if dump_ekspedisi.ID == 0 {
		dataAlamatEks, err := ParseJNELocations()
		if err != nil {
			fmt.Println(err)
		}

		if err := db_system.Write.CreateInBatches(&dataAlamatEks, 2000).Error; err != nil {
			fmt.Println("Gagal insert batches:", err)
		}
	}
}
