package seller_barang_service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	barang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/barang"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_alamat_gudang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/alamat_gudang"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_komentar "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/komentar"
	stsk_rekening_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/rekening_seller"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Masukan Barang Seller
// Berfungsi untuk melayani seller yang hendak memasukan barang nya ke sistem burung
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func MasukanBarangInduk(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadMasukanBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "MasukanBarangInduk"

	if len(data.BarangInduk.NamaBarang) < 5 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Nama Barang harus lebih dari 5 karakter",
		}
	}

	if !helper.SellerDedicationValidation(data.BarangInduk.JenisBarang) {
		return &response.ResponseForm{
			Status:   http.StatusNotAcceptable,
			Services: services,
			Message:  "Gagal jenis barang tidak dikenal",
		}
	}

	if data.IdAlamatGudang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdAlamatGudang tak boleh lebih kecil atau sama dengan 0",
		}
	}

	if data.IdRekening <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdRekening tak boleh lebih kecil atau sama dengan 0",
		}
	}

	for _, k := range data.KategoriBarang {
		if len(k.Nama) < 5 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, Nama Kategori tak boleh kurang 5 karakter",
			}
		}

		if !helper.HexaColorValidation(k.Warna) {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, warna bukan merupakan hexa yang valid",
			}
		}

		if k.Harga < 200 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, harga lebih kecil dari 200 perak",
			}
		}

		if k.BeratGram <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, BeratGram Tak boleh 0",
			}
		}

		if k.DimensiPanjang <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, Panjang Tak boleh 0",
			}
		}

		if k.DimensiLebar <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, Lebar Tak boleh 0",
			}
		}

		if strings.TrimSpace(k.Sku) == "" {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Sku tak boleh kosong",
			}
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal memasukkan barang karena kredensial seller tidak valid",
		}
	}

	data.BarangInduk.SellerID = data.IdentitasSeller.IdSeller

	var id_data_barang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).
		Select("id").
		Where(&sot_models.BarangInduk{
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
	for i, _ := range data.KategoriBarang {
		if data.KategoriBarang[i].IsOriginal && data.BarangInduk.OriginalKategori == int64(i+1) {
			harga_original = int64(data.KategoriBarang[i].Harga)
			break
		}
	}

	if harga_original <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Harga original tidak boleh 0",
		}
	}

	barang_induk := sot_models.BarangInduk{
		ID:             data.BarangInduk.ID,
		SellerID:       data.IdentitasSeller.IdSeller,
		NamaBarang:     data.BarangInduk.NamaBarang,
		JenisBarang:    data.BarangInduk.JenisBarang,
		Deskripsi:      data.BarangInduk.Deskripsi,
		HargaKategoris: int32(harga_original),
	}

	var banyakKategoriBarang int = len(data.KategoriBarang)

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

		if err := tx.WithContext(ctx).CreateInBatches(&data.KategoriBarang, banyakKategoriBarang).Error; err != nil {
			return err
		}

		var id_origin_kategori int64 = 0
		if err := tx.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
			IdBarangInduk: int32(barang_induk.ID),
			IsOriginal:    true,
		}).Limit(1).Scan(&id_origin_kategori).Error; err != nil {
			return err
		}

		if err := tx.Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{
			ID: barang_induk.ID,
		}).Update("original_kategori", id_origin_kategori).Error; err != nil {
			return err
		}

		var id_kategoris []int64
		if err := tx.Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
			IdBarangInduk: barang_induk.ID,
		}).Limit(banyakKategoriBarang).Scan(&id_kategoris).Error; err != nil {
			return err
		}

		var totalBatchVarian int64 = 0
		var varian_barang []sot_models.VarianBarang
		for i, _ := range id_kategoris {
			var kategori sot_models.KategoriBarang
			if err := tx.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
				ID: id_kategoris[i],
			}).Limit(1).Find(&kategori).Error; err != nil {
				return err
			}

			for i := 0; i < int(kategori.Stok); i++ {
				varian_barang = append(varian_barang, sot_models.VarianBarang{
					IdBarangInduk: barang_induk.ID,
					IdKategori:    kategori.ID,
					Sku:           kategori.Sku,
					Status:        barang_enums.Down,
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
	} else {
		go func(Bi sot_models.BarangInduk, Kb []sot_models.KategoriBarang, Trh *environment.InternalDBReadWriteSystem, publisher *mb_cud_publisher.Publisher) {
			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(Bi.SellerID),
			}

			thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
				IdBarangInduk: int64(Bi.ID),
			}

			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			fmt.Println("NIH KB NGENTOT", len(Kb))

			if err := thresholdSeller.CustomIncrement(konteks, Trh.Write, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_seller.BarangInduk,
					Count:     1,
				},
				{
					FieldName: stsk_seller.KategoriBarang,
					Count:     len(Kb),
				},
			}); err != nil {
				fmt.Println("Gagal increment seller threshold field barang induk dan kategori barang")
			}

			if err := thresholdBarangInduk.Inisialisasi(konteks, Trh.Write); err != nil {
				fmt.Println("Gagal inisialisasi thresholdbarang induk")
			}

			var dataBarangIndukNew sot_models.BarangInduk
			if err := Trh.Read.WithContext(konteks).Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{
				ID: Bi.ID,
			}).Limit(1).Take(&dataBarangIndukNew).Error; err != nil {
				fmt.Println("Gagal mendapatkan data barang induk", err)
			}

			createNewBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataBarangIndukNew).SetTableName(dataBarangIndukNew.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createNewBarangIndukPublish); err != nil {
				fmt.Println("Gagal publish create new barang induk ke message broker")
			}

			var total_varian int = 0

			for _, k := range Kb {
				total_varian += int(k.Stok)

				thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{IdKategoriBarang: k.ID}

				thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{IdAlamatGudang: k.IDAlamat}

				thresholdRekeningSeller := sot_threshold.RekeningSellerThreshold{IdRekeningSeller: k.IDRekening}
				if err := thresholdKategoriBarang.Inisialisasi(konteks, Trh.Write); err != nil {
					fmt.Println("Gagal membuat threshold untuk kategori ber Id: ", k.ID, " Dan Bernama: ", k.Nama)
				}

				if err := thresholdKategoriBarang.CustomIncrement(konteks, Trh.Write, []sot_threshold.CustomCounter{
					{
						FieldName: stsk_kategori_barang.Down,
						Count:     int(k.Stok),
					},
				}); err != nil {
					fmt.Println("Gagal increment varian barang di threshold kategori barang")
				}

				thresholdAlamatGudang.IdAlamatGudang = k.IDAlamat
				if err := thresholdAlamatGudang.Increment(konteks, Trh.Write, stsk_alamat_gudang.KategoriBarang); err != nil {
					fmt.Println("Gagal, increment threshold alamat gudang kategori barang")
				}

				thresholdRekeningSeller.IdRekeningSeller = k.IDRekening
				if err := thresholdRekeningSeller.Increment(konteks, Trh.Write, stsk_rekening_seller.KategoriBarang); err != nil {
					fmt.Println("Gagal, increment threshold rekening seller field kategori barang")
				}

				createNewKategoriBarangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(k).SetTableName(k.TableName()).SetRole(mb_cud_seeders.Seller)
				if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createNewKategoriBarangPublish); err != nil {
					fmt.Println("Gagal publish create new kategori ke message broker")
				}
			}

			if err := thresholdBarangInduk.CustomIncrement(konteks, Trh.Write, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_baranginduk.KategoriBarang,
					Count:     len(Kb),
				},
				{
					FieldName: stsk_baranginduk.VarianBarang,
					Count:     total_varian,
				},
			}); err != nil {
				fmt.Println("Gagal increment threshold barang induk")
			}

		}(barang_induk, data.KategoriBarang, db, cud_publisher)
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Barang berhasil ditambahkan.",
	}
} // Tuned

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Edit Barang Induk
// Berfungsi untuk seller dalam melakukan edit atau pembaruan informasi seputar barang induknya
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditBarangInduk(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadEditBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditBarangInduk"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk lebih kecil dari 0",
		}
	}

	if len(data.NamaBarang) < 5 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, nama barang tak boleh kurang dari 5 karakter",
		}
	}

	if !helper.SellerDedicationValidation(data.JenisBarang) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Jenis Barang Tak Valid",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial Seller Tidak Valid",
		}
	}

	var id_data_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{
		ID: int32(data.IdBarangInduk),
	}).Updates(&sot_models.BarangInduk{
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
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var updatedBarangInduk sot_models.BarangInduk
		if err := Read.WithContext(konteks).Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{
			ID: int32(IdBarangInduk),
		}).Limit(1).Take(&updatedBarangInduk).Error; err != nil {
			fmt.Println("Gagal mengambil data terbaru barang induk")
			return
		}

		updatedBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(updatedBarangInduk).SetTableName(updatedBarangInduk.TableName()).SetRole(mb_cud_seeders.Seller)
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

func HapusBarangInduk(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadHapusBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusBarangInduk"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
		}
	}

	var dataBarangInduk sot_models.BarangInduk
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").
		Where("id_barang_induk = ? AND status IN ?", dataBarangInduk.ID, []string{barang_enums.Dipesan, barang_enums.Diproses, barang_enums.Ready}).
		Limit(1).Scan(&id_varian_dalam_transaksi).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada database",
		}
	}

	// Ã¢Å“â€¦ 3. Early exit jika masih ada transaksi aktif
	if id_varian_dalam_transaksi > 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Masih ada varian dalam transaksi",
		}
	}

	thresholdBarangInduk := sot_threshold.BarangIndukThreshold{ID: int64(dataBarangInduk.ID)}
	err_tkb, totalKategoriBarang := thresholdBarangInduk.GetKolomCount(ctx, db.Read, stsk_baranginduk.KategoriBarang)
	if err_tkb != nil {
		fmt.Println("gagal mengambil total kategori barang", err_tkb)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi kesalahan pada database",
		}
	}

	fmt.Println("Total barang ngentotttt: ", totalKategoriBarang)

	var dataKategoriBarang []sot_models.KategoriBarang = make([]sot_models.KategoriBarang, 0, totalKategoriBarang)
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
		IdBarangInduk: dataBarangInduk.ID,
	}).Limit(totalKategoriBarang).Find(&dataKategoriBarang).Error; err != nil {
		fmt.Println("Gagal mengambil data kategori barang", err)
	}

	// Jalankan proses penghapusan dalam goroutine (asynchronous)
	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// Ã°Å¸â€Â¸ Hapus varian (permanent delete)
		if err := tx.Unscoped().Model(&sot_models.VarianBarang{}).Where(&sot_models.VarianBarang{IdBarangInduk: int32(data.IdBarangInduk)}).
			Delete(&sot_models.VarianBarang{}).Error; err != nil {
			return fmt.Errorf("hapus varian gagal: %w", err)
		}

		// Ã°Å¸â€Â¸ Hapus kategori (soft delete)
		if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{IdBarangInduk: int32(data.IdBarangInduk)}).
			Delete(&sot_models.KategoriBarang{}).Error; err != nil {
			return fmt.Errorf("hapus kategori gagal: %w", err)
		}

		// Ã°Å¸â€Â¸ Hapus barang induk (soft delete)
		if err := tx.Model(&sot_models.BarangInduk{}).Where(&sot_models.BarangInduk{ID: int32(data.IdBarangInduk)}).
			Delete(&sot_models.BarangInduk{}).Error; err != nil {
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
	} else {
		go func(DBI sot_models.BarangInduk, DBK []sot_models.KategoriBarang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(DBI.SellerID),
			}

			konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
			defer cancel()

			if err := thresholdSeller.CustomDecrement(konteks, Trh, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_seller.BarangInduk,
					Count:     1,
				},
				{
					FieldName: stsk_seller.KategoriBarang,
					Count:     len(DBK),
				},
			}); err != nil {
				fmt.Println("Gagal decrement seller threshold field barang induk dan kategori barang")
			}

			if err := Trh.WithContext(konteks).Model(&sot_threshold.BarangIndukThreshold{}).Where(&sot_threshold.BarangIndukThreshold{
				IdBarangInduk: int64(DBI.ID),
			}).Delete(&sot_threshold.BarangIndukThreshold{}).Error; err != nil {
				fmt.Println("Gagal menghapus barang_induk threshold")
			}

			deleteBarangIndukPublish := mb_cud_serializer.NewJsonPayload().SetPayload(DBI).SetTableName(DBI.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteBarangIndukPublish); err != nil {
				fmt.Printf("Gagal publish delete barang induk ber id: %v, bernama %s ke message broker", DBI.ID, DBI.NamaBarang)
			}

			for _, dbkategori := range DBK {
				if err := Trh.WithContext(konteks).Model(&sot_threshold.KategoriBarangThreshold{}).Where(&sot_threshold.KategoriBarangThreshold{
					IdKategoriBarang: dbkategori.ID,
				}).Delete(&sot_threshold.KategoriBarangThreshold{}).Error; err != nil {
					fmt.Printf("Gagal menghapus data kategori barang ber id %v dan bernama %s", dbkategori.ID, dbkategori.Nama)
				}

				thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{IdAlamatGudang: dbkategori.IDAlamat}

				thresholdRekeningSeller := sot_threshold.RekeningSellerThreshold{IdRekeningSeller: dbkategori.IDRekening}

				if err := thresholdAlamatGudang.Decrement(konteks, Trh, stsk_alamat_gudang.KategoriBarang); err != nil {
					fmt.Println("Gagal decrement threshold alamat gudang field kategori barang")
				}

				if err := thresholdRekeningSeller.Decrement(konteks, Trh, stsk_rekening_seller.KategoriBarang); err != nil {
					fmt.Println("Gagal decrement threshold rekening seller field kategori barang")
				}

				deleteKategoriBarangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dbkategori).SetTableName(dbkategori.TableName()).SetRole(mb_cud_seeders.Seller)
				if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteKategoriBarangPublish); err != nil {
					fmt.Printf("Gagal publish delete kategori barang ber id %v dan bernama %s ke message broker", dbkategori.ID, dbkategori.Nama)
				}
			}

		}(dataBarangInduk, dataKategoriBarang, db.Write, cud_publisher)
	}

	// Kembalikan respons sukses
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Barang berhasil dihapus.",
	}
} // Tuned

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Tambah Kategori Barang Induk
// Berfungsi untuk seller menambahkan kategori barang pada barang induk
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TambahKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadTambahKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahKategoriBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	for _, k := range data.KategoriBarang {
		if len(k.Nama) < 5 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, Nama Kategori tak boleh kurang 5 karakter",
			}
		}

		if !helper.HexaColorValidation(k.Warna) {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, warna bukan merupakan hexa yang valid",
			}
		}

		if k.Harga < 200 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, harga lebih kecil dari 200 perak",
			}
		}

		if k.BeratGram <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, BeratGram Tak boleh 0",
			}
		}

		if k.DimensiPanjang <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, Panjang Tak boleh 0",
			}
		}

		if k.DimensiLebar <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal, Lebar Tak boleh 0",
			}
		}

		if strings.TrimSpace(k.Sku) == "" {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Sku tak boleh kosong",
			}
		}
	}

	if data.IdAlamatGudang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdAlamatGudang Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdRekening <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdAlamatGudang Tak boleh lebih kecil dari 0",
		}
	}

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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.AlamatGudang{}).Select("id").Where(&sot_models.AlamatGudang{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.RekeningSeller{}).Select("id").Where(&sot_models.RekeningSeller{
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

	var kategori_barang []sot_models.KategoriBarang

	// 1Ã¯Â¸ÂÃ¢Æ’Â£ Loop validasi dan siapkan batch kategori
	for i := range data.KategoriBarang {
		var id_data_kategori_barang int64 = 0

		// Cek apakah kategori dengan nama yang sama sudah ada
		if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).
			Select("id").
			Where(&sot_models.KategoriBarang{
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
		kategori_barang = append(kategori_barang, sot_models.KategoriBarang{
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

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := tx.CreateInBatches(&kategori_barang, len(kategori_barang)).Error; err != nil {
			return err
		}

		var varian_barang_total []sot_models.VarianBarang
		var VarianBatch int64 = 0

		for _, kategori := range kategori_barang {
			for s := 0; s < int(kategori.Stok); s++ {
				varian_barang_total = append(varian_barang_total, sot_models.VarianBarang{
					IdBarangInduk: data.IdBarangInduk,
					IdKategori:    kategori.ID, // Ã°Å¸Â§Â  langsung pakai ID dari hasil insert batch
					Sku:           kategori.Sku,
					Status:        barang_enums.Down,
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

	go func(Kb []sot_models.KategoriBarang, IBI int64, IdSeller int32, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		var thresholdBarangInduk sot_threshold.BarangIndukThreshold = sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IBI,
		}

		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(IdSeller),
		}

		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{}
		thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{}
		thresholdRekeningSeller := sot_threshold.RekeningSellerThreshold{}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdBarangInduk.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{
			{
				FieldName: stsk_baranginduk.KategoriBarang,
				Count:     len(Kb),
			},
		}); err != nil {
			fmt.Println("gagal count up threshold barang induk untuk kategori barang", err)
		}

		if err := thresholdSeller.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{
			{
				FieldName: stsk_seller.KategoriBarang,
				Count:     len(Kb),
			},
		}); err != nil {
			fmt.Println("gagal count up threshold barang induk untuk kategori barang", err)
		}

		for _, kategoribarangdata := range Kb {
			thresholdKategoriBarang.IdKategoriBarang = kategoribarangdata.ID
			thresholdAlamatGudang.IdAlamatGudang = kategoribarangdata.IDAlamat
			thresholdRekeningSeller.IdRekeningSeller = kategoribarangdata.IDRekening

			if err := thresholdKategoriBarang.Inisialisasi(konteks, Trh); err != nil {
				fmt.Printf("Gagal membuat threshold kategori barang ber id %v, bernama %s", kategoribarangdata.ID, kategoribarangdata.Nama)
			}

			if err := thresholdKategoriBarang.CustomIncrement(konteks, Trh, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Down,
					Count:     int(kategoribarangdata.Stok),
				},
			}); err != nil {
				fmt.Println("Gagal increment kategori barang threshold field down")
			}

			if err := thresholdAlamatGudang.Increment(konteks, Trh, stsk_alamat_gudang.KategoriBarang); err != nil {
				fmt.Println("Gagal increment threshold alamat gudang field kategori barang")
			}

			if err := thresholdRekeningSeller.Increment(konteks, Trh, stsk_rekening_seller.KategoriBarang); err != nil {
				fmt.Println("Gagal increment threshold rekening sseller field kategori barang")
			}

			newKategoriBarangCreatePublish := mb_cud_serializer.NewJsonPayload().SetPayload(kategoribarangdata).SetTableName(kategoribarangdata.TableName()).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKategoriBarangCreatePublish); err != nil {
				fmt.Printf("Gagal mempublish create kategori barang ber id %v dan bernama %s ke message broker", kategoribarangdata.ID, kategoribarangdata.Nama)
			}
		}
	}(kategori_barang, int64(data.IdBarangInduk), data.IdentitasSeller.IdSeller, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kategori barang berhasil ditambahkan",
	}
} // Tuned

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Edit Kategori Barang
// Berfungsi untuk mengedit data informasi tentang kategori barang induk yang dituju
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func EditKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadEditKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditKategoriBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang Tak boleh lebih kecil dari 0",
		}
	}

	if len(data.Nama) < 5 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Nama Barang Kurang dari 5 karakter",
		}
	}

	if !helper.HexaColorValidation(data.Warna) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Format warna hexa tidak valid",
		}
	}

	if data.BeratGram <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, BeratGram Tak boleh 0",
		}
	}

	if data.DimensiPanjang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Panjang Tak boleh 0",
		}
	}

	if data.DimensiLebar <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Lebar Tak boleh 0",
		}
	}

	if strings.TrimSpace(data.Sku) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Sku tak boleh kosong",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
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
		if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: data.IdKategoriBarang,
		}).Updates(&sot_models.KategoriBarang{
			Nama:           data.Nama,
			Deskripsi:      data.Deskripsi,
			Warna:          data.Warna,
			BeratGram:      int16(data.BeratGram),
			DimensiPanjang: data.DimensiPanjang,
			DimensiLebar:   data.DimensiLebar,
			Sku:            data.Sku,
		}).Error; err != nil {
			return err
		}

		if data.Sku != "" {
			if err := tx.Model(&sot_models.VarianBarang{}).Where(&sot_models.VarianBarang{
				IdKategori: data.IdKategoriBarang,
			}).Updates(&sot_models.VarianBarang{
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
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var dataKategoriBarangUpdated sot_models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName(dataKategoriBarangUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
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
} // Tuned

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Hapus Kategori Barang Induk
// Berfungsi untuk menghapus kategori barang induk yang ada
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func HapusKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadHapusKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusKategoriBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang Tak boleh lebih kecil dari 0",
		}
	}

	// Validasi kredensial seller
	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
		}
	}

	var data_kategori sot_models.KategoriBarang
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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
	if errStock := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").
		Where("id_barang_induk = ? AND id_kategori = ? AND status IN ?", data.IdBarangInduk, data.IdKategoriBarang, []string{barang_enums.Dipesan, barang_enums.Diproses, barang_enums.Ready}).
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
		if err := tx.Model(&sot_models.VarianBarang{}).Where(&sot_models.VarianBarang{
			IdKategori: data.IdKategoriBarang,
		}).Delete(&sot_models.KategoriBarang{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: data.IdKategoriBarang,
		}).Delete(&sot_models.KategoriBarang{}).Error; err != nil {
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

	go func(Kb sot_models.KategoriBarang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdSeller := sot_threshold.SellerThreshold{
			IdSeller: int64(Kb.SellerID),
		}

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(Kb.IdBarangInduk),
		}

		thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
			IdAlamatGudang: Kb.IDAlamat,
		}

		thresholdRekeningSeller := sot_threshold.RekeningSellerThreshold{
			IdRekeningSeller: Kb.IDRekening,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdSeller.Decrement(konteks, Trh, stsk_seller.KategoriBarang); err != nil {
			fmt.Println("Gagal decrement seller threshold field kategori barang")
		}

		if err := thresholdBarangInduk.Decrement(konteks, Trh, stsk_baranginduk.KategoriBarang); err != nil {
			fmt.Println("Gagal decrement kategori barang dalam ts baranginduk")
		}

		if err := thresholdAlamatGudang.Decrement(konteks, Trh, stsk_alamat_gudang.KategoriBarang); err != nil {
			fmt.Println("Gagal decrement alamat gudang threshold field kategori barang")
		}

		if err := thresholdRekeningSeller.Decrement(konteks, Trh, stsk_rekening_seller.KategoriBarang); err != nil {
			fmt.Println("Gagal decrement rekening seller threshold field kategori barang")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.KategoriBarangThreshold{}).Where(&sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: Kb.ID,
		}).Delete(&sot_threshold.KategoriBarangThreshold{}).Error; err != nil {
			fmt.Printf("Gagal menghapus threshold kategori barang ber ID:%v", Kb.ID)
		}

		kategoriBarangDeletePublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kb).SetTableName(Kb.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangDeletePublish); err != nil {
			fmt.Printf("Gagal publish hapus kategori barang ke message broker")
		}

	}(data_kategori, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Kategori barang berhasil dihapus (soft delete manual).",
	}
} // Tuned

func UbahHargaKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadUbahHargaKategori, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "UbahHargaKategoriBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang Tak boleh lebih kecil dari 0",
		}
	}

	if data.HargaBarangBaru < 200 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, Harga tak boleh lebih kecil dari 200 perak",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: Kredensial seller tidak valid",
		}
	}

	var id_data_kategori int64
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
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

	var exist_varian_transaksi int64 = 0
	if errStock := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").
		Where("id_barang_induk = ? AND id_kategori = ? AND status IN ?", data.IdBarangInduk, data.IdKategoriBarang, []string{barang_enums.Dipesan, barang_enums.Diproses, barang_enums.Ready}).
		Limit(1).Scan(&exist_varian_transaksi).Error; errStock != nil {
	}

	if exist_varian_transaksi != 0 {
		return &response.ResponseForm{
			Status:   http.StatusConflict,
			Services: services,
			Message:  "Gagal Kategori ini masih ada dalam transaksi yang belum selesai, Down kan dulu",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: data.IdKategoriBarang,
		}).Updates(&sot_models.KategoriBarang{
			Harga: int32(data.HargaBarangBaru),
		}).Error; err != nil {
			return err
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
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var dataKategoriBarangUpdated sot_models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang untuk publish")
			return
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName(services).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update harga kategori barang Id: %v ke message broker\n", IdKb)
		}
	}(data.IdKategoriBarang, db.Read, cud_publisher)

	log.Printf("[INFO] Harga kategori barang berhasil diubah pada barang induk ID %d oleh seller ID %d (kondisi aman / down)", data.IdBarangInduk, data.IdentitasSeller.IdSeller)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Harga kategori barang berhasil diubah.",
	}
} // Tuned

