package migrate

import (
	"log"
	"sync"

	"gorm.io/gorm"

	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
)

func UpEntity(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	// Daftar model
	sot_modelsToMigrate := []struct {
		name  string
		model interface{}
	}{
		{"seller", &sot_models.Seller{}},
		{"pengguna", &sot_models.Pengguna{}},
		{"kurir", &sot_models.Kurir{}},
	}

	wg.Add(len(sot_modelsToMigrate))

	for _, m := range sot_modelsToMigrate {
		go func(mName string, mModel interface{}) {
			defer wg.Done()
			// Cek dulu apakah table sudah ada
			hasTable := db.Migrator().HasTable(mModel)
			if hasTable {
				log.Printf("Table %s already exists, skipping migration Ã¢Å¡Â Ã¯Â¸Â", mName)
				return
			}

			// Kalau belum ada, lakukan migrate
			if err := db.AutoMigrate(mModel); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %s Ã¢Å“â€¦", mName)
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

	log.Println("All migrations completed successfully Ã°Å¸Å¡â‚¬")
}

func UpBarang(db *gorm.DB) {
	// BarangInduk
	if db.Migrator().HasTable(&sot_models.BarangInduk{}) {
		log.Println("Table BarangInduk sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.BarangInduk{}); err != nil {
		log.Fatalf("Gagal Migrasi Table BarangInduk: %v", err)
	} else {
		log.Println("Migration Table BarangInduk Berhasil Ã¢Å“â€¦")
	}

	// KategoriBarang
	if db.Migrator().HasTable(&sot_models.KategoriBarang{}) {
		log.Println("Table KategoriBarang sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.KategoriBarang{}); err != nil {
		log.Fatalf("Gagal Migrasi Table KategoriBarang: %v", err)
	} else {
		log.Println("Migration Table KategoriBarang Berhasil Ã¢Å“â€¦")
	}

	// VarianBarang
	if db.Migrator().HasTable(&sot_models.VarianBarang{}) {
		log.Println("Table VarianBarang sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.VarianBarang{}); err != nil {
		log.Fatalf("Gagal Migrasi Table VarianBarang: %v", err)
	} else {
		log.Println("Migration Table VarianBarang Berhasil Ã¢Å“â€¦")
	}
}

func UpTransaksi(db *gorm.DB) {
	// Transaksi
	// Pembayaran
	if db.Migrator().HasTable(&sot_models.Pembayaran{}) {
		log.Println("Table Pembayaran sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.Pembayaran{}); err != nil {
		log.Fatalf("Gagal Membuat Table Pembayaran: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Pembayaran Ã¢Å“â€¦")
	}

	if db.Migrator().HasTable(&sot_models.Transaksi{}) {
		log.Println("Table Transaksi sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.Transaksi{}); err != nil {
		log.Fatalf("Gagal Migrasi Table Transaksi: %v", err)
	} else {
		log.Println("Berhasil membuat Table Transaksi Ã¢Å“â€¦")
	}

	if db.Migrator().HasTable(&sot_models.TransaksiFailed{}) {
		log.Println("Table Paid Failed sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.TransaksiFailed{}); err != nil {
		log.Fatalf("Gagal Migrasi Table Paid Failed: %v", err)
	} else {
		log.Println("Berhasil membuat Table Paid Failed Ã¢Å“â€¦")
	}

	// Pengiriman
	if db.Migrator().HasTable(&sot_models.Pengiriman{}) {
		log.Println("Table Pengiriman sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.Pengiriman{}); err != nil {
		log.Fatalf("Gagal Membuat Table Pengiriman: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Pengiriman Ã¢Å“â€¦")
	}

	// JejakPengiriman
	if db.Migrator().HasTable(&sot_models.JejakPengiriman{}) {
		log.Println("Table JejakPengiriman sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.JejakPengiriman{}); err != nil {
		log.Fatalf("Gagal Membuat Table JejakPengiriman: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Jejak Pengiriman Ã¢Å“â€¦")
	}

	if db.Migrator().HasTable(&sot_models.PengirimanEkspedisi{}) {
		log.Println("Table Pengiriman Ekspedisi sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.PengirimanEkspedisi{}); err != nil {
		log.Fatalf("Gagal Membuat Table Pengiriman Ekspedisi: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Pengiriman Ekspedisi Ã¢Å“â€¦")
	}

	// JejakPengiriman
	if db.Migrator().HasTable(&sot_models.JejakPengirimanEkspedisi{}) {
		log.Println("Table JejakPengiriman Ekspedisi sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â")
	} else if err := db.AutoMigrate(&sot_models.JejakPengirimanEkspedisi{}); err != nil {
		log.Fatalf("Gagal Membuat Table JejakPengiriman Ekspedisi: %v", err)
	} else {
		log.Println("Berhasil Membuat Table Jejak Pengiriman Ekspedisi Ã¢Å“â€¦")
	}
}

func UpEngagementEntity(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 30)

	sot_modelsToMigrate := []interface{}{
		&sot_models.Komentar{},
		&sot_models.KomentarChild{},
		&sot_models.Keranjang{},
		&sot_models.BarangDisukai{},
		&sot_models.Follower{},
		&sot_models.EntitySocialMedia{},
		&sot_models.AlamatPengguna{},
		&sot_models.RekeningSeller{},
		&sot_models.Jenis_Seller{},
		&sot_models.BatalTransaksi{},
		&sot_models.AlamatGudang{},
		&sot_models.InformasiKendaraanKurir{},
		&sot_models.InformasiKurir{},
		&sot_models.AlamatKurir{},
		&sot_models.RekeningKurir{},
		&sot_models.DistributorData{},
		&sot_models.BrandData{},
		&sot_models.Etalase{},
		&sot_models.BarangKeEtalase{},
		&sot_models.DiskonProduk{},
		&sot_models.BarangDiDiskon{},
		&sot_models.Review{},
		&sot_models.ReviewLike{},
		&sot_models.ReviewDislike{},
		&sot_models.Wishlist{},
		&sot_models.BidKurirData{},
		&sot_models.BidKurirNonEksScheduler{},
		&sot_models.BidKurirEksScheduler{},
		&sot_models.PayOutSeller{},
		&sot_models.PayOutKurir{},
	}

	wg.Add(len(sot_modelsToMigrate))

	for _, m := range sot_modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T Ã¢Å“â€¦", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations Engagement entity completed successfully Ã°Å¸Å¡â‚¬")
}

func UpSystemData(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	sot_modelsToMigrate := []interface{}{
		&sot_models.AlamatEkspedisi{},
		&sot_models.KebijakanSistem{},
		&sot_models.RekeningSistem{},
		&sot_models.PayOutSistem{},
	}

	wg.Add(len(sot_modelsToMigrate))

	for _, m := range sot_modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T Ã¢Å“â€¦", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations System data completed successfully Ã°Å¸Å¡â‚¬")
}

func UpTresholdData(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	sot_modelsToMigrate := []interface{}{
		&sot_threshold.ThresholdOrderSeller{},
	}

	wg.Add(len(sot_modelsToMigrate))

	for _, m := range sot_modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T Ã¢Å“â€¦", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations treshold data completed successfully Ã°Å¸Å¡â‚¬")
}

func UpMediaData(db *gorm.DB) {
	var wg sync.WaitGroup
	errCh := make(chan error, 31)

	sot_modelsToMigrate := []interface{}{
		&sot_models.MediaPenggunaProfilFoto{},
		&sot_models.MediaSellerProfilFoto{},
		&sot_models.MediaSellerBannerFoto{},
		&sot_models.MediaSellerTokoFisikFoto{},
		&sot_models.MediaKurirProfilFoto{},
		&sot_models.MediaEtalaseFoto{},
		&sot_models.MediaBarangIndukFoto{},
		&sot_models.MediaBarangIndukVideo{},
		&sot_models.MediaKategoriBarangFoto{},
		&sot_models.MediaDistributorDataDokumen{},
		&sot_models.MediaDistributorDataNPWPFoto{},
		&sot_models.MediaDistributorDataNIBFoto{},
		&sot_models.MediaDistributorDataSuratKerjasamaDokumen{},
		&sot_models.MediaBrandDataPerwakilanDokumen{},
		&sot_models.MediaBrandDataSertifikatFoto{},
		&sot_models.MediaBrandDataNIBFoto{},
		&sot_models.MediaBrandDataNPWPFoto{},
		&sot_models.MediaBrandDataLogoFoto{},
		&sot_models.MediaBrandDataSuratKerjasamaDokumen{},
		&sot_models.MediaInformasiKendaraanKurirKendaraanFoto{},
		&sot_models.MediaInformasiKendaraanKurirBPKBFoto{},
		&sot_models.MediaInformasiKendaraanKurirSTNKFoto{},
		&sot_models.MediaInformasiKurirKTPFoto{},
		&sot_models.MediaReviewFoto{},
		&sot_models.MediaReviewVideo{},
		&sot_models.MediaTransaksiApprovedFoto{},
		&sot_models.MediaTransaksiApprovedVideo{},
		&sot_models.MediaPengirimanPickedUpFoto{},
		&sot_models.MediaPengirimanSampaiFoto{},
		&sot_models.MediaPengirimanEkspedisiPickedUpFoto{},
		&sot_models.MediaPengirimanEkspedisiSampaiAgentFoto{},
	}

	wg.Add(len(sot_modelsToMigrate))

	for _, m := range sot_modelsToMigrate {
		go func(model interface{}) {
			defer wg.Done()
			if db.Migrator().HasTable(model) {
				log.Printf("Table %T sudah ada, skipping migration Ã¢Å¡Â Ã¯Â¸Â", model)
				return
			}

			if err := db.AutoMigrate(model); err != nil {
				errCh <- err
				return
			}
			log.Printf("Migration success: %T Ã¢Å“â€¦", model)
		}(m)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations Media Data completed successfully Ã°Å¸Å¡â‚¬")
}
