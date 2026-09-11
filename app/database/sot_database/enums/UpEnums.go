package enums

import (
	"fmt"

	"gorm.io/gorm"

	barang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/barang"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	kurir_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity/kurir"
	seller_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity/seller"
	keranjang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/keranjang"
	pengiriman_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/pengiriman"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
)

func UpEnumsEntity(db *gorm.DB) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	enumMap := map[string][]string{
		entity_enums.NamaEntityJenisEnums():     entity_enums.EntityJenisEnums(),
		entity_enums.NamaEntityStatusEnums():    entity_enums.EntityStatusEnums(),
		seller_enums.NamaJenisSellerEnums():     seller_enums.JenisSellerEnums(),
		seller_enums.NamaSellerDedicationEnum(): seller_enums.SellerDedicationEnums(),

		seller_enums.NamaStatusJenisSellerEnums():    seller_enums.StatusJenisSellerEnums(),
		seller_enums.NamaStatusDiskonProdukEnums():   seller_enums.StatusDiskonProdukEnums(),
		seller_enums.NamaStatusBarangDiDiskonEnums(): seller_enums.StatusBarangDiDiskonEnums(),

		kurir_enums.NamaStatusPerizinanEnums():     kurir_enums.StatusPerizinanEnums(),
		kurir_enums.NamaJenisKendaraanKurirEnums(): kurir_enums.JenisKendaraanKurirEnums(),
		kurir_enums.NamaRodaKendaraanKurirEnums():  kurir_enums.RodaKendaraanKurirEnums(),
		kurir_enums.NamaStatusKurirEnums():         kurir_enums.StatusKurirEnums(),
		kurir_enums.NamaStatusBidDataEnums():       kurir_enums.StatusBidDataEnums(),
		kurir_enums.NamaStatusBidSchedulerEnums():  kurir_enums.StatusBidSchedulerEnums(),
		kurir_enums.NamaModeBidKurirEnums():        kurir_enums.ModeBidKurirEnums(),
		kurir_enums.NamaJenisLayananKurirEnums():   kurir_enums.JenisLayananKurirEnums(),

		/* Udh bikin enum */ "nama_provinsi": {"banten", "jawa_barat", "jawa_tengah", "di_yogyakarta", "dki_jakarta", "jawa_timur"},
		/* Udh bikin enum */ "nama_kota": {
			"cilegon",
			"pandeglang",
			"lebak",
			"serang",
			"tangerang",
			"tangerang selatan",

			// Jawa Barat
			"bandung",
			"cimahi",
			"sumedang",
			"garut",
			"bandung barat",
			"cianjur",
			"bekasi",
			"bogor",
			"cirebon",
			"indramayu",
			"kuningan",
			"majalengka",
			"depok",
			"karawang",
			"purwakarta",
			"subang",
			"sukabumi",
			"tasikmalaya",
			"banjar",
			"ciamis",
			"pangandaran",

			// Jawa Tengah
			"cilacap",
			"magelang",
			"kebumen",
			"wonosobo",
			"purworejo",
			"temanggung",
			"surakarta",
			"boyolali",
			"karanganyar",
			"klaten",
			"sragen",
			"sukoharjo",
			"wonogiri",
			"semarang",
			"jepara",
			"kudus",
			"pekalongan",
			"batang",
			"blora",
			"demak",
			"kendal",
			"pati",
			"pemalang",
			"grobogan",
			"rembang",
			"salatiga",
			"purbalingga",
			"banjarnegara",
			"tegal",
			"brebes",
			"banyumas",

			// DI Yogyakarta
			"yogyakarta",
			"bantul",
			"sleman",
			"kulon progo",
			"gunung kidul",

			// DKI Jakarta
			"jakarta barat",
			"jakarta selatan",
			"jakarta pusat",
			"jakarta utara",
			"jakarta timur",
			"kepulauan seribu",

			// Jawa Timur
			"jember",
			"banyuwangi",
			"bondowoso",
			"kediri",
			"madiun",
			"magetan",
			"ngawi",
			"pacitan",
			"ponorogo",
			"mojokerto",
			"jombang",
			"nganjuk",
			"malang",
			"blitar",
			"batu",
			"probolinggo",
			"lumajang",
			"situbondo",
			"pasuruan",
			"bojonegoro",
			"surabaya",
			"gresik",
			"lamongan",
			"bangkalan",
			"pamekasan",
			"sampang",
			"sidoarjo",
			"sumenep",
			"tuban",
			"tulungagung",
			"trenggalek"},
	}

	for enumName, values := range enumMap {
		var exists bool
		checkSQL := "SELECT EXISTS(SELECT 1 FROM pg_type WHERE typname = ?);"
		if err := tx.Raw(checkSQL, enumName).Scan(&exists).Error; err != nil {
			tx.Rollback()
			return err
		}

		if !exists {
			createSQL := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", enumName, joinWithQuotes(values))
			if err := tx.Exec(createSQL).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func UpBarangEnums(db *gorm.DB) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	enumMap := map[string][]string{
		barang_enums.NamaStatusVarianBarangEnums(): barang_enums.StatusVarianBarangEnums(),
	}

	for enumName, values := range enumMap {
		// Cek apakah enum sudah ada
		var exists bool
		checkSQL := "SELECT EXISTS(SELECT 1 FROM pg_type WHERE typname = ?);"
		if err := tx.Raw(checkSQL, enumName).Scan(&exists).Error; err != nil {
			tx.Rollback()
			return err
		}

		if !exists {
			// Create type baru
			createSQL := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", enumName, joinWithQuotes(values))
			if err := tx.Exec(createSQL).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func UpEngagementEntityEnums(db *gorm.DB) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	enumMap := map[string][]string{
		pengiriman_enums.NamaStatusPengirimanNonEkspedisi(): pengiriman_enums.StatusPengirimanNonEkspedisi(),

		pengiriman_enums.NamaStatusPengirimanEkspedisiEnums(): pengiriman_enums.StatusPengirimanEkspedisiEnums(),
		keranjang_enums.NamaStatusKeranjangEnums():            keranjang_enums.StatusKeranjangEnums(),
	}

	for enumName, values := range enumMap {
		// Cek apakah enum sudah ada
		var exists bool
		checkSQL := "SELECT EXISTS(SELECT 1 FROM pg_type WHERE typname = ?);"
		if err := tx.Raw(checkSQL, enumName).Scan(&exists).Error; err != nil {
			tx.Rollback()
			return err
		}

		if !exists {
			// Create type baru
			createSQL := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", enumName, joinWithQuotes(values))
			if err := tx.Exec(createSQL).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}
func UpEnumsTransaksi(db *gorm.DB) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	enumMap := map[string][]string{
		transaksi_enums.NamaStatusTransaksiEnums(): transaksi_enums.StatusTransaksiEnums(),
		// Dibayar adalah status default sebuah transaksi sampai seller melakukan approval.
		// Setelah transaksi di-approve oleh seller, status akan berubah menjadi "Diproses".
		// Status akan berubah lagi menjadi "Waiting" setelah seller memutuskan untuk mengirim barang.
		// Status menjadi "Dikirim" ketika seller sudah menyerahkan barang ke kurir.
		// Status menjadi "Selesai" ketika pengguna telah menerima barang dan mengonfirmasi bahwa transaksi telah selesai.
		// "Dibatalkan" digunakan ketika pengguna atau seller membatalkan transaksi, baik karena kesepakatan maupun sepihak.
		// Pembatalan hanya bisa dilakukan selama status masih "Dibayar".

		transaksi_enums.NamaStatusPaidFailedEnums(): transaksi_enums.StatusPaidFailedEnums(),
		// "Ditinjau" berarti sistem sedang melakukan pemeriksaan terhadap transaksi gagal.
		// Kegagalan umumnya disebabkan oleh kesalahan foreign key, sehingga sistem akan melakukan self-healing data.
		// Setelah proses perbaikan (self-healing) selesai, status akan otomatis berubah menjadi "Pending".
		// Pada tahap "Pending", pengguna dapat memilih untuk melanjutkan atau membatalkan.
		// Secara default, jika tidak ada tindakan, status akan otomatis berubah menjadi "Batal" setelah 5 jam.
		// Jika pengguna memilih untuk melanjutkan, status berubah menjadi "Lanjut".
		// Dalam status "Lanjut", data akan dialihkan ke tabel transaksi dan pembayaran,
		// kemudian proses akan berlanjut seperti transaksi normal pada umumnya.
	}

	for enumName, values := range enumMap {
		var exists bool
		checkSQL := "SELECT EXISTS(SELECT 1 FROM pg_type WHERE typname = ?);"
		if err := tx.Raw(checkSQL, enumName).Scan(&exists).Error; err != nil {
			tx.Rollback()
			return err
		}

		if !exists {
			createSQL := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", enumName, joinWithQuotes(values))
			if err := tx.Exec(createSQL).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func joinWithQuotes(values []string) string {
	res := ""
	for i, v := range values {
		res += fmt.Sprintf("'%s'", v)
		if i != len(values)-1 {
			res += ","
		}
	}
	return res
}