// ////////////////////////////////////////////////////////////////////////////////
// STOK BARANG
// ////////////////////////////////////////////////////////////////////////////////

func EditStokKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadEditStokKategoriBarang, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditStokBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang Tak boleh lebih kecil dari 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal: kredensial seller tidak valid",
		}
	}

	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("stok").Where(&sot_models.KategoriBarang{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").
		Where(&sot_models.VarianBarang{
			IdKategori: data.IdKategoriBarang,
			Status:     barang_enums.Down,
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

	var tambah bool
	var sebanyak int

	if len(id_varians) > int(data.UpdateStok) {
		tambah = false
		sebanyak = len(id_varians) - int(data.UpdateStok)
		if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&sot_models.VarianBarang{}).Where("id IN ?", id_varians[data.UpdateStok:]).Delete(&sot_models.VarianBarang{}).Error; err != nil {
				return err
			}

			if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
				ID: data.IdKategoriBarang,
			}).Updates(&sot_models.KategoriBarang{
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
		tambah = true
		sebanyak = int(data.UpdateStok) - len(id_varians)
		var sku string = ""
		if err := db.Read.Model(&sot_models.KategoriBarang{}).Select("sku").Where(&sot_models.KategoriBarang{
			ID: id_data_kategori,
		}).Limit(1).Scan(&sku).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
		var buat_varian_baru []sot_models.VarianBarang
		for js := 0; js < int(data.UpdateStok)-len(id_varians); js++ {
			buat_varian_baru = append(buat_varian_baru, sot_models.VarianBarang{
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

			if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
				ID: data.IdKategoriBarang,
			}).Updates(&sot_models.KategoriBarang{
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

	go func(IdKb int64, t bool, s int, Write *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: IdKb,
		}
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if t {
			thresholdKategoriBarang.CustomIncrement(konteks, Write, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Down,
					Count:     s,
				},
			})
		} else {
			thresholdKategoriBarang.CustomDecrement(konteks, Write, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Down,
					Count:     s,
				},
			})
		}

		var dataKategoriBarangupdated sot_models.KategoriBarang
		if err := Write.WithContext(konteks).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangupdated).Error; err != nil {
			fmt.Println("Gagal mengambil data kategori barang")
			return
		}

		updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangupdated).SetTableName(dataKategoriBarangupdated.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedDataKategoriBarang); err != nil {
			fmt.Println("Gagal publish updated kategori barang stok ke message broker")
		}
	}(data.IdKategoriBarang, tambah, sebanyak, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Proses update stok sedang berjalan",
	}
} // Tuned

