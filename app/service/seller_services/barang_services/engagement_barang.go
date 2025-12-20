package seller_barang_service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	barang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/barang"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/seller_dedication"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_komentar "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/komentar"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Masukan Barang Seller
// Berfungsi untuk melayani seller yang hendak memasukan barang nya ke sistem burung
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadMasukanBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal memasukkan barang karena kredensial seller tidak valid",
		}
	}

	if _, ok := seller_dedication.CategoryMap[data.BarangInduk.JenisBarang]; !ok {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal jenis barang tidak dikenal",
		}
	}

	data.BarangInduk.SellerID = data.IdentitasSeller.IdSeller

	var id_data_barang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).
		Select("id").
		Where(&models.BarangInduk{
			SellerID:   data.IdentitasSeller.IdSeller,
			NamaBarang: data.BarangInduk.NamaBarang,
		}).Limit(1).Scan(&id_data_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_barang != 0 {
		log.Printf("[WARN] Seller ID %d sudah memiliki barang dengan nama '%s'", data.IdentitasSeller.IdSeller, data.BarangInduk.NamaBarang)
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal: Nama barang sudah terdaftar untuk seller ini",
		}
	}

	var harga_original int64 = 0
	var success bool = false
	for i, _ := range data.KategoriBarang {
		if data.KategoriBarang[i].IsOriginal {
			success = true
			harga_original = int64(data.KategoriBarang[i].Harga)
			break
		}
	}

	if !success || harga_original <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Harga original tidak boleh 0",
		}
	}

	barang_induk := models.BarangInduk{
		SellerID:       data.IdentitasSeller.IdSeller,
		NamaBarang:     data.BarangInduk.NamaBarang,
		JenisBarang:    data.BarangInduk.JenisBarang,
		Deskripsi:      data.BarangInduk.Deskripsi,
		HargaKategoris: int32(harga_original),
	}
	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := tx.WithContext(ctx).Create(&barang_induk).Error; err != nil {
			return err
		}

		for i, _ := range data.KategoriBarang {
			data.KategoriBarang[i].IdBarangInduk = int32(barang_induk.ID)
			data.KategoriBarang[i].SellerID = data.IdentitasSeller.IdSeller
			data.KategoriBarang[i].IDAlamat = data.IdAlamatGudang
			data.KategoriBarang[i].IDRekening = data.IdRekening
		}

		if err := tx.WithContext(ctx).CreateInBatches(&data.KategoriBarang, len(data.KategoriBarang)).Error; err != nil {
			return err
		}

		var id_origin_kategori int64 = 0
		if err := tx.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
			IdBarangInduk: int32(barang_induk.ID),
			IsOriginal:    true,
		}).Limit(1).Scan(&id_origin_kategori).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BarangInduk{}).Where(&models.BarangInduk{
			ID: barang_induk.ID,
		}).Update("original_kategori", id_origin_kategori).Error; err != nil {
			return err
		}

		var id_kategoris []int64
		if err := tx.Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
			IdBarangInduk: barang_induk.ID,
		}).Limit(len(data.KategoriBarang)).Scan(&id_kategoris).Error; err != nil {
			return err
		}

		var totalBatchVarian int64 = 0
		var varian_barang []models.VarianBarang
		for i, _ := range id_kategoris {
			var kategori models.KategoriBarang
			if err := tx.WithContext(ctx).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: id_kategoris[i],
			}).Limit(1).Take(&kategori).Error; err != nil {
				return err
			}

			for i := 0; i < int(kategori.Stok); i++ {
				varian_barang = append(varian_barang, models.VarianBarang{
					IdBarangInduk: barang_induk.ID,
					IdKategori:    kategori.ID,
					Sku:           kategori.Sku,
					Status:        "Ready",
				})
			}

			totalBatchVarian += int64(kategori.Stok)
		}

		if err := tx.WithContext(ctx).CreateInBatches(&varian_barang, int(totalBatchVarian)).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		fmt.Println(err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(Bi models.BarangInduk, Kb []models.KategoriBarang, Trh *config.InternalDBReadWriteSystem, publisher *mb_cud_publisher.Publisher) {
		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Bi.SellerID),
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Bi.ID),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdSeller.Increment(konteks, Trh.Write, stsk_seller.BarangInduk); err != nil {
			fmt.Println("Gagal increment barang induk counter ke threshold seller")
		}

		if err := thresholdBarangInduk.Inisialisasi(konteks, Trh.Write); err != nil {
			fmt.Println("Gagal inisialisasi thresholdbarang induk")
		}

		var dataBarangIndukNew models.BarangInduk
		if err := Trh.Read.WithContext(ctx).Model(&models.BarangInduk{}).Where(&models.BarangInduk{
			ID: Bi.ID,
		}).Limit(1).Take(&dataBarangIndukNew).Error; err != nil {
			fmt.Println("Gagal mendapatkan data barang induk")
		}

		createNewBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBarangIndukNew).SetTableName(dataBarangIndukNew.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createNewBarangIndukPublish); err != nil {
			fmt.Println("Gagal publish create new barang induk ke message broker")
		}

		for _, k := range Kb {
			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: k.ID,
			}

			if err := thresholdKategoriBarang.Inisialisasi(konteks, Trh.Write); err != nil {
				fmt.Println("Gagal membuat threshold untuk kategori ber Id: ", k.ID, " Dan Bernama: ", k.Nama)
			}

			createNewKategoriBarangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(k).SetTableName(k.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createNewKategoriBarangPublish); err != nil {
				fmt.Println("Gagal publish create new kategori ke message broker")
			}
		}

	}(barang_induk, data.KategoriBarang, db, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Barang berhasil ditambahkan.",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Edit Barang Induk
// Berfungsi untuk seller dalam melakukan edit atau pembaruan informasi seputar barang induknya
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadEditBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial Seller Tidak Valid",
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       int32(data.IdBarangInduk),
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang_induk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_barang_induk == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data barang tidak valid",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.BarangInduk{}).Where(&models.BarangInduk{
		ID: int32(data.IdBarangInduk),
	}).Updates(&models.BarangInduk{
		NamaBarang:  data.NamaBarang,
		JenisBarang: data.JenisBarang,
		Deskripsi:   data.Deskripsi,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdBarangInduk int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var updatedBarangInduk models.BarangInduk
		if err := Read.WithContext(konteks).Model(&models.BarangInduk{}).Where(&models.BarangInduk{
			ID: int32(IdBarangInduk),
		}).Limit(1).Take(&updatedBarangInduk).Error; err != nil {
			fmt.Println("Gagal mengambil data terbaru barang induk")
			return
		}

		updatedBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(updatedBarangInduk).SetTableName(updatedBarangInduk.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedBarangIndukPublish); err != nil {
			fmt.Println("Gagal publish update barang induk ke message broker")
		}

	}(id_data_barang_induk, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Barang Berhasil Diubah.",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Barang Induk
// Berfungsi untuk seller dalam menghapus barang induknya, akan otomatis menghapus kategori barang dan varian barang
// nya
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadHapusBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusBarang"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
		}
	}

	var dataBarangInduk models.BarangInduk
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Where(&models.BarangInduk{
		ID:       int32(data.IdBarangInduk),
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&dataBarangInduk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if dataBarangInduk.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data barang tidak valid",
		}
	}

	// Cek apakah masih ada varian dalam transaksi (status: Dipesan/Diproses)
	var id_varian_dalam_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").
		Where("id_barang_induk = ? AND status IN ?", dataBarangInduk.ID, []string{"Dipesan", "Diproses"}).
		Limit(1).Scan(&id_varian_dalam_transaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada database",
		}
	}

	// ✅ 3. Early exit jika masih ada transaksi aktif
	if id_varian_dalam_transaksi > 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Masih ada varian dalam transaksi",
		}
	}

	thresholdBarangInduk := sot_threshold.BarangIndukThreshold{ID: int64(dataBarangInduk.ID)}
	_, totalKategoriBarang := thresholdBarangInduk.GetKolomCount(ctx, db.Read, stsk_baranginduk.KategoriBarang)

	var dataKategoriBarang []models.KategoriBarang = make([]models.KategoriBarang, 0, totalKategoriBarang)
	_ = db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
		IdBarangInduk: dataBarangInduk.ID,
	}).Limit(totalKategoriBarang).Take(&dataKategoriBarang)

	// Jalankan proses penghapusan dalam goroutine (asynchronous)
	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 🔸 Hapus varian (permanent delete)
		if err := tx.Unscoped().Model(&models.VarianBarang{}).Where(&models.VarianBarang{IdBarangInduk: int32(data.IdBarangInduk)}).
			Delete(&models.VarianBarang{}).Error; err != nil {
			return fmt.Errorf("hapus varian gagal: %w", err)
		}

		// 🔸 Hapus kategori (soft delete)
		if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{IdBarangInduk: int32(data.IdBarangInduk)}).
			Delete(&models.KategoriBarang{}).Error; err != nil {
			return fmt.Errorf("hapus kategori gagal: %w", err)
		}

		// 🔸 Hapus barang induk (soft delete)
		if err := tx.Model(&models.BarangInduk{}).Where(&models.BarangInduk{ID: int32(data.IdBarangInduk)}).
			Delete(&models.BarangInduk{}).Error; err != nil {
			return fmt.Errorf("hapus barang induk gagal: %w", err)
		}

		return nil
	}); err != nil {
		fmt.Println(err)

		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal menghapus barang.",
		}
	}

	go func(DBI models.BarangInduk, DBK []models.KategoriBarang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := Trh.WithContext(konteks).Model(&sot_threshold.BarangIndukThreshold{}).Where(&sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(DBI.ID),
		}).Delete(&sot_threshold.BarangIndukThreshold{}).Error; err != nil {
			fmt.Println("Gagal menghapus barang_induk threshold")
		}

		deleteBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(DBI).SetTableName(DBI.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteBarangIndukPublish); err != nil {
			fmt.Printf("Gagal publish delete barang induk ber id: %v, bernama %s ke message broker", DBI.ID, DBI.NamaBarang)
		}

		for _, dbkategori := range DBK {
			if err := Trh.WithContext(konteks).Model(&sot_threshold.KategoriBarangThreshold{}).Where(&sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: dbkategori.ID,
			}).Delete(&sot_threshold.KategoriBarangThreshold{}).Error; err != nil {
				fmt.Printf("Gagal menghapus data kategori barang ber id %v dan bernama %s", dbkategori.ID, dbkategori.Nama)
			}

			deleteKategoriBarangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dbkategori).SetTableName(dbkategori.TableName())
			if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteKategoriBarangPublish); err != nil {
				fmt.Printf("Gagal publish delete kategori barang ber id %v dan bernama %s ke message broker", dbkategori.ID, dbkategori.Nama)
			}
		}

	}(dataBarangInduk, dataKategoriBarang, db.Write, cud_publisher)
	// Kembalikan respons sukses
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Barang berhasil dihapus.",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Tambah Kategori Barang Induk
// Berfungsi untuk seller menambahkan kategori barang pada barang induk
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TambahKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadTambahKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahKategoriBarang"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: kredensial seller tidak valid",
		}
	}

	// Pastikan barang induk milik seller
	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       data.IdBarangInduk,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang_induk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_barang_induk == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data barang tidak valid",
		}
	}

	var id_data_alamat int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Select("id").Where(&models.AlamatGudang{
		ID:       data.IdAlamatGudang,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_alamat).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_alamat == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Alamat Tidak Valid",
		}
	}

	var id_data_rekening int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.RekeningSeller{}).Select("id").Where(&models.RekeningSeller{
		ID:       data.IdRekening,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_rekening).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_rekening == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Data Rekening Tidak Valid",
		}
	}

	var kategori_barang []models.KategoriBarang

	// 1️⃣ Loop validasi dan siapkan batch kategori
	for i := range data.KategoriBarang {
		var id_data_kategori_barang int64 = 0

		// Cek apakah kategori dengan nama yang sama sudah ada
		if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).
			Select("id").
			Where(&models.KategoriBarang{
				IdBarangInduk: data.IdBarangInduk,
				Nama:          data.KategoriBarang[i].Nama,
			}).
			Limit(1).
			Scan(&id_data_kategori_barang).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

		// Lewati jika sudah ada
		if id_data_kategori_barang != 0 {
			continue
		}

		// Tambahkan kategori baru ke batch
		kategori_barang = append(kategori_barang, models.KategoriBarang{
			SellerID:       data.IdentitasSeller.IdSeller,
			IdBarangInduk:  data.IdBarangInduk,
			IDAlamat:       data.IdAlamatGudang,
			IDRekening:     data.IdRekening,
			Nama:           data.KategoriBarang[i].Nama,
			Deskripsi:      data.KategoriBarang[i].Deskripsi,
			Warna:          data.KategoriBarang[i].Warna,
			Stok:           data.KategoriBarang[i].Stok,
			Harga:          data.KategoriBarang[i].Harga,
			BeratGram:      data.KategoriBarang[i].BeratGram,
			DimensiPanjang: data.KategoriBarang[i].DimensiPanjang,
			DimensiLebar:   data.KategoriBarang[i].DimensiLebar,
			Sku:            data.KategoriBarang[i].Sku,
			IsOriginal:     false,
		})
	}
	// Jalankan async tapi dengan salinan data yang aman
	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := tx.CreateInBatches(&kategori_barang, len(kategori_barang)).Error; err != nil {
			return err
		}

		var varian_barang_total []models.VarianBarang
		var VarianBatch int64 = 0

		for _, kategori := range kategori_barang {
			for s := 0; s < int(kategori.Stok); s++ {
				varian_barang_total = append(varian_barang_total, models.VarianBarang{
					IdBarangInduk: data.IdBarangInduk,
					IdKategori:    kategori.ID, // 🧠 langsung pakai ID dari hasil insert batch
					Sku:           kategori.Sku,
					Status:        "Ready",
				})
			}
			VarianBatch += int64(kategori.Stok)
		}

		if len(varian_barang_total) > 0 {
			if err := tx.CreateInBatches(&varian_barang_total, int(VarianBatch)).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	log.Printf("[INFO] Permintaan tambah kategori diterima untuk BarangInduk ID %d oleh Seller ID %d",
		data.IdBarangInduk, data.IdentitasSeller.IdSeller)

	go func(Kb []models.KategoriBarang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		for _, kategoribarangdata := range Kb {
			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: kategoribarangdata.ID,
			}

			if err := thresholdKategoriBarang.Inisialisasi(konteks, Trh); err != nil {
				fmt.Printf("Gagal membuat threshold kategori barang ber id %v, bernama %s", kategoribarangdata.ID, kategoribarangdata.Nama)
			}

			newKategoriBarangCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(kategoribarangdata).SetTableName(kategoribarangdata.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKategoriBarangCreatePublish); err != nil {
				fmt.Printf("Gagal mempublish create kategori barang ber id %v dan bernama %s ke message broker", kategoribarangdata.ID, kategoribarangdata.Nama)
			}
		}
	}(data.KategoriBarang, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kategori barang berhasil ditambahkan (async-safe).",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Edit Kategori Barang
// Berfungsi untuk mengedit data informasi tentang kategori barang induk yang dituju
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadEditKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditKategoriBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
		ID:            data.IdKategoriBarang,
		IdBarangInduk: data.IdBarangInduk,
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_kategori).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_kategori == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kategori barang tidak valid",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: data.IdKategoriBarang,
		}).Updates(&models.KategoriBarang{
			Nama:           data.Nama,
			Deskripsi:      data.Deskripsi,
			Warna:          data.Warna,
			DimensiPanjang: data.DimensiPanjang,
			DimensiLebar:   data.DimensiLebar,
			Sku:            data.Sku,
		}).Error; err != nil {
			return err
		}

		if data.Sku != "" {
			if err := tx.Model(&models.VarianBarang{}).Where(&models.VarianBarang{
				IdKategori: data.IdKategoriBarang,
			}).Updates(&models.VarianBarang{
				Sku: data.Sku,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		fmt.Println(err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdKb int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKategoriBarangUpdated models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName(dataKategoriBarangUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update kategori barang Id: %v ke message broker", IdKb)
		}
	}(data.IdKategoriBarang, db.Read, cud_publisher)

	log.Printf("[INFO] Kategori barang berhasil diubah pada barang induk ID %d oleh seller ID %d", data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kategori barang berhasil diubah.",
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Kategori Barang Induk
// Berfungsi untuk menghapus kategori barang induk yang ada
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadHapusKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKategoriBarang"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
		}
	}

	var data_kategori models.KategoriBarang
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
		ID:            data.IdKategoriBarang,
		IdBarangInduk: data.IdBarangInduk,
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&data_kategori).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	if data_kategori.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kategori barang tidak valid",
		}
	}

	// Cek apakah kategori yang akan dihapus masih punya varian dalam transaksi

	var exist_varian_transaksi int64 = 0
	if errStock := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").
		Where("id_barang_induk = ? AND id_kategori = ? AND status IN ?", data.IdBarangInduk, data.IdKategoriBarang, []string{"Dipesan", "Diproses"}).
		Limit(1).Scan(&exist_varian_transaksi).Error; errStock != nil {
	}

	if exist_varian_transaksi != 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal Kategori ini masih ada dalam transaksi yang belum selesai, Down kan dulu",
		}
	}

	// Jalankan proses penghapusan di goroutine
	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VarianBarang{}).Where(&models.VarianBarang{
			IdKategori: data.IdKategoriBarang,
		}).Delete(&models.KategoriBarang{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: data.IdKategoriBarang,
		}).Delete(&models.KategoriBarang{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	log.Printf("[INFO] Kategori barang berhasil dihapus (soft delete manual) pada BarangInduk ID %d oleh Seller ID %d",
		data.IdBarangInduk, data.IdentitasSeller.IdSeller)

	go func(Kb models.KategoriBarang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := Trh.WithContext(konteks).Model(&sot_threshold.KategoriBarangThreshold{}).Where(&sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Kb.ID,
		}).Delete(&sot_threshold.KategoriBarangThreshold{}).Error; err != nil {
			fmt.Println("Gagal menghapus threshold kategori barang ber ID:%v", Kb.ID)
		}

		kategoriBarangDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kb).SetTableName(Kb.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangDeletePublish); err != nil {
			fmt.Printf("Gagal publish hapus kategori barang ke message broker")
		}

	}(data_kategori, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kategori barang berhasil dihapus (soft delete manual).",
	}
}

