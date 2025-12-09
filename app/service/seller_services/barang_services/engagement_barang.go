package seller_barang_service

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	barang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/barang"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/enums/seller_dedication"
	"github.com/anan112pcmec/Burung-backend-1/app/database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Masukan Barang Seller
// Berfungsi untuk melayani seller yang hendak memasukan barang nya ke sistem burung
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadMasukanBarangInduk) *response.ResponseForm {
	services := "MasukanBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

func EditBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadEditBarangInduk) *response.ResponseForm {
	services := "EditBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

func HapusBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadHapusBarangInduk) *response.ResponseForm {
	services := "HapusBarang"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
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

	// Cek apakah masih ada varian dalam transaksi (status: Dipesan/Diproses)
	var id_varian_dalam_transaksi int64 = 0
	if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").
		Where("id_barang_induk = ? AND status IN ?", data.IdBarangInduk, []string{"Dipesan", "Diproses"}).
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

func TambahKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadTambahKategori) *response.ResponseForm {
	services := "TambahKategoriBarang"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

func EditKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadEditKategori) *response.ResponseForm {
	services := "EditKategoriBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

func HapusKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadHapusKategori) *response.ResponseForm {
	services := "HapusKategoriBarang"

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kategori barang berhasil dihapus (soft delete manual).",
	}
}

// ////////////////////////////////////////////////////////////////////////////////
// STOK BARANG
// ////////////////////////////////////////////////////////////////////////////////

func EditStokKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadEditStokKategoriBarang) *response.ResponseForm {
	services := "EditStokBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Proses update stok sedang berjalan",
	}
}

func DownStokBarangInduk(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadDownBarangInduk) *response.ResponseForm {
	services := "DownStokBarangInduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

	log.Printf("[INFO] Semua stok barang induk ID %d berhasil di-down-kan oleh seller ID %d", data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func DownKategoriBarang(ctx context.Context, db *config.InternalDBReadWriteSystem, data PayloadDownKategoriBarang) *response.ResponseForm {
	services := "DownKategoriBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

	log.Printf("[INFO] Semua stok kategori ID %d pada barang induk ID %d berhasil di-down-kan oleh seller ID %d", data.IdKategoriBarang, data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditRekeningBarangInduk(ctx context.Context, data PayloadEditRekeningBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditRekeningBarangInduk"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditAlamatGudangBarangInduk(ctx context.Context, data PayloadEditAlamatBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahAlamatGudangBarangInduk"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read)

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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil diubah.",
	}
}

func EditAlamatGudangBarangKategori(ctx context.Context, data PayloadEditAlamatBarangKategori, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "TambahAlamatGudangBarangKategori"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read)

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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil diubah.",
	}
}

func MasukanKomentarBarang(ctx context.Context, data PayloadMasukanKomentarBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
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

	if err := db.Write.WithContext(ctx).Create(&models.Komentar{
		IdBarangInduk: data.IdBarangInduk,
		IdEntity:      int64(data.IdentitasSeller.IdSeller),
		JenisEntity:   entity_enums.Seller,
		Komentar:      data.Komentar,
		IsSeller:      is_seller,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Memposting Komentar",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditKomentarBarang(ctx context.Context, data PayloadEditKomentarBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditKomentarBarang"

	_, status := data.IdentitasSeller.Validating(ctx, db.Read)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak valid",
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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusKomentarBarang(ctx context.Context, data PayloadHapusKomentarBarangInduk, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusKomentarBarang"

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func MasukanChildKomentar(ctx context.Context, data PayloadMasukanChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
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
	if err := db.Write.WithContext(ctx).Create(&models.KomentarChild{
		IdKomentar:  data.IdKomentarBarang,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
		IsiKomentar: data.Komentar,
		IsSeller:    is_seller,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Mengunggah Komentar",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func MentionChildKomentar(ctx context.Context, data PayloadMentionChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
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

	if err := db.Write.WithContext(ctx).Create(&models.KomentarChild{
		IdKomentar:  data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
		IsiKomentar: data.Komentar,
		IsSeller:    is_seller,
		Mention:     data.UsernameMentioned,
	}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Membalas Komentar",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditChildKomentar(ctx context.Context, data PayloadEditChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "EditChildKomentar"

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

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func HapusChildKomentar(ctx context.Context, data PayloadHapusChildKomentar, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "HapusChildKomentar"
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
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}