func DownStokBarangInduk(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadDownBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "DownStokBarangInduk"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_barang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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
		if err := tx.Model(&sot_models.VarianBarang{}).Where("id_barang_induk = ? AND status IN ?", data.IdBarangInduk, [1]string{barang_enums.Ready}).Updates(&sot_models.VarianBarang{
			Status: barang_enums.Down,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		err_tk, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)
		if err_tk != nil {
			fmt.Println("gagal mengambil total kategori barang")
		}

		var thresholdBarangIndukFieldKategoriBarang int64 = 0
		if err := Read.WithContext(ctx).Model(&sot_threshold.BarangIndukThreshold{}).Select(stsk_baranginduk.KategoriBarang).Where(&sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(IdBi),
		}).Limit(1).Take(&thresholdBarangIndukFieldKategoriBarang); err != nil {
			fmt.Println("Gagal mengambil threshold barang induk")
		}

		var id_kategori_barang []int64

		if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
			IdBarangInduk: data.IdBarangInduk,
		}).Limit(func(ibi int32) int {
			if ibi != 0 {
				return int(ibi)
			}
			return 1
		}(int32(IdBi))).Find(&id_kategori_barang).Error; err != nil {
			fmt.Println("Gagal mengambil id bkategori barang")
		}

		for _, ik := range id_kategori_barang {
			kategoriBarangThreshold := sot_threshold.KategoriBarangThreshold{IdKategoriBarang: ik}

			ecr, countReady := kategoriBarangThreshold.GetKolomCount(konteks, Read, stsk_kategori_barang.Ready)
			if ecr != nil {
				fmt.Println("Gagal mengambil count")
			}

			if err := kategoriBarangThreshold.CustomDecrement(ctx, Read, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Ready,
					Count:     int(countReady),
				},
			}); err != nil {
				fmt.Println("Gagal increment kategori ready")
			}

			if err := kategoriBarangThreshold.CustomIncrement(ctx, Read, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Down,
					Count:     int(countReady),
				},
			}); err != nil {
				fmt.Println("Gagal decrement kategori down")
			}
		}

		var updatesDownKategori []sot_models.KategoriBarang = make([]sot_models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&sot_models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Find(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated down")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(services).SetRole(mb_cud_seeders.Seller)
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
} // Tuned

func DownKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadDownKategoriBarang, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "DownKategoriBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang Tak boleh lebih kecil dari 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
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

	var limit_varian int64 = 0

	if data.Jumlah == 0 {
		if err := db.Read.WithContext(ctx).Model(&sot_threshold.KategoriBarangThreshold{}).Select(stsk_kategori_barang.Ready).Where(&sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: data.IdKategoriBarang,
		}).Limit(1).Take(&limit_varian).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
	} else {
		limit_varian = data.Jumlah
	}

	var id_varian_barang []int64

	if err := db.Read.WithContext(ctx).
		Model(&sot_models.VarianBarang{}).
		Select("id").
		Where(&sot_models.VarianBarang{
			IdKategori: data.IdKategoriBarang,
			Status:     barang_enums.Ready,
		}).
		Limit(int(limit_varian)).
		Find(&id_varian_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		const stok string = "stok"

		var exp map[string]interface{} = map[string]interface{}{stok: gorm.Expr(fmt.Sprintf("%s - ?", stok), len(id_varian_barang))}
		if err := tx.Model(&sot_models.KategoriBarang{}).
			Where(&sot_models.KategoriBarang{ID: data.IdKategoriBarang}).
			Updates(exp).Error; err != nil {
			return err
		}

		if len(id_varian_barang) > 0 {
			if err := tx.Model(&sot_models.VarianBarang{}).
				Where("id IN ?", id_varian_barang).
				Update("status", barang_enums.Down).Error; err != nil {
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

	go func(IdKb int64, Downed int, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		kategoriBarangThreshold := sot_threshold.KategoriBarangThreshold{IdKategoriBarang: IdKb}
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := kategoriBarangThreshold.CustomDecrement(ctx, Read, []sot_threshold.CustomCounter{
			{
				FieldName: stsk_kategori_barang.Ready,
				Count:     Downed,
			},
		}); err != nil {
			fmt.Println("Gagal increment kategori ready")
		}

		if err := kategoriBarangThreshold.CustomIncrement(ctx, Read, []sot_threshold.CustomCounter{
			{
				FieldName: stsk_kategori_barang.Down,
				Count:     Downed,
			},
		}); err != nil {
			fmt.Println("Gagal decrement kategori down")
		}

		var dataKategoriBarangUpdated sot_models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName("DownKategoriBarang").SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update kategori barang Id: %v ke message broker", IdKb)
		}
	}(data.IdKategoriBarang, len(id_varian_barang), db.Read, cud_publisher)

	log.Printf("[INFO] Semua stok kategori ID %d pada barang induk ID %d berhasil di-down-kan oleh seller ID %d", data.IdKategoriBarang, data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} // tuned

func UpBarangInduk(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadUpBarangInduk, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "UpBarangInduk"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal id barang induk tak boleh lebih kecil atau sama dengan 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_barang_induk int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
		ID:       data.IdBarangInduk,
		SellerID: data.IdentitasSeller.IdSeller,
	}).Limit(1).Take(&id_barang_induk).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal barang induk tidak ditemukan",
		}
	}

	if id_barang_induk == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal barang induk tidak ditemukan",
		}
	}

	var thresholdBarangIndukFieldKategoriBarang int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_threshold.BarangIndukThreshold{}).Select(stsk_baranginduk.KategoriBarang).Where(&sot_threshold.BarangIndukThreshold{
		IdBarangInduk: int64(data.IdBarangInduk),
	}).Limit(1).Take(&thresholdBarangIndukFieldKategoriBarang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if thresholdBarangIndukFieldKategoriBarang == 0 {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal, tak ada kategori dalam barang induk tersebut",
		}
	}

	var id_kategori_barang []int64
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
		IdBarangInduk: data.IdBarangInduk,
	}).Limit(int(thresholdBarangIndukFieldKategoriBarang)).Find(&id_kategori_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sot_models.VarianBarang{}).Where("id_barang_induk = ? AND status IN ?", data.IdBarangInduk, [3]string{barang_enums.Down}).Updates(&sot_models.VarianBarang{
			Status: barang_enums.Ready,
		}).Error; err != nil {
			return err
		}

		for _, ik := range id_kategori_barang {
			var countDown int32 = 0
			if err := tx.Model(&sot_threshold.KategoriBarangThreshold{}).Select(stsk_kategori_barang.Down).Where(&sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: ik,
			}).Limit(1).Take(countDown).Error; err != nil {
				return err
			}

			const stok string = "stok"

			var exp map[string]interface{} = map[string]interface{}{stok: gorm.Expr(fmt.Sprintf("%s + ?", stok), countDown)}
			if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
				IdBarangInduk: int32(ik),
			}).Updates(exp).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	go func(IdBi int64, IDK []int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		err_tk, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)
		if err_tk != nil {
			fmt.Println("gagal mengambil total kategori barang")
		}

		for _, ik := range IDK {
			kategoriBarangThreshold := sot_threshold.KategoriBarangThreshold{IdKategoriBarang: ik}

			var countDown int32 = 0
			if err := Read.WithContext(konteks).Model(&sot_threshold.KategoriBarangThreshold{}).Select(stsk_kategori_barang.Down).Where(&sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: ik,
			}).Limit(1).Take(&countDown).Error; err != nil {
				fmt.Println("Gagal mengambil Countdown")
			}

			if err := kategoriBarangThreshold.CustomIncrement(ctx, Read, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Ready,
					Count:     int(countDown),
				},
			}); err != nil {
				fmt.Println("Gagal increment kategori ready")
			}

			if err := kategoriBarangThreshold.CustomDecrement(ctx, Read, []sot_threshold.CustomCounter{
				{
					FieldName: stsk_kategori_barang.Down,
					Count:     int(countDown),
				},
			}); err != nil {
				fmt.Println("Gagal decrement kategori down")
			}
		}

		var updatesDownKategori []sot_models.KategoriBarang = make([]sot_models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&sot_models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Find(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated down")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(services).SetRole(mb_cud_seeders.Seller)
			if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatedDataKategoriBarang); err != nil {
				fmt.Println("Gagal publish updated kategori barang stok ke message broker")
			}
		}
	}(int64(data.IdBarangInduk), id_kategori_barang, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func UpKategoriBarang(ctx context.Context, db *environment.InternalDBReadWriteSystem, data PayloadUpKategoriBarang, rds *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "UpKategoriBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang tak boleh lebih kecil dari 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	// 1. Validasi keberadaan kategori barang milik seller
	var id_data_kategori int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Select("id").Where(&sot_models.KategoriBarang{
		ID:       data.IdKategoriBarang,
		SellerID: data.IdentitasSeller.IdSeller,
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

	// 2. Tentukan limit varian (jika data.Jumlah == 0, ambil dari threshold Down)
	var limit_varian int64 = 0

	if data.Jumlah == 0 {
		if err := db.Read.WithContext(ctx).Model(&sot_threshold.KategoriBarangThreshold{}).Select(stsk_kategori_barang.Down).Where(&sot_threshold.KategoriBarangThreshold{
			IdKategoriBarang: data.IdKategoriBarang,
		}).Limit(1).Take(&limit_varian).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
	} else {
		limit_varian = data.Jumlah
	}

	if limit_varian <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Gagal, tidak ada varian barang yang berstatus down untuk di-up",
		}
	}

	// 3. Ambil ID varian barang yang statusnya Down berdasarkan limit
	var id_varian_barang []int64

	if err := db.Read.WithContext(ctx).
		Model(&sot_models.VarianBarang{}).
		Select("id").
		Where(&sot_models.VarianBarang{
			IdKategori: data.IdKategoriBarang,
			Status:     barang_enums.Down,
		}).
		Limit(int(limit_varian)).
		Find(&id_varian_barang).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	// 4. Jalankan transaksi database (Update stok kategori + Update status varian menjadi Ready)
	if err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		const stok string = "stok"

		// Tambah stok kategori sebanyak jumlah varian yang di-up
		var exp map[string]interface{} = map[string]interface{}{
			stok: gorm.Expr(fmt.Sprintf("%s + ?", stok), len(id_varian_barang)),
		}
		if err := tx.Model(&sot_models.KategoriBarang{}).
			Where(&sot_models.KategoriBarang{ID: data.IdKategoriBarang}).
			Updates(exp).Error; err != nil {
			return err
		}

		// Update status varian barang dari Down menjadi Ready secara massal (1 query)
		if len(id_varian_barang) > 0 {
			if err := tx.Model(&sot_models.VarianBarang{}).
				Where("id IN ?", id_varian_barang).
				Update("status", barang_enums.Ready).Error; err != nil {
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

	// 5. Background goroutine untuk update threshold & publish message broker (mengikuti pattern DownKategoriBarang)
	go func(IdKb int64, Upped int, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		kategoriBarangThreshold := sot_threshold.KategoriBarangThreshold{IdKategoriBarang: IdKb}
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		// Karena di-up: Decrement counter Down, Increment counter Ready
		if err := kategoriBarangThreshold.CustomDecrement(ctx, Read, []sot_threshold.CustomCounter{
			{
				FieldName: stsk_kategori_barang.Down,
				Count:     Upped,
			},
		}); err != nil {
			fmt.Println("Gagal decrement kategori down")
		}

		if err := kategoriBarangThreshold.CustomIncrement(ctx, Read, []sot_threshold.CustomCounter{
			{
				FieldName: stsk_kategori_barang.Ready,
				Count:     Upped,
			},
		}); err != nil {
			fmt.Println("Gagal increment kategori ready")
		}

		// Ambil data kategori terbaru untuk dipublish
		var dataKategoriBarangUpdated sot_models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName("UpKategoriBarang").SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update kategori barang Id: %v ke message broker\n", IdKb)
		}
	}(data.IdKategoriBarang, len(id_varian_barang), db.Read, cud_publisher)

	log.Printf("[INFO] Sejumlah %d stok kategori ID %d pada barang induk ID %d berhasil di-up-kan oleh seller ID %d", len(id_varian_barang), data.IdKategoriBarang, data.IdBarangInduk, data.IdentitasSeller.IdSeller)
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
}

func EditRekeningBarangInduk(ctx context.Context, data PayloadEditRekeningBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditRekeningBarangInduk"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdRekeningSeller <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdRekeningSeller Tak boleh lebih kecil dari 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal Kredensial Seller Tidak Valid",
		}
	}

	var id_data_rekening int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.RekeningSeller{}).Select("id").Where(&sot_models.RekeningSeller{
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
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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

	if err_kategori := db.Write.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		_, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)

		var updatesDownKategori []sot_models.KategoriBarang = make([]sot_models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&sot_models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Find(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated rekening")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(dk.TableName()).SetRole(mb_cud_seeders.Seller)
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
} // Tuned

func EditAlamatGudangBarangInduk(ctx context.Context, data PayloadEditAlamatBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahAlamatGudangBarangInduk"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdAlamatGudang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdAlamatGudang Tak boleh lebih kecil dari 0",
		}
	}

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

	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id").Where(&sot_models.BarangInduk{
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

	if errCheck := db.Read.WithContext(ctx).Model(&sot_models.AlamatGudang{}).Select("id").
		Where(&sot_models.AlamatGudang{
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

	if err_edit := db.Write.WithContext(ctx).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
		IdBarangInduk: int32(id_data_barang_induk),
	}).Update("id_alamat_gudang", data.IdAlamatGudang).Error; err_edit != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal, server sedang sibuk. Coba lagi lain waktu",
		}
	}

	go func(IdBi int64, Read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: IdBi,
		}

		err_tk, totalKategori := thresholdBarangInduk.GetKolomCount(konteks, Read, stsk_baranginduk.KategoriBarang)
		if err_tk != nil {
			fmt.Println("Gagal mengambil total kategori")
		}

		var updatesDownKategori []sot_models.KategoriBarang = make([]sot_models.KategoriBarang, 0, totalKategori)
		if err := Read.WithContext(konteks).Where(&sot_models.KategoriBarang{
			IdBarangInduk: int32(IdBi),
		}).Limit(totalKategori).Find(&updatesDownKategori).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang updated alamat gudang")
		}

		for _, dk := range updatesDownKategori {
			updatedDataKategoriBarang := mb_cud_serializer.NewJsonPayload().SetPayload(dk).SetTableName(dk.TableName()).SetRole(mb_cud_seeders.Seller)
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
} // Tuned