// ////////////////////////////////////////////////////////////////////////////////
// STOK BARANG
// ////////////////////////////////////////////////////////////////////////////////

func EditStokKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadEditStokKategoriBarang, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditStokBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: kredensial seller tidak valid",
		}
	}

	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
		ID:            data.IdKategoriBarang,
		IdBarangInduk: data.IdBarangInduk,
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_kategori).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_kategori == 0 {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data kategori barang tidak ditemukan",
		}
	}

	var stok_saat_ini int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("stok").Where(&models.KategoriBarang{
		ID: id_data_kategori,
	}).Limit(1).Scan(&stok_saat_ini).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	limit := int(stok_saat_ini)
	if stok_saat_ini == 0 {
		limit = 1
	}

	var id_varians []int64
	if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").
		Where(&models.VarianBarang{
			IdBarangInduk: data.IdBarangInduk,
			IdKategori:    data.IdKategoriBarang,
			Status:        barang_enums.Ready,
		}).
		Or(&models.VarianBarang{
			IdBarangInduk: data.IdBarangInduk,
			IdKategori:    data.IdKategoriBarang,
			Status:        barang_enums.Pending,
		}).Limit(limit).Scan(&id_varians).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	// Jika stok sama -> lanjut ke kategori berikutnya
	if len(id_varians) == int(data.UpdateStok) {
		return &response.ResponseForm{
			Status:   http.StatusNotModified,
			Services: services,
			Message:  "Gagal Stok nya sama saja",
		}
	}

	if len(id_varians) > int(data.UpdateStok) {
		if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.VarianBarang{}).Where("id IN ?", id_varians[data.UpdateStok:]).Delete(&models.VarianBarang{}).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: data.IdKategoriBarang,
			}).Updates(&models.KategoriBarang{
				Stok: int32(data.UpdateStok),
			}).Error; err != nil {
				return err
			}

			return nil
		}); err != nil {
			fmt.Println(err)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

	}

	if len(id_varians) < int(data.UpdateStok) {
		var sku string = ""
		if err := db.Read.Model(&models.KategoriBarang{}).Select("sku").Where(&models.KategoriBarang{
			ID: id_data_kategori,
		}).Limit(1).Scan(&sku).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
		var buat_varian_baru []models.VarianBarang
		for js := 0; js < int(data.UpdateStok)-len(id_varians); js++ {
			buat_varian_baru = append(buat_varian_baru, models.VarianBarang{
				IdBarangInduk: data.IdBarangInduk,
				IdKategori:    data.IdKategoriBarang,
				Sku:           sku,
				Status:        barang_enums.Ready,
			})
		}
		if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.CreateInBatches(&buat_varian_baru, len(buat_varian_baru)).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: data.IdKategoriBarang,
			}).Updates(&models.KategoriBarang{
				Stok: int32(len(id_varians) + len(buat_varian_baru)),
			}).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
	}

	go func(IdKb int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKategoriBarangupdated models.KategoriBarang
		if err := Read.WithContext(ctx).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangupdated).Error; err != nil {
			fmt.Println("Gagal mengambil data kategori barang")
			return
		}

		updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangupdated).SetTableName(dataKategoriBarangupdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedDataKategoriBarang); err != nil {
			fmt.Println("Gagal publish updated kategori barang stok ke message broker")
		}
	}(data.IdKategoriBarang, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Proses update stok sedang berjalan",
	}
}

func DownStokBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadDownBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "DownStokBarangInduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_barang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       data.IdBarangInduk,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_barang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data Barang Induk Tidak Valid",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VarianBarang{}).Where("id_barang_induk = ? AND status IN ?", data.IdBarangInduk, [3]string{"Pending", "Ready", "Terjual"}).Updates(&models.VarianBarang{
			Status: "Down",
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			IdBarangInduk: data.IdBarangInduk,
			SellerID:      data.IdentitasSeller.IdSeller,
		}).Update("stok", 0).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		fmt.Println(err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdBi int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		_, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)

		var updatesDownKategori []models.KategoriBarang = make([]models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Take(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated down")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(dk.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedDataKategoriBarang); err != nil {
				fmt.Println("Gagal publish updated kategori barang stok ke message broker")
			}
		}
	}(int64(data.IdBarangInduk), db.Read, cud_publisher)

	log.Printf("[INFO] Semua stok barang induk ID %d berhasil di-down-kan oleh seller ID %d", data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func DownKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadDownKategoriBarang, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "DownKategoriBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KategoriBarang{}).Select("id").Where(&models.KategoriBarang{
		ID:            data.IdKategoriBarang,
		IdBarangInduk: data.IdBarangInduk,
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_kategori).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_kategori == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data kategori barang tidak valid",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VarianBarang{}).Where("id_kategori = ? AND status IN ?", data.IdKategoriBarang, [3]string{"Pending", "Ready", "Terjual"}).Updates(&models.VarianBarang{Status: "Down"}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: data.IdKategoriBarang,
		}).Update("stok", 0).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdKb int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKategoriBarangUpdated models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName(dataKategoriBarangUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update kategori barang Id: %v ke message broker", IdKb)
		}
	}(data.IdKategoriBarang, db.Read, cud_publisher)

	log.Printf("[INFO] Semua stok kategori ID %d pada barang induk ID %d berhasil di-down-kan oleh seller ID %d", data.IdKategoriBarang, data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditRekeningBarangInduk(ctx context.Context, data PayloadEditRekeningBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditRekeningBarangInduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_rekening int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.RekeningSeller{}).Select("id").Where(&models.RekeningSeller{
		ID:       data.IdRekeningSeller,
		IDSeller: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_rekening).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang dibuk coba lagi lain waktu",
		}
	}

	if id_data_rekening == 0 {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal data rekening tidak valid",
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       data.IdBarangInduk,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang_induk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_data_barang_induk == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Barang tidak ditemukan",
		}
	}

	if err_kategori := db.Write.WithContext(ctx).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
		IdBarangInduk: data.IdBarangInduk,
		SellerID:      data.IdentitasSeller.IdSeller,
	}).Update("id_rekening", data.IdRekeningSeller).Error; err_kategori != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdBi int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		_, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)

		var updatesDownKategori []models.KategoriBarang = make([]models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Take(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated rekening")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(dk.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedDataKategoriBarang); err != nil {
				fmt.Println("Gagal publish updated kategori barang rekening ke message broker")
			}
		}
	}(int64(data.IdBarangInduk), db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditAlamatGudangBarangInduk(ctx context.Context, data PayloadEditAlamatBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahAlamatGudangBarangInduk"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial seller tidak valid untuk ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial seller tidak valid.",
		}
	}

	var id_data_barang_induk int64 = 0

	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id").Where(&models.BarangInduk{
		ID:       data.IdBarangInduk,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_data_barang_induk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Server sedang sibuk coba lagi nanti",
		}
	}

	if id_data_barang_induk == 0 {
		log.Printf("[WARN] Barang induk tidak ditemukan untuk edit alamat gudang. IdBarangInduk=%d, IdSeller=%d", data.IdBarangInduk, data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, barang induk tidak ditemukan atau kredensial seller tidak valid.",
		}
	}

	var id_alamat_gudang int64 = 0

	if errCheck := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Select("id").
		Where(&models.AlamatGudang{
			ID:       data.IdAlamatGudang,
			IDSeller: data.IdentitasSeller.IdSeller,
		}).Limit(1).Scan(&id_alamat_gudang).Error; errCheck != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi nanti.",
		}
	}

	if id_alamat_gudang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial alamat tidak valid.",
		}
	}

	if err_edit := db.Write.WithContext(ctx).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
		IdBarangInduk: int32(id_data_barang_induk),
	}).Update("id_alamat_gudang", data.IdAlamatGudang).Error; err_edit != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	go func(IdBi int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		_, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)

		var updatesDownKategori []models.KategoriBarang = make([]models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Take(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated alamat gudang")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(dk.TableName())
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedDataKategoriBarang); err != nil {
				fmt.Println("Gagal publish updated kategori barang alamat gudang ke message broker")
			}
		}
	}(int64(data.IdBarangInduk), db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil diubah.",
	}
}

