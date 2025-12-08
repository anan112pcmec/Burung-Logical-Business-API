package migrate

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/database/threshold"
)

func UpEntity(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	// Daftar model
	modelsToMigrate := []struct {
		name  string
		model interface{}
	}{
		{"seller", &models.Seller{}},
		{"pengguna", &models.Pengguna{}},
		{"kurir", &models.Kurir{}},
	}

	wg.Add(len(modelsToMigrate))

	for _, m := range modelsToMigrate {
		go func(mName string, mModel interface{}) {
			defer wg.Done()
			// Cek dulu apakah table sudah ada
			hasTable := db.Migrator().HasTable(mModel)
			if hasTable {
				log.Printf("Table %s already exists, skipping migration ⚠️", mName)
				return
			}

			// Kalau belum ada, lakukan migrate
			if err := db.AutoMigrate(mModel); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %s ✅", mName)
		}(m.name, m.model)
	}

	wg.Wait()
	close(errCh)

	// cek apakah ada error
	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations completed successfully 🚀")
}

func UpBarang(db *gorm.DB) {
	// BarangInduk
	if db.Migrator().HasTable(&models.BarangInduk{}) {
		log.Println("Table BarangInduk sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.BarangInduk{}); err != nil {
		log.Fatalf("Gagal Migrasi Table BarangInduk: %v", err)
	} else {
		log.Println("Migration Table BarangInduk Berhasil ✅")
	}

	// KategoriBarang
	if db.Migrator().HasTable(&models.KategoriBarang{}) {
		log.Println("Table KategoriBarang sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.KategoriBarang{}); err != nil {
		log.Fatalf("Gagal Migrasi Table KategoriBarang: %v", err)
	} else {
		log.Println("Migration Table KategoriBarang Berhasil ✅")
	}

	// VarianBarang
	if db.Migrator().HasTable(&models.VarianBarang{}) {
		log.Println("Table VarianBarang sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.VarianBarang{}); err != nil {
		log.Fatalf("Gagal Migrasi Table VarianBarang: %v", err)
	} else {
		log.Println("Migration Table VarianBarang Berhasil ✅")
	}
}

func UpTransaksi(db *gorm.DB) {
	// Transaksi
	// Pembayaran
	if db.Migrator().HasTable(&models.Pembayaran{}) {
		log.Println("Table Pembayaran sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.Pembayaran{}); err != nil {
		log.Fatalf("Gagal Membuat Table Pembayaran: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Pembayaran ✅")
	}

	if db.Migrator().HasTable(&models.Transaksi{}) {
		log.Println("Table Transaksi sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.Transaksi{}); err != nil {
		log.Fatalf("Gagal Migrasi Table Transaksi: %v", err)
	} else {
		log.Println("Berhasil membuat Table Transaksi ✅")
	}

	if db.Migrator().HasTable(&models.TransaksiFailed{}) {
		log.Println("Table Paid Failed sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.TransaksiFailed{}); err != nil {
		log.Fatalf("Gagal Migrasi Table Paid Failed: %v", err)
	} else {
		log.Println("Berhasil membuat Table Paid Failed ✅")
	}

	// Pengiriman
	if db.Migrator().HasTable(&models.Pengiriman{}) {
		log.Println("Table Pengiriman sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.Pengiriman{}); err != nil {
		log.Fatalf("Gagal Membuat Table Pengiriman: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Pengiriman ✅")
	}

	// JejakPengiriman
	if db.Migrator().HasTable(&models.JejakPengiriman{}) {
		log.Println("Table JejakPengiriman sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.JejakPengiriman{}); err != nil {
		log.Fatalf("Gagal Membuat Table JejakPengiriman: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Jejak Pengiriman ✅")
	}

	if db.Migrator().HasTable(&models.PengirimanEkspedisi{}) {
		log.Println("Table Pengiriman Ekspedisi sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.PengirimanEkspedisi{}); err != nil {
		log.Fatalf("Gagal Membuat Table Pengiriman Ekspedisi: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Pengiriman Ekspedisi ✅")
	}

	// JejakPengiriman
	if db.Migrator().HasTable(&models.JejakPengirimanEkspedisi{}) {
		log.Println("Table JejakPengiriman Ekspedisi sudah ada, skipping migration ⚠️")
	} else if err := db.AutoMigrate(&models.JejakPengirimanEkspedisi{}); err != nil {
		log.Fatalf("Gagal Membuat Table JejakPengiriman Ekspedisi: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Jejak Pengiriman Ekspedisi ✅")
	}
}

func UpEngagementEntity(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 30)

	modelsToMigrate := []interface{}{
		&models.Komentar{},
		&models.KomentarChild{},
		&models.Keranjang{},
		&models.BarangDisukai{},
		&models.Follower{},
		&models.EntitySocialMedia{},
		&models.AlamatPengguna{},
		&models.RekeningSeller{},
		&models.Jenis_Seller{},
		&models.BatalTransaksi{},
		&models.AlamatGudang{},
		&models.InformasiKendaraanKurir{},
		&models.InformasiKurir{},
		&models.AlamatKurir{},
		&models.RekeningKurir{},
		&models.DistributorData{},
		&models.BrandData{},
		&models.Etalase{},
		&models.BarangKeEtalase{},
		&models.DiskonProduk{},
		&models.BarangDiDiskon{},
		&models.Review{},
		&models.ReviewDislike{},
		&models.ReviewDislike{},
		&models.Wishlist{},
		&models.BidKurirData{},
		&models.BidKurirNonEksScheduler{},
		&models.BidKurirEksScheduler{},
		&models.PayOutSeller{},
		&models.PayOutKurir{},
	}

	wg.Add(len(modelsToMigrate))

	for _, m := range modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration ⚠️", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T ✅", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations Engagement entity completed successfully 🚀")
}

func UpSystemData(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	modelsToMigrate := []interface{}{
		&models.AlamatEkspedisi{},
	}

	wg.Add(len(modelsToMigrate))

	for _, m := range modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration ⚠️", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T ✅", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations System data completed successfully 🚀")
}

func UpTresholdData(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	modelsToMigrate := []interface{}{
		&threshold.ThresholdTransaksiSeller{},
		&threshold.ThresholdOrderSeller{},
	}

	wg.Add(len(modelsToMigrate))

	for _, m := range modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration ⚠️", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T ✅", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations treshold data completed successfully 🚀")
}

func UpMediaData(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 31)

	modelsToMigrate := []interface{}{
		&models.MediaPenggunaProfilFoto{},
		&models.MediaSellerProfilFoto{},
		&models.MediaSellerBannerFoto{},
		&models.MediaSellerTokoFisikFoto{},
		&models.MediaKurirProfilFoto{},
		&models.MediaEtalaseFoto{},
		&models.MediaBarangIndukFoto{},
		&models.MediaBarangIndukVideo{},
		&models.MediaKategoriBarangFoto{},
		&models.MediaDistributorDataDokumenFoto{},
		&models.MediaDistributorDataNPWPFoto{},
		&models.MediaDistributorDataNIBFoto{},
		&models.MediaDistributorDataSuratKerjasamaFoto{},
		&models.MediaBrandDataDokumenPerwakilanFoto{},
		&models.MediaBrandDataSertifikatFoto{},
		&models.MediaBrandDataNIBFoto{},
		&models.MediaBrandDataNPWPFoto{},
		&models.MediaBrandDataLogoBrandFoto{},
		&models.MediaBrandDataSuratKerjasamaFoto{},
		&models.MediaInformasiKendaraanKurirKendaraanFoto{},
		&models.MediaInformasiKendaraanKurirBPKBFoto{},
		&models.MediaInformasiKendaraanKurirSTNKFoto{},
		&models.MediaInformasiKurirKTPFoto{},
		&models.MediaReviewFoto{},
		&models.MediaReviewVideo{},
		&models.MediaTransaksiApprovedFoto{},
		&models.MediaTransaksiApprovedVideo{},
		&models.MediaPengirimanPickedUpFoto{},
		&models.MediaPengirimanSampaiFoto{},
		&models.MediaPengirimanEkspedisiApprovedFoto{},
		&models.MediaPengirimanEkspedisiSampaiAgentFoto{},
	}

	wg.Add(len(modelsToMigrate))

	for _, m := range modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration ⚠️", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T ✅", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations Media Data completed successfully 🚀")
}