func EditAlamatGudangBarangKategori(ctx context.Context, data PayloadEditAlamatBarangKategori, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahAlamatGudangBarangKategori"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdKategoriBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKategoriBarang Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdAlamatGudang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdAlamatGudang Tak boleh lebih kecil dari 0",
		}
	}

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
	if err := db.Read.Model(sot_models.KategoriBarang{}).Select("id").Where(sot_models.KategoriBarang{
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

	if errCheck := db.Read.Model(&sot_models.AlamatGudang{}).Select("id").
		Where(&sot_models.AlamatGudang{
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

	if err_edit := db.Write.Model(sot_models.KategoriBarang{}).Where(sot_models.KategoriBarang{
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
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var dataKategoriBarangUpdated sot_models.KategoriBarang
		if err := Read.WithContext(konteks).Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
			ID: IdKb,
		}).Limit(1).Take(&dataKategoriBarangUpdated).Error; err != nil {
			fmt.Println("Gagal mendapatkan data kategori barang")
		}

		kategoriBarangUpdatedPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKategoriBarangUpdated).SetTableName(dataKategoriBarangUpdated.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, kategoriBarangUpdatedPublish); err != nil {
			fmt.Printf("Gagal publish update kategori barang Id: %v ke message broker", IdKb)
		}
	}(data.IdKategoriBarang, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Alamat gudang berhasil diubah.",
	}
} // Tuned

func MasukanKomentarBarang(ctx context.Context, data PayloadMasukanKomentarBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "TambahKomentarBarang"

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if strings.TrimSpace(data.Komentar) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Sku tak boleh kosong",
		}
	}

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	is_seller := false
	var id_seller_take int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id_seller").Where(&sot_models.BarangInduk{
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

	NewKomentar := sot_models.Komentar{
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

	go func(K sot_models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
			IdBarangInduk: int64(K.IdBarangInduk),
		}

		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: K.ID,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdKomentar.Inisialisasi(konteks, Trh); err != nil {
			fmt.Println("Gagal membuat threshold komentar")
		}

		if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal increment total komentar barang induk ke threshold barang induk")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar baru barang induk ke message broker")
		}

	}(NewKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} // Tuned