func EditAlamatGudangBarangKategori(ctx context.Context, data PayloadEditAlamatBarangKategori, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahAlamatGudangBarangKategori"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)

	if !status {
		log.Printf("[WARN] Kredensial seller tidak valid untuk ID %d", data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial seller tidak valid.",
		}
	}

	var id_barang_kategori int64 = 0
	if err := db.Read.Model(models.KategoriBarang{}).Select("id").Where(models.KategoriBarang{
		ID:       data.IdKategoriBarang,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Scan(&id_barang_kategori).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_barang_kategori == 0 {
		log.Printf("[WARN] Barang induk tidak ditemukan untuk edit alamat gudang kategori. IdBarangInduk=%d, IdSeller=%d", data.IdBarangInduk, data.IdentitasSeller.IdSeller)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, barang induk tidak ditemukan atau kredensial seller tidak valid.",
		}
	}

	var id_data_alamat_gudang int64 = 0

	if errCheck := db.Read.Model(&models.AlamatGudang{}).Select("id").
		Where(&models.AlamatGudang{
			ID:       data.IdAlamatGudang,
			IDSeller: data.IdentitasSeller.IdSeller,
		}).Limit(1).Scan(&id_data_alamat_gudang).Error; errCheck != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi nanti.",
		}
	}

	if id_data_alamat_gudang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, kredensial alamat tidak valid.",
		}
	}

	if err_edit := db.Write.Model(models.KategoriBarang{}).Where(models.KategoriBarang{
		IdBarangInduk: data.IdBarangInduk,
		ID:            data.IdKategoriBarang,
	}).Update("id_alamat_gudang", data.IdAlamatGudang).Error; err_edit != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	go func(IdKb int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKategoriBarangUpdated models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName(dataKategoriBarangUpdated.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update kategori barang Id: %v ke message broker", IdKb)
		}
	}(data.IdKategoriBarang, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil diubah.",
	}
}

