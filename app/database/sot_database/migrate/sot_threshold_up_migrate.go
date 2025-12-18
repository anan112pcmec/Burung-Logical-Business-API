package migrate

import (
	"log"
	"sync"

	"gorm.io/gorm"

	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"

)

func UpThresholdTable(db *gorm.DB) {
	// Tabel tanpa FK dependency
	independent := []interface{}{
		sot_threshold.PenggunaThreshold{},
		sot_threshold.KurirThreshold{},
		sot_threshold.SellerThreshold{},
		sot_threshold.BarangIndukThreshold{},
		sot_threshold.KategoriBarangThreshold{},
	}

	// Tabel dependent (FK harus dibuat setelah referensi)
	dependent := []interface{}{
		sot_threshold.AlamatGudangThreshold{},
		sot_threshold.TransaksiThreshold{},
		sot_threshold.PembayaranThreshold{},
		sot_threshold.PengirimanEkspedisiThreshold{},
		sot_threshold.PengirimanNonEkspedisiThreshold{},
		sot_threshold.BrandDataThreshold{},
		sot_threshold.DistributorDataThreshold{},
		sot_threshold.EtalaseThreshold{},
		sot_threshold.ReviewThreshold{},
		sot_threshold.InformasiKendaraanKurirThreshold{},
		sot_threshold.BidKurirDataThreshold{},
		sot_threshold.DiskonProdukThreshold{},
		sot_threshold.KomentarThreshold{},
		sot_threshold.InformasiKurirThreshold{},
		sot_threshold.RekeningSellerThreshold{},
		sot_threshold.AlamatEkspedisiThreshold{},
	}

	// --- parallel untuk independent ---
	var wg sync.WaitGroup
	errCh := make(chan error, len(independent))

	wg.Add(len(independent))
	for _, m := range independent {
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

	// --- sekuensial untuk dependent ---
	for _, m := range dependent {
		if db.Migrator().HasTable(m) {
			log.Printf("Table %T sudah ada, skipping migration ⚠️", m)
			continue
		}
		if err := db.AutoMigrate(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Printf("Migration success: %T ✅", m)
	}

	close(errCh)
	for err := range errCh {
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("All migrations Media Data completed successfully 🚀")
}