func EditKomentarBarang(ctx context.Context, data PayloadEditKomentarBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditKomentarBarang"

	if data.IdKomentar <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKomentar Tak boleh lebih kecil dari 0",
		}
	}

	if strings.TrimSpace(data.Komentar) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Sku tak boleh kosong",
		}
	}

	_, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak valid",
		}
	}

	var id_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.Komentar{}).Select("id").Where(&sot_models.Komentar{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
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
		komentarData := sot_models.Komentar{}
		if err := Read.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
			ID: idKomen,
		}).Limit(1).Take(&komentarData); err != nil {
			return
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		newUpdateKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(komentarData).SetTableName(komentarData.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newUpdateKomentarPublish); err != nil {
			fmt.Println("Gagal publish update komentar barang ke message broker")
		}

	}(id_komentar, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} //Tuned

func HapusKomentarBarang(ctx context.Context, data PayloadHapusKomentarBarangInduk, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusKomentarBarang"

	if data.IdKomentar <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKomentar Tak boleh lebih kecil dari 0",
		}
	}

	if _, status := data.IdentitasSeller.Validating(ctx, db.Read, rds_session); !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal data seller tidak ditemukan",
		}
	}

	var Komentar sot_models.Komentar
	if err := db.Read.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.Komentar{}).Where(&sot_models.Komentar{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Delete(&sot_models.Komentar{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Menghapus Komentar",
		}
	}

	go func(K sot_models.Komentar, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		barangIndukThreshold := sot_threshold.BarangIndukThreshold{
			ID: int64(K.IdBarangInduk),
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := barangIndukThreshold.Decrement(konteks, Trh, stsk_baranginduk.Komentar); err != nil {
			fmt.Println("Gagal decr komentar barang induk ke threshold barang induk")
		}

		if err := Trh.WithContext(konteks).Model(&sot_threshold.KomentarThreshold{}).Where(&sot_threshold.KomentarThreshold{ID: K.ID}).Delete(&sot_threshold.KomentarThreshold{}).Error; err != nil {
			fmt.Println("Gagal menghapus threshold komentar")
		}

		newDeleteKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(K).SetTableName(K.TableName()).SetRole(mb_cud_seeders.Seller)
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

func MasukanChildKomentar(ctx context.Context, data PayloadMasukanChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "MasukanChildKomentar"

	if data.IdKomentarBarang <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKomentarBarang Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if strings.TrimSpace(data.Komentar) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Sku tak boleh kosong",
		}
	}

	is_seller := false

	var id_seller_take int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id_seller").Where(&sot_models.BarangInduk{
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

	newKomentar := sot_models.KomentarChild{
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

	go func(Kc sot_models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar reply ke message broker")
		}
	}(newKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} // Tuned

func MentionChildKomentar(ctx context.Context, data PayloadMentionChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "MentionChildKomentar"

	if data.IdKomentar <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKomentarBarang Tak boleh lebih kecil dari 0",
		}
	}

	if data.IdBarangInduk <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdBarangInduk Tak boleh lebih kecil dari 0",
		}
	}

	if strings.TrimSpace(data.UsernameMentioned) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, UsernameMention Tak boleh kosong",
		}
	}

	if !helper.Contains(data.UsernameMentioned, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "BLANK"}) || !helper.Contains(data.UsernameMentioned, []string{"_", "BLANK"}) {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, UsernameMention Tak valid",
		}
	}

	if strings.TrimSpace(data.Komentar) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Sku tak boleh kosong",
		}
	}

	is_seller := false

	var id_seller_take int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("id_seller").Where(&sot_models.BarangInduk{
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

	newKomentar := sot_models.KomentarChild{
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

	go func(Kc sot_models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		thresholdKomentar := sot_threshold.KomentarThreshold{
			IdKomentar: Kc.IdKomentar,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := thresholdKomentar.Increment(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal increment total komentar child induk ke threshold komentar")
		}

		newKomentarPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newKomentarPublish); err != nil {
			fmt.Println("Gagal publish komentar reply ke message broker")
		}
	}(newKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} // Tuned

func EditChildKomentar(ctx context.Context, data PayloadEditChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "EditChildKomentar"

	if data.IdKomentar <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKomentarBarang Tak boleh lebih kecil dari 0",
		}
	}

	if strings.TrimSpace(data.Komentar) == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Sku tak boleh kosong",
		}
	}

	var id_edit_child_komentar int64 = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.KomentarChild{}).Select("id").Where(&sot_models.KomentarChild{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
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
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		var dataKomentarChild sot_models.KomentarChild
		if err := Read.WithContext(konteks).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
			ID: IdKc,
		}).Limit(1).Take(&dataKomentarChild).Error; err != nil {
			return
		}

		updateKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataKomentarChild).SetTableName(dataKomentarChild.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updateKomentarChildPublish); err != nil {
			fmt.Println("Gagal publish update child komentar ke message broker")
		}
	}(id_edit_child_komentar, db.Read, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} // Tuned