func MasukanKomentarBarang(ctx context.Context, data PayloadMasukanKomentarBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "TambahKomentarBarang"
	is_seller := false
	var id_seller_take int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id_seller").Where(&models.BarangInduk{
		ID: data.IdBarangInduk,
	}).Limit(1).Scan(&id_seller_take).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Barang Tidak Ada",
		}
	}

	if id_seller_take == int64(data.IdentitasSeller.IdSeller) {
		is_seller = true
	}

	NewKomentar := models.Komentar{
		IdBarangInduk: data.IdBarangInduk,
		IdEntity:      int64(data.IdentitasSeller.IdSeller),
		JenisEntity:   entity_enums.Seller,
		Komentar:      data.Komentar,
		IsSeller:      is_seller,
	}
	if err := db.Write.WithContext(ctx).Create(&NewKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Memposting Komentar",
		}
	}

	go func(K models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(K.IdBarangInduk),
		}

		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: K.ID,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdKomentar.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold komentar")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal increment total komentar barang induk ke threshold barang induk")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar baru barang induk ke message broker")
		}

	}(NewKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditKomentarBarang(ctx context.Context, data PayloadEditKomentarBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditKomentarBarang"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	var id_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.Komentar{}).Select("id").Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Limit(1).Scan(&id_komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Update("komentar", data.Komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Mengedit Komentar",
		}
	}

	go func(idKomen int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarData := models.Komentar{}
		if err := Read.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
			ID: idKomen,
		}).Limit(1).Take(&komentarData); err != nil {
			return
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		newUpdateKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(komentarData).SetTableName(komentarData.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newUpdateKomentarPublish); err != nil {
			fmt.Println("Gagal publish update komentar barang ke message broker")
		}

	}(id_komentar, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusKomentarBarang(ctx context.Context, data PayloadHapusKomentarBarangInduk, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusKomentarBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var Komentar models.Komentar
	if err := db.Read.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Limit(1).Scan(&Komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if Komentar.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.Komentar{}).Where(&models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Delete(&models.Komentar{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Menghapus Komentar",
		}
	}

	go func(K models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			ID: int64(K.IdBarangInduk),
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal decr komentar barang induk ke threshold barang induk")
		}

		newDeleteKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newDeleteKomentarPublish); err != nil {
			fmt.Println("Gagal publish delete komentar ke message broker")
		}

	}(Komentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func MasukanChildKomentar(ctx context.Context, data PayloadMasukanChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MasukanChildKomentar"
	is_seller := false

	var id_seller_take int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id_seller").Where(&models.BarangInduk{
		ID: data.IdBarangInduk,
	}).Limit(1).Take(&id_seller_take).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Barang Tidak Ada",
		}
	}

	if id_seller_take == int64(data.IdentitasSeller.IdSeller) {
		is_seller = true
	}

	newKomentar := models.KomentarChild{
		IdKomentar:  data.IdKomentarBarang,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
		IsiKomentar: data.Komentar,
		IsSeller:    is_seller,
	}
	if err := db.Write.WithContext(ctx).Create(&newKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Mengunggah Komentar",
		}
	}

	go func(Kc models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar reply ke message broker")
		}
	}(newKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func MentionChildKomentar(ctx context.Context, data PayloadMentionChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "MentionChildKomentar"

	is_seller := false

	var id_seller_take int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("id_seller").Where(&models.BarangInduk{
		ID: data.IdBarangInduk,
	}).Limit(1).Take(&id_seller_take).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Barang Tidak Ada",
		}
	}

	if id_seller_take == int64(data.IdentitasSeller.IdSeller) {
		is_seller = true
	}

	newKomentar := models.KomentarChild{
		IdKomentar:  data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
		IsiKomentar: data.Komentar,
		IsSeller:    is_seller,
		Mention:     data.UsernameMentioned,
	}

	if err := db.Write.WithContext(ctx).Create(&newKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Membalas Komentar",
		}
	}

	go func(Kc models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName())
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar reply ke message broker")
		}
	}(newKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditChildKomentar(ctx context.Context, data PayloadEditChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "EditChildKomentar"

	var id_edit_child_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.KomentarChild{}).Select("id").Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Limit(1).Scan(&id_edit_child_komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if id_edit_child_komentar == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Update("komentar", data.Komentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Mengedit Komentar",
		}
	}

	go func(IdKc int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		var dataKomentarChild models.KomentarChild
		if err := Read.WithContext(konteks).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
			ID: IdKc,
		}).Limit(1).Take(&dataKomentarChild).Error; err != nil {
			return
		}

		updateKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKomentarChild).SetTableName(dataKomentarChild.TableName())
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateKomentarChildPublish); err != nil {
			fmt.Println("Gagal publish update child komentar ke message broker")
		}
	}(id_edit_child_komentar, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusChildKomentar(ctx context.Context, data PayloadHapusChildKomentar, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "HapusChildKomentar"

	var childKomentar models.KomentarChild
	if err := db.Read.WithContext(ctx).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Limit(1).Scan(&childKomentar).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if childKomentar.ID == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal komentar tidak ditemukan",
		}
	}

	if err := db.Write.WithContext(ctx).Model(&models.KomentarChild{}).Where(&models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Delete(&models.KomentarChild{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Menghapus Komentar",
		}
	}

	go func(Kc models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarThreshold := sot_threshold.KomentarThreshold{
			ID: Kc.IdKomentar,
		}

		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()

		if err := komentarThreshold.Decrement(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal decrement komentar child ke threshold komentar")
		}

		deleteKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName())
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteKomentarChildPublish); err != nil {
			fmt.Println("Gagal publish delete komentar child ke message broker")
		}
	}(childKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