func HapusChildKomentar(ctx context.Context, data PayloadHapusChildKomentar, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "HapusChildKomentar"

	if data.IdKomentar <= 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal, IdKomentarBarang Tak boleh lebih kecil dari 0",
		}
	}

	var childKomentar sot_models.KomentarChild
	if err := db.Read.WithContext(ctx).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
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

	if err := db.Write.WithContext(ctx).Model(&sot_models.KomentarChild{}).Where(&sot_models.KomentarChild{
		ID:          data.IdKomentar,
		IdEntity:    int64(data.IdentitasSeller.IdSeller),
		JenisEntity: entity_enums.Seller,
	}).Delete(&sot_models.KomentarChild{}).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal Menghapus Komentar",
		}
	}

	go func(Kc sot_models.KomentarChild, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		komentarThreshold := sot_threshold.KomentarThreshold{
			ID: Kc.IdKomentar,
		}

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()

		if err := komentarThreshold.Decrement(konteks, Trh, stsk_komentar.KomentarChild); err != nil {
			fmt.Println("Gagal decrement komentar child ke threshold komentar")
		}

		deleteKomentarChildPublish := mb_cud_serializer.NewJsonPayload().SetPayload(Kc).SetTableName(Kc.TableName()).SetRole(mb_cud_seeders.Seller)
		if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, deleteKomentarChildPublish); err != nil {
			fmt.Println("Gagal publish delete komentar child ke message broker")
		}
	}(childKomentar, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil",
	}
} //Tuned
