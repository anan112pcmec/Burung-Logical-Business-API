package pengguna_transaction_services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	ekspedisi "github.com/anan112pcmec/Burung-backend-1/app/api/ekspedisi_raja_ongkir"
	ekspedisi_cost "github.com/anan112pcmec/Burung-backend-1/app/api/ekspedisi_raja_ongkir/cost"
	open_route_direction "github.com/anan112pcmec/Burung-backend-1/app/api/open_route_map/direction"
	payment_gateaway "github.com/anan112pcmec/Burung-backend-1/app/api/payment_in_midtrans"
	payment_in_gerai "github.com/anan112pcmec/Burung-backend-1/app/api/payment_in_midtrans/gerai"
	payment_in_va "github.com/anan112pcmec/Burung-backend-1/app/api/payment_in_midtrans/virtual_account"
	payment_in_wallet "github.com/anan112pcmec/Burung-backend-1/app/api/payment_in_midtrans/wallet"
	data_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/data"
	"github.com/anan112pcmec/Burung-backend-1/app/config"
	barang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/barang"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/jenis_kendaraan_kurir"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_alamat_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/alamat_pengguna"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/transaction_services/response_transaction_pengguna"
)

func CheckoutBarangUser(ctx context.Context, data PayloadCheckoutBarang, db *config.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "CheckoutBarangUser"
	log.Printf("[%s] Memulai proses checkout untuk user ID: %v", services, data.IdentitasPengguna.ID)

	// Validasi pengguna
	if _, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !status {
		log.Printf("[%s] Kredensial pengguna tidak valid untuk user ID: %v", services, data.IdentitasPengguna.ID)
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Message:  "Gagal identitas pengguna tidak ditemukan",
		}
	}

	totalDipesan := 0

	// Preallocate ukuran slice idKeranjang sesuai jumlah data checkout
	// AMAN: jangan set lebih besar dari len(data.DataCheckout) supaya loop tidak OOB
	dataLen := len(data.DataCheckout)
	idKeranjang := make([]int64, 0, dataLen)
	KeranjangData := make([]models.Keranjang, 0, dataLen)

	// Loop menggunakan dataLen agar aman, tidak baca len() berulang
	for i := 0; i < dataLen; i++ {
		// tambahan defensive: pastikan index valid sebelum akses
		if i < 0 || i >= len(data.DataCheckout) {
			continue
		}
		item := data.DataCheckout[i]

		// Validasi item agar tidak nil atau field kosong (opsional)
		if item.ID == 0 {
			log.Printf("[CheckoutBarangUser] ID keranjang tidak valid pada indeks %d", i)
			continue
		}

		totalDipesan += int(item.Jumlah)
		idKeranjang = append(idKeranjang, item.ID)
	}

	if err := db.Read.WithContext(ctx).Model(&models.Keranjang{}).Where("id IN ?", idKeranjang).Limit(dataLen).Take(&KeranjangData).Error; err != nil {
		fmt.Println("Gagal mendapatkan data seluruh keranjang")
	}

	responseData := make([]response_transaction_pengguna.CheckoutData, 0, dataLen)
	varianUpdates := make([]int64, 0, totalDipesan)
	kategoriUpdates := make(map[int32]int32, dataLen)
	BarangInduk := make(map[int64]models.BarangInduk, dataLen)
	KategoriBarang := make(map[int64]models.KategoriBarang, dataLen)
	NamaSeller := make(map[int64]string, dataLen)

	for i := 0; i < len(data.DataCheckout); i++ {
		if i < 0 || i >= len(data.DataCheckout) {
			continue
		}

		jumlahNeeded := int(data.DataCheckout[i].Jumlah)
		var idsVarianStok []int64 = make([]int64, 0, jumlahNeeded)
		if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").Where(&models.VarianBarang{
			IdBarangInduk: data.DataCheckout[i].IdBarangInduk,
			IdKategori:    data.DataCheckout[i].IdKategori,
			Status:        barang_enums.Ready,
		}).Limit(jumlahNeeded).Scan(&idsVarianStok).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

		if len(idsVarianStok) < jumlahNeeded {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal barang lebih sedikit dibanding yang kamu pesan",
			}
		}

		varianUpdates = append(varianUpdates, idsVarianStok...)

		if BarangInduk[int64(data.DataCheckout[i].IdBarangInduk)].NamaBarang == "" {
			barang := models.BarangInduk{}

			if err := db.Read.WithContext(ctx).Model(&models.BarangInduk{}).Select("nama_barang", "id_seller", "jenis_barang").Where(&models.BarangInduk{
				ID: int32(data.DataCheckout[i].IdBarangInduk),
			}).Limit(1).Scan(&barang).Error; err != nil {
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Gagal server sedang sibuk coba lagi lain waktu",
				}
			}

			BarangInduk[int64(data.DataCheckout[i].IdBarangInduk)] = barang
		}

		if KategoriBarang[data.DataCheckout[i].IdKategori].Nama == "" {
			var kategori models.KategoriBarang = models.KategoriBarang{Nama: ""}
			if err := db.Read.Model(&models.KategoriBarang{}).Select("nama", "harga", "stok", "id_barang_induk", "id_alamat_gudang", "berat_gram").
				Where(&models.KategoriBarang{ID: data.DataCheckout[i].IdKategori}).Limit(1).Scan(&kategori).Error; err != nil {
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Gagal server sedang sibuk coba lagi lain waktu",
				}
			}

			if kategori.Nama == "" {
				return &response.ResponseForm{
					Status:   http.StatusNotFound,
					Services: services,
					Message:  "gagal data kategori tidak ditemukan",
				}
			}

			KategoriBarang[data.DataCheckout[i].IdKategori] = kategori
		}

		kategoriUpdates[int32(data.DataCheckout[i].IdKategori)] += int32(data.DataCheckout[i].Jumlah)

		if NamaSeller[int64(data.DataCheckout[i].IdSeller)] == "" {
			var namaSeller string = ""
			if err := db.Read.Model(&models.Seller{}).Select("nama").
				Where(&models.Seller{ID: data.DataCheckout[i].IdSeller}).
				Limit(1).Scan(&namaSeller).Error; err != nil {
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Gagal server sedang sibuk coba lagi lain waktu",
				}
			}

			if namaSeller == "" {
				return &response.ResponseForm{
					Status:   http.StatusNotFound,
					Services: services,
					Message:  "Gagal seller tidak ditemukan",
				}
			}

			NamaSeller[int64(data.DataCheckout[i].IdSeller)] = namaSeller
		}

		resp := response_transaction_pengguna.CheckoutData{
			IDUser:           data.IdentitasPengguna.ID,
			IDSeller:         data.DataCheckout[i].IdSeller,
			NamaSeller:       NamaSeller[int64(data.DataCheckout[i].IdSeller)],
			JenisBarang:      BarangInduk[int64(data.DataCheckout[i].IdBarangInduk)].JenisBarang,
			IdBarangInduk:    data.DataCheckout[i].IdBarangInduk,
			IdKategoriBarang: data.DataCheckout[i].IdKategori,
			IdAlamatGudang:   KategoriBarang[data.DataCheckout[i].IdKategori].IDAlamat,
			HargaKategori:    KategoriBarang[data.DataCheckout[i].IdKategori].Harga,
			NamaBarang:       BarangInduk[int64(data.DataCheckout[i].IdBarangInduk)].NamaBarang,
			NamaKategori:     KategoriBarang[data.DataCheckout[i].IdKategori].Nama,
			BeratKategori:    KategoriBarang[data.DataCheckout[i].IdKategori].BeratGram,
			Dipesan:          int32(data.DataCheckout[i].Jumlah),
			Message:          "Siap",
			Status:           true,
		}

		responseData = append(responseData, resp)
	}

	err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// Update status varian sekaligus
		if len(varianUpdates) > 0 {
			if err := tx.Model(&models.VarianBarang{}).
				Where("id IN ?", varianUpdates).
				Updates(&models.VarianBarang{
					Status:       barang_enums.Dipesan,
					HoldBy:       data.IdentitasPengguna.ID,
					HolderEntity: entity_enums.Pengguna,
				}).Error; err != nil {
				return err
			}
		}

		// Update stok kategori secara atomic
		for kategoriID, totalDipesan := range kategoriUpdates {
			if err := tx.Model(&models.KategoriBarang{}).
				Where("id = ? AND stok >= ?", kategoriID, totalDipesan).
				UpdateColumn("stok", gorm.Expr("stok - ?", totalDipesan)).Error; err != nil {
				return err
			}
		}

		// Hapus keranjang menggunakan tx agar konsisten dengan transaksi
		if len(idKeranjang) > 0 {
			if err := tx.WithContext(ctx).Model(&models.Keranjang{}).Where("id IN ?", idKeranjang).Delete(&models.Keranjang{}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	go func(Dk []models.Keranjang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		var wg sync.WaitGroup
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()
		for _, k := range Dk {
			wg.Add(1)
			go func(datakeranjang models.Keranjang, t *gorm.DB, p *mb_cud_publisher.Publisher) {
				defer wg.Done()
				thresholdPengguna := sot_threshold.PenggunaThreshold{
					IdPengguna: datakeranjang.IdPengguna,
				}

				thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
					IdBarangInduk: int64(datakeranjang.IdBarangInduk),
				}

				thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
					IdKategoriBarang: datakeranjang.IdKategori,
				}

				if err := thresholdPengguna.Decrement(konteks, t, stsk_pengguna.Keranjang); err != nil {
					fmt.Println("Gagal decr count keranjang ke pengguna threshold")
				}

				if err := thresholdBarangInduk.Decrement(konteks, t, stsk_baranginduk.Keranjang); err != nil {
					fmt.Println("Gagal decr count keranjang ke barang induk threshold")
				}

				if err := thresholdKategoriBarang.Decrement(konteks, t, stsk_kategori_barang.Keranjang); err != nil {
					fmt.Println("Gagal decr count keranjang ke kategori barang threshold")
				}

				deleteKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(datakeranjang).SetTableName(datakeranjang.TableName())
				if err := mb_cud_publisher.DeletePublish[*mb_cud_serializer.PublishPayloadJson](konteks, p, deleteKeranjangPublish); err != nil {
					fmt.Println("Gagal publish delete keranjang ke message broker")
				}
			}(k, Trh, publisher)
		}

		wg.Wait()

	}(KeranjangData, db.Write, cud_publisher)

	if err != nil {
		log.Printf("[%s] Gagal checkout: %v", services, err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Terjadi Kesalahan pada server, Silahkan coba lagi lain waktu",
		}
	}

	// Hapus keranjang

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "CheckoutBerhasil",
		Payload: response_transaction_pengguna.ResponseDataCheckout{
			DataResponse: responseData,
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Batal Checkout User
// Befungsi Untuk MembatalkanCheckout yang telah dilakukan
// ////////////////////////////////////////////////////////////////////////////////////

func BatalCheckoutUser(data response_transaction_pengguna.ResponseDataCheckout, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "BatalCheckoutKeranjang"

	var varianIDs []int64
	kategoriUpdates := make(map[int32]int32) // kategoriID => total jumlah dikembalikan

	for _, keranjang := range data.DataResponse {
		var varian_id []int64
		if err := db.Read.Model(&models.VarianBarang{}).
			Where(models.VarianBarang{
				IdBarangInduk: keranjang.IdBarangInduk,
				IdKategori:    keranjang.IdKategoriBarang,
				Status:        barang_enums.Dipesan,
				HoldBy:        keranjang.IDUser,
			}).
			Limit(int(keranjang.Dipesan)).
			Pluck("id", &varian_id).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}
		varianIDs = append(varianIDs, varian_id...)
		kategoriUpdates[int32(keranjang.IdKategoriBarang)] += keranjang.Dipesan
	}

	err := db.Write.Transaction(func(tx *gorm.DB) error {
		// Update status semua varian sekaligus
		if len(varianIDs) > 0 {
			if err := tx.Model(&models.VarianBarang{}).
				Where("id IN ?", varianIDs).
				Updates(map[string]interface{}{
					"status":        barang_enums.Ready,
					"hold_by":       0,
					"holder_entity": "",
				}).Error; err != nil {
				return err
			}
		}

		// Update stok kategori secara atomic
		for kategoriID, totalDikembalikan := range kategoriUpdates {
			if err := tx.Model(&models.KategoriBarang{}).
				Where("id = ?", kategoriID).
				UpdateColumn("stok", gorm.Expr("stok + ?", totalDikembalikan)).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server sedang sibuk coba lagi lain waktu",
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil data checkout di hapus",
	}
}

// ////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Snap Transaksi
// Fungsi yang melayani api pada pengguna dan memanfaaykan Validate Transaksi Dan Formatting transaksi(2 fungsi
// pendukungnya)
// ////////////////////////////////////////////////////////////////////////////////////

func SnapTransaksi(ctx context.Context, data PayloadSnapTransaksiRequest, db *config.InternalDBReadWriteSystem, rds_session *redis.Client) *response.ResponseForm {
	services := "SnapTransaksiUser"
	fmt.Println("[TRACE] Start SnapTransaksi")

	// Defensive: Validate context
	if ctx == nil {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Context tidak valid",
		}
	}

	// Defensive: Validate database connection
	if db == nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Database connection tidak tersedia",
		}
	}

	model, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload:  "Gagal Validasi User Tidak Valid",
		}
	}

	// Defensive: Validate checkout data
	if len(data.DataCheckout.DataResponse) == 0 {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Data checkout kosong",
		}
	}

	lenData := len(data.DataCheckout.DataResponse)

	var sellerTransaction map[int32]models.Seller = make(map[int32]models.Seller, lenData)

	for i := 0; i < lenData; i++ {
		var errcheck bool = false

		// Defensive: Validate item data
		if data.DataCheckout.DataResponse[i].IDSeller <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Message:  fmt.Sprintf("ID Seller tidak valid pada item ke-%d", i+1),
			}
		}

		if data.DataCheckout.DataResponse[i].Dipesan <= 0 {
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Message:  fmt.Sprintf("Jumlah pesanan tidak valid pada item ke-%d", i+1),
			}
		}

		// Defensive: Check if seller exists in map
		if _, exists := sellerTransaction[data.DataCheckout.DataResponse[i].IDSeller]; !exists {
			var seller models.Seller
			if err := db.Read.WithContext(ctx).Model(&models.Seller{}).Where(&models.Seller{
				ID: data.DataCheckout.DataResponse[i].IDSeller,
			}).Limit(1).First(&seller).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &response.ResponseForm{
						Status:   http.StatusNotFound,
						Services: services,
						Message:  fmt.Sprintf("Seller dengan ID %d tidak ditemukan", data.DataCheckout.DataResponse[i].IDSeller),
					}
				}
				errcheck = true
			} else {
				sellerTransaction[data.DataCheckout.DataResponse[i].IDSeller] = seller
			}
		}

		var varianIds []int64 = make([]int64, 0, int(data.DataCheckout.DataResponse[i].Dipesan))
		if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").Where(&models.VarianBarang{
			IdBarangInduk: data.DataCheckout.DataResponse[i].IdBarangInduk,
			IdKategori:    data.DataCheckout.DataResponse[i].IdKategoriBarang,
			Status:        barang_enums.Dipesan,
			HoldBy:        data.DataCheckout.DataResponse[i].IDUser,
		}).Limit(int(data.DataCheckout.DataResponse[i].Dipesan)).Find(&varianIds).Error; err != nil {
			errcheck = true
			fmt.Printf("[ERROR] Gagal query varian barang: %v\n", err)
		}

		if len(varianIds) != int(data.DataCheckout.DataResponse[i].Dipesan) {
			errcheck = true
		}

		if errcheck {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusUnavailableForLegalReasons,
				Services: services,
				Message:  "Data Dipesan Tidak Konsisten dengan checkout",
			}
		}
	}

	fmt.Println("[TRACE] Generate PaymentCode")
	var PaymentCode string
	var err_payment error
	maxRetry := 10
	for i := 0; i < maxRetry; i++ {
		PaymentCode, err_payment = helper.GenerateAutoPaymentId(db.Read)
		if err_payment == nil {
			fmt.Printf("[TRACE] PaymentCode berhasil dibuat: %s (percobaan ke-%d)\n", PaymentCode, i+1)
			break
		} else {
			fmt.Printf("[TRACE] Gagal generate PaymentCode (percobaan ke-%d): %v\n", i+1, err_payment)
		}
	}

	// Defensive: Validate PaymentCode generation
	if err_payment != nil || PaymentCode == "" {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal generate kode pembayaran setelah beberapa percobaan",
		}
	}

	// Defensive: Validate address information
	if data.AlamatInformation.NamaAlamat == "" || data.AlamatInformation.Kota == "" {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Data alamat tidak lengkap",
		}
	}

	AlamatPengguna := midtrans.CustomerAddress{
		Address:     data.AlamatInformation.NamaAlamat,
		City:        data.AlamatInformation.Kota,
		Phone:       data.AlamatInformation.NomorTelephone,
		Postcode:    data.AlamatInformation.KodePos,
		CountryCode: data.AlamatInformation.KodeNegara,
	}

	fmt.Println("Berhasil Membuat Alamat Pengguna")

	var PM []snap.SnapPaymentType
	switch data.PaymentMethod {
	case "va":
		PM = []snap.SnapPaymentType{
			snap.PaymentTypeBCAVA,
			snap.PaymentTypeBNIVA,
			snap.PaymentTypeBRIVA,
			snap.PaymentTypePermataVA,
		}
	case "wallet":
		PM = []snap.SnapPaymentType{
			snap.PaymentTypeGopay,
			snap.PaymentTypeShopeepay,
		}
	case "gerai":
		PM = []snap.SnapPaymentType{
			snap.PaymentTypeIndomaret,
			snap.PaymentTypeAlfamart,
		}
	case "credit":
		PM = []snap.SnapPaymentType{
			snap.PaymentTypeAkulaku,
			snap.PaymentTypeCreditCard,
		}
	default:
		// Defensive: Handle invalid payment method
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  fmt.Sprintf("Metode pembayaran '%s' tidak valid", data.PaymentMethod),
		}
	}

	var hasil []midtrans.ItemDetails = make([]midtrans.ItemDetails, 0, lenData)

	// Defensive: Validate operational data
	if data_cache.OperationalPengirimanData.CommittedOperationalData.DataTarifPengiriman.TarifSistem <= 0 {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Tarif sistem tidak valid",
		}
	}

	var biaya_platform = data_cache.OperationalPengirimanData.CommittedOperationalData.DataTarifPengiriman.TarifSistem
	var AlamatGudang map[int64]models.AlamatGudang = make(map[int64]models.AlamatGudang, lenData)
	var dataTransaksi []response_transaction_pengguna.DataTransaksi = make([]response_transaction_pengguna.DataTransaksi, 0, lenData)

	fmt.Println("Berhasil Mengambil biaya platform:", biaya_platform)
	for i := 0; i < lenData; i++ {
		// Defensive: Validate price and weight
		if data.DataCheckout.DataResponse[i].HargaKategori <= 0 {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Message:  fmt.Sprintf("Harga tidak valid pada item ke-%d", i+1),
			}
		}

		if data.DataCheckout.DataResponse[i].BeratKategori <= 0 {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Message:  fmt.Sprintf("Berat tidak valid pada item ke-%d", i+1),
			}
		}

		totalHargapembelian := data.DataCheckout.DataResponse[i].HargaKategori * data.DataCheckout.DataResponse[i].Dipesan
		beratTotal := data.DataCheckout.DataResponse[i].BeratKategori * int16(data.DataCheckout.DataResponse[i].Dipesan) / 1000
		totalHargaBerat := data_cache.OperationalPengirimanData.CommittedOperationalData.DataTarifPengiriman.TarifKurirPerKg * int64(beratTotal)

		hasil = append(hasil, midtrans.ItemDetails{
			ID:           fmt.Sprintf("%v--%v", data.DataCheckout.DataResponse[i].IdBarangInduk, data.DataCheckout.DataResponse[i].IdKategoriBarang),
			Price:        int64(data.DataCheckout.DataResponse[i].HargaKategori),
			Qty:          data.DataCheckout.DataResponse[i].Dipesan,
			Name:         fmt.Sprintf("%s - %s", data.DataCheckout.DataResponse[i].NamaBarang, data.DataCheckout.DataResponse[i].NamaKategori),
			MerchantName: data.DataCheckout.DataResponse[i].NamaSeller,
			Category:     data.DataCheckout.DataResponse[i].JenisBarang,
		})

		// Defensive: Check if warehouse address exists
		if _, exists := AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang]; !exists {
			var alamat models.AlamatGudang
			if err := db.Read.WithContext(ctx).Model(&models.AlamatGudang{}).Where(&models.AlamatGudang{
				ID: data.DataCheckout.DataResponse[i].IdAlamatGudang,
			}).Limit(1).Take(&alamat).Error; err != nil {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &response.ResponseForm{
						Status:   http.StatusNotFound,
						Services: services,
						Message:  fmt.Sprintf("Alamat gudang tidak ditemukan untuk item ke-%d", i+1),
					}
				}
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Gagal mengambil data alamat gudang",
				}
			}
			AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang] = alamat
		}

		var isEkspedisi bool = false
		if AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Provinsi != data.AlamatInformation.Provinsi {
			isEkspedisi = true
		}

		var key struct {
			TitikMulaiLat   float64
			TitikMulaiLong  float64
			TitikTujuanLat  float64
			TitikTujuanLong float64
		}

		var IdAlamatEkspedisi int64 = 0
		if isEkspedisi {
			var id_alamat_eks int64 = 0
			if err := db.Read.WithContext(ctx).Model(&models.AlamatEkspedisi{}).Select("id").Where(&models.AlamatEkspedisi{
				Kota: AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota,
			}).Order("id DESC").Limit(1).Scan(&id_alamat_eks).Error; err != nil {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Gagal mengambil data alamat ekspedisi",
				}
			}

			if id_alamat_eks == 0 {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusNotFound,
					Services: services,
					Message:  fmt.Sprintf("Alamat ekspedisi tidak ditemukan untuk kota %s", data.AlamatInformation.Kota),
				}
			}

			// Defensive: Validate ekspedisi data exists
			if _, cityExists := data_cache.DataAlamatEkspedisi[AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota]; !cityExists {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusNotFound,
					Services: services,
					Message:  "Data ekspedisi tidak tersedia untuk kota tujuan",
				}
			}

			if _, addrExists := data_cache.DataAlamatEkspedisi[AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota][id_alamat_eks]; !addrExists {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusNotFound,
					Services: services,
					Message:  "Alamat ekspedisi tidak valid",
				}
			}

			key.TitikMulaiLong = AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Longitude
			key.TitikMulaiLat = AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Latitude
			key.TitikTujuanLong = data_cache.DataAlamatEkspedisi[AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota][id_alamat_eks].Longitude
			key.TitikTujuanLat = data_cache.DataAlamatEkspedisi[AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota][id_alamat_eks].Latitude

			IdAlamatEkspedisi = data_cache.DataAlamatEkspedisi[AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota][id_alamat_eks].ID
		} else {
			key.TitikMulaiLong = AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Longitude
			key.TitikMulaiLat = AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Latitude
			key.TitikTujuanLong = data.AlamatInformation.Longitude
			key.TitikTujuanLat = data.AlamatInformation.Latitude
		}

		// Defensive: Validate coordinates
		if key.TitikMulaiLong == 0 || key.TitikMulaiLat == 0 || key.TitikTujuanLong == 0 || key.TitikTujuanLat == 0 {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Message:  "Koordinat lokasi tidak valid",
			}
		}

		Jarak, hargaJarak, status := open_route_direction.HitungJarakHargaDirection(
			[2]float64{key.TitikMulaiLong, key.TitikMulaiLat},
			[2]float64{key.TitikTujuanLong, key.TitikTujuanLat},
		)

		// Defensive: Validate distance calculation
		if !status {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal menghitung jarak pengiriman",
			}
		}

		if Jarak <= 0 {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Message:  "Jarak pengiriman tidak valid",
			}
		}

		if Jarak >= 80 {
			isEkspedisi = true
		}

		// Defensive: Validate delivery service type
		validLayanan := map[string]bool{"reguler": true, "fast": true, "instant": true}
		if !validLayanan[data.LayananPengirimanKurir] {
			data.LayananPengirimanKurir = "reguler"
		}

		switch data.LayananPengirimanKurir {
		case "fast":
			if Jarak > 50 {
				data.LayananPengirimanKurir = "reguler"
			}
		case "instant":
			if Jarak > 30 {
				data.LayananPengirimanKurir = "reguler"
			}
		default:
			// Already reguler or handled above
		}

		// Defensive: Validate tarif data exists
		if _, exists := data_cache.DataTarifJenisPengiriman[data.LayananPengirimanKurir]; !exists {
			_ = BatalCheckoutUser(data.DataCheckout, db)
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Tarif pengiriman tidak tersedia",
			}
		}

		hargaJarak += data_cache.DataTarifJenisPengiriman[data.LayananPengirimanKurir].Harga

		var hargaEkspedisi int64 = 0
		if isEkspedisi {
			// Defensive: Validate city mapping
			originCity, originExists := ekspedisi.JawaCities[AlamatGudang[data.DataCheckout.DataResponse[i].IdAlamatGudang].Kota]
			destinationCity, destExists := ekspedisi.JawaCities[data.AlamatInformation.Kota]

			if !originExists || !destExists {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusBadRequest,
					Services: services,
					Message:  "Kota asal atau tujuan tidak didukung untuk ekspedisi",
				}
			}

			weight := float64(data.DataCheckout.DataResponse[i].BeratKategori) * float64(data.DataCheckout.DataResponse[i].Dipesan)
			if weight <= 0 {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusBadRequest,
					Services: services,
					Message:  "Berat paket tidak valid",
				}
			}

			reqPayload := ekspedisi_cost.StarterDomesticCostReq{
				Origin:      originCity,
				Destination: destinationCity,
				Weight:      strconv.FormatFloat(weight, 'f', -1, 64),
				Courier:     "jne",
				Price:       "lowest",
			}

			res := reqPayload.DomesticCostReq(ctx)

			// Defensive: Validate ekspedisi response
			if len(res.Data) == 0 {
				_ = BatalCheckoutUser(data.DataCheckout, db)
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Gagal mendapatkan tarif ekspedisi",
				}
			}

			hargaEkspedisi = int64(res.Data[0].Cost)
			if hargaEkspedisi < 0 {
				hargaEkspedisi = 0
			}
		}

		dataTransaksiIterasi := response_transaction_pengguna.DataTransaksi{
			IdAlamatEkspedisi: IdAlamatEkspedisi,
			HargaBarang:       int64(totalHargapembelian),
			HargaBerat:        totalHargaBerat,
			HargaJarak:        hargaJarak,
			HargaEkspedisi:    hargaEkspedisi,
			IsEkspedisi:       isEkspedisi,
			Jarak:             Jarak,
			TotalTagihan:      int64(totalHargapembelian) + totalHargaBerat + hargaJarak + hargaEkspedisi,
		}
		dataTransaksi = append(dataTransaksi, dataTransaksiIterasi)
	}

	var harga_kirim int64 = 0
	for i := 0; i < len(dataTransaksi); i++ {
		harga_kirim += dataTransaksi[i].HargaBerat + dataTransaksi[i].HargaEkspedisi + dataTransaksi[i].HargaJarak
	}

	hasil = append(hasil, midtrans.ItemDetails{
		ID:           "fee-courier",
		Price:        harga_kirim,
		Qty:          1,
		Name:         fmt.Sprintf("Biaya Kurir - %s", data_cache.DataTarifJenisPengiriman[data.LayananPengirimanKurir].Nama),
		MerchantName: "Courier",
		Category:     "fee",
	})
	total := harga_kirim

	// ==== Biaya aplikasi ====
	hasil = append(hasil, midtrans.ItemDetails{
		ID:           "fee-app",
		Price:        biaya_platform,
		Qty:          1,
		Name:         "Biaya Aplikasi",
		MerchantName: "Platform",
		Category:     "fee",
	})
	total += biaya_platform

	// Defensive: Add item prices to total
	for i := 0; i < len(dataTransaksi); i++ {
		total += dataTransaksi[i].HargaBarang
	}

	// Defensive: Validate total amount
	if total <= 0 {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Total pembayaran tidak valid",
		}
	}

	fmt.Println("[TRACE] Buat SnapRequest")
	SnapRequest := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  PaymentCode,
			GrossAmt: total,
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName:    "Username : " + model.Username,
			LName:    "Nama : " + model.Nama,
			Email:    model.Email,
			Phone:    data.AlamatInformation.NomorTelephone,
			BillAddr: &AlamatPengguna,
			ShipAddr: &AlamatPengguna,
		},
		Items:           &hasil,
		EnabledPayments: PM,
	}

	fmt.Println("[TRACE] FormattingTransaksi sukses, lanjut ke ValidateTransaksi()")

	SnapResponse, SnapStatus := ValidateTransaksi(SnapRequest)
	if !SnapStatus {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal memvalidasi transaksi dengan payment gateway",
		}
	}

	// Defensive: Validate SnapResponse
	if SnapResponse == nil {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Response dari payment gateway tidak valid",
		}
	}

	if SnapResponse.Token == "" {
		_ = BatalCheckoutUser(data.DataCheckout, db)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Token pembayaran tidak diterima",
		}
	}

	fmt.Printf("[TRACE] SnapResponse Token: %s\n", SnapResponse.Token)
	fmt.Printf("[TRACE] SnapResponse RedirectURL: %s\n", SnapResponse.RedirectURL)

	fmt.Println("[TRACE] Selesai SnapTransaksi, return response ke client")
	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.SnapTransaksi{
			SnapTransaksi: &snap.Response{
				Token:       SnapResponse.Token,
				RedirectURL: "/",
				StatusCode:  "Berhasil",
			},
			DataCheckout:  data.DataCheckout.DataResponse,
			DataTransaksi: dataTransaksi,
			DataAlamat:    data.AlamatInformation,
		},
	}
}

func BatalTransaksi(ctx context.Context, data response_transaction_pengguna.SnapTransaksi, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "BatalTransaksi"

	var total_varian int64 = 0
	for i := 0; i < len(data.DataCheckout); i++ {
		total_varian += int64(data.DataCheckout[i].Dipesan)
	}

	var varianIds []int64 = make([]int64, 0, total_varian)
	var idkategori map[int64]int64 = make(map[int64]int64, len(data.DataCheckout))

	for i := 0; i < len(data.DataCheckout); i++ {
		idkategori[data.DataCheckout[i].IdKategoriBarang] = int64(data.DataCheckout[i].Dipesan)
		if err := db.Read.WithContext(ctx).Model(&models.VarianBarang{}).Select("id").Where(&models.VarianBarang{
			IdBarangInduk: data.DataCheckout[i].IdBarangInduk,
			IdKategori:    data.DataCheckout[i].IdKategoriBarang,
			Status:        barang_enums.Dipesan,
			HoldBy:        data.DataCheckout[i].IDUser,
		}).Limit(int(data.DataCheckout[i].Dipesan)).Scan(&varianIds).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal coba hubungi customer service",
			}
		}
	}

	err := db.Write.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VarianBarang{}).
			Where("id IN ?", varianIds).
			Updates(map[string]interface{}{
				"status":        barang_enums.Ready,
				"hold_by":       0,
				"holder_entity": "",
			}).Error; err != nil {
			return err
		}

		for ind, jumlah := range idkategori {
			if err := tx.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: ind,
			}).Update("stok", gorm.Expr("stok + ?", jumlah)).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponseBatalTransaksi{
				Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponseBatalTransaksi{
			Message: "Transaksi berhasil dibatalkan.",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Lock Transaksi VA
// Befungsi saat sebuah transaksi sudah di bayar, setelah transaksi di bayar maka fungsi
// lock transaksi akan menjalankan rentetan yang perlu di jalankan ke db utama sesuai dengan
// jenis pembayaran yang dilakukan oleh pengguna disini adalah VA (virtual account)
// ////////////////////////////////////////////////////////////////////////////////////

func LockTransaksiVa(data PayloadLockTransaksiVa, db *config.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "LockTransaksiVa"

	for i := 0; i < len(data.DataHold); i++ {
		if data.DataHold[i].IDSeller == 0 || data.DataHold[i].IDUser == 0 || data.DataHold[i].IdBarangInduk == 0 {
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Payload: response_transaction_pengguna.ResponseLockTransaksi{
					Message: "Data keranjang tidak valid.",
				},
			}
		}
	}

	bank, err_p := payment_gateaway.ParseVirtualAccount(data.PaymentResult)
	if err_p != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Server sedang sibuk coba lagi lain waktu",
		}
	}

	var (
		resp payment_in_va.Response
	)

	d, err_m := json.Marshal(data.PaymentResult)
	if err_m != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Server sedang sibuk coba lagi lain waktu",
		}
	}

	switch bank {
	case "bca":
		var obj payment_in_va.BcaVirtualAccountResponse
		if err := json.Unmarshal(d, &obj); err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Server sedang sibuk coba lagi lain waktu",
			}
		}
		resp = &obj

	case "bni":
		var obj payment_in_va.BniVirtualAccountResponse
		if err := json.Unmarshal(d, &obj); err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Server sedang sibuk coba lagi lain waktu",
			}
		}
		resp = &obj

	case "bri":
		var obj payment_in_va.BriVirtualAccountResponse
		if err := json.Unmarshal(d, &obj); err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Server sedang sibuk coba lagi lain waktu",
			}
		}
		resp = &obj

	case "permata":
		var obj payment_in_va.PermataVirtualAccount
		if err := json.Unmarshal(d, &obj); err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Server sedang sibuk coba lagi lain waktu",
			}
		}
		resp = &obj

	default:
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Va Tidak Dikenali",
		}
	}

	pembayaran, ok := resp.Pembayaran()
	if !ok {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Va Tidak Dikenali",
		}
	}

	pembayaran.IdPengguna = data.DataHold[0].IDUser
	var transaksi_save []models.Transaksi = make([]models.Transaksi, 0, len(data.DataHold))

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pembayaran).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			var kategori models.KategoriBarang
			if err := db.Read.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: data.DataHold[i].IdKategoriBarang,
			}).Limit(1).Take(&kategori).Error; err != nil {
				return err
			}
			beratTotalKg := int32(kategori.BeratGram) * int32(data.DataHold[i].Dipesan) / 1000
			if beratTotalKg == 0 {
				beratTotalKg = 1
			}

			var kendaraan string = ""

			if beratTotalKg <= 10 {
				kendaraan = jenis_kendaraan_kurir.Motor
			}

			if beratTotalKg <= 20 {
				kendaraan = jenis_kendaraan_kurir.Mobil
			}

			if beratTotalKg > 20 {
				kendaraan = jenis_kendaraan_kurir.Pickup
			}

			transaksi_save = append(transaksi_save, models.Transaksi{
				IdPengguna:          data.DataHold[i].IDUser,
				IdSeller:            data.DataHold[i].IDSeller,
				IdBarangInduk:       int64(data.DataHold[i].IdBarangInduk),
				IdAlamatGudang:      data.DataHold[i].IdAlamatGudang,
				IdAlamatEkspedisi:   data.DataTransaksi[i].IdAlamatEkspedisi,
				IdKategoriBarang:    data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna:    data.IdAlamatUser,
				IdPembayaran:        pembayaran.ID,
				JenisPengiriman:     data.JenisLayananKurir,
				KendaraanPengiriman: kendaraan,
				JarakTempuh:         strconv.FormatFloat(data.DataTransaksi[i].Jarak, 'f', 2, 64),
				BeratTotalKg:        int16(beratTotalKg),
				KodeOrderSistem:     pembayaran.KodeOrderSistem,
				Status:              transaksi_enums.Dibayar,
				DibatalkanOleh:      nil,
				KuantitasBarang:     int32(data.DataHold[i].Dipesan),
				IsEkspedisi:         data.DataTransaksi[i].IsEkspedisi,
				SellerPaid:          data.DataTransaksi[i].HargaBarang,
				KurirPaid:           data.DataTransaksi[i].HargaBerat + data.DataTransaksi[i].HargaJarak,
				EkspedisiPaid:       data.DataTransaksi[i].HargaEkspedisi,
				Total:               data.DataTransaksi[i].TotalTagihan,
			})
		}

		if err := tx.CreateInBatches(&transaksi_save, len(transaksi_save)).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			if err := tx.Model(&models.VarianBarang{}).Where(&models.VarianBarang{
				IdBarangInduk: data.DataHold[i].IdBarangInduk,
				IdKategori:    data.DataHold[i].IdKategoriBarang,
				HoldBy:        data.DataHold[i].IDUser,
				HolderEntity:  entity_enums.Pengguna,
				Status:        "Dipesan",
			}).Updates(&models.VarianBarang{
				Status:      "Terjual",
				IdTransaksi: transaksi_save[i].ID,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		fmt.Printf("[ERROR] Transaction rollback | Err=%v\n", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponseLockTransaksi{
				Message: "Terjadi kesalahan pada server. Silakan coba lagi nanti.",
			},
		}
	}

	go func(Dt []models.Transaksi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()
		for _, t := range Dt {
			thresholdPengguna := sot_threshold.PenggunaThreshold{
				IdPengguna: t.IdPengguna,
			}

			if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke pengguna threshold")
			}

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(t.IdSeller),
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke seller threshold")
			}

			thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
				IdBarangInduk: t.IdBarangInduk,
			}

			if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke varang induk threshold")
			}

			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: t.IdKategoriBarang,
			}

			if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke kategori barang threshold")
			}

			thresholdAlamatPengguna := sot_threshold.AlamatPenggunaThreshold{}
			if t.IdAlamatPengguna != 0 {
				thresholdAlamatPengguna.IdAlamatPengguna = t.IdAlamatPengguna

				if err := thresholdAlamatPengguna.Increment(konteks, Trh, stsk_alamat_pengguna.Transaksi); err != nil {
					fmt.Println("Gagal incr count transaksi ke alamat pengguna threshold")
				}
			}

			thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
				IdAlamatGudang: t.IdAlamatGudang,
			}

			if err := thresholdAlamatGudang.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke alamat gudang threshold")
			}

			thresholdAlamatEkspedisi := sot_threshold.AlamatEkspedisiThreshold{}
			if t.IdAlamatEkspedisi != 0 {
				thresholdAlamatEkspedisi.IdAlamatEkspedisi = t.IdAlamatEkspedisi

				if err := thresholdAlamatEkspedisi.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
					fmt.Println("Gagal incr count transaksi ke seller threshold")
				}

			}

			thresholdPembayaran := sot_threshold.PembayaranThreshold{
				IdPembayaran: t.IdPembayaran,
			}

			if err := thresholdPembayaran.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke pembayaran threshold")
			}

			newTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(t).SetTableName(t.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newTransaksiPublish); err != nil {
				fmt.Println("Gagal publish transaksi baru ke message broker")
			}

		}
	}(transaksi_save, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponseLockTransaksi{
			Message: "Transaksi berhasil dikunci.",
		},
	}
}

func PaidFailedTransaksiVa(data PayloadPaidFailedTransaksiVa, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "PaidFailedTransaksiVa"

	bank, err_p := payment_gateaway.ParseVirtualAccount(data.PaymentResult)
	if err_p != nil {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			},
		}
	}

	raw, err_m := json.Marshal(data.PaymentResult)
	if err_m != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			},
		}
	}

	var resp payment_in_va.Response

	switch bank {
	case "bca":
		var obj payment_in_va.BcaVirtualAccountResponse
		if err := json.Unmarshal(raw, &obj); err != nil {
			return &response.ResponseForm{Status: http.StatusBadRequest, Services: services, Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			}}
		}
		resp = &obj

	case "bni":
		var obj payment_in_va.BniVirtualAccountResponse
		if err := json.Unmarshal(raw, &obj); err != nil {
			return &response.ResponseForm{Status: http.StatusBadRequest, Services: services, Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			}}
		}
		resp = &obj

	case "bri":
		var obj payment_in_va.BriVirtualAccountResponse
		if err := json.Unmarshal(raw, &obj); err != nil {
			return &response.ResponseForm{Status: http.StatusBadRequest, Services: services, Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			}}
		}
		resp = &obj

	case "permata":
		var obj payment_in_va.PermataVirtualAccount
		if err := json.Unmarshal(raw, &obj); err != nil {
			return &response.ResponseForm{Status: http.StatusBadRequest, Services: services, Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			}}
		}
		resp = &obj

	default:
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Bank tidak dikenali",
			},
		}
	}

	if resp == nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengenali virtual account",
			},
		}
	}

	standard_response, ok := resp.Pembayaran()
	if !ok {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: "Gagal mengambil standard response",
			},
		}
	}

	standard_response.IdPengguna = data.DataHold[0].IDUser

	err := db.Write.Transaction(func(tx *gorm.DB) error {

		if err := tx.Create(&standard_response).Error; err != nil {
			return fmt.Errorf("gagal menyimpan PaidFailed: %w", err)
		}

		if standard_response.ID == 0 {
			return fmt.Errorf("id PaidFailed tidak ditemukan")
		}

		for i := range data.DataHold {
			tf := models.TransaksiFailed{
				IdPembayaran:     standard_response.ID,
				IdPengguna:       data.DataHold[i].IDUser,
				IdSeller:         data.DataHold[i].IDSeller,
				IdBarangInduk:    data.DataHold[i].IdBarangInduk,
				IdKategoriBarang: data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna: data.IdAlamatUser,
				Catatan:          data.DataHold[i].Message,
				KuantitasBarang:  data.DataHold[i].Dipesan,
				Total:            int64(data.DataHold[i].Dipesan) * int64(data.DataHold[i].HargaKategori),
				JenisPengiriman:  data.JenisLayananKurir,
			}

			if err := tx.Create(&tf).Error; err != nil {
				return fmt.Errorf("gagal menyimpan transaksi ke-%d: %w", i+1, err)
			}
		}

		return nil
	})

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: fmt.Sprintf("Transaksi gagal: %v", err),
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
			Message: "Berhasil",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Lock Transaksi Wallet
// Befungsi saat sebuah transaksi sudah di bayar, setelah transaksi di bayar maka fungsi
// lock transaksi akan menjalankan rentetan yang perlu di jalankan ke db utama sesuai dengan
// jenis pembayaran yang dilakukan oleh pengguna disini adalah Wallet
// ////////////////////////////////////////////////////////////////////////////////////

func LockTransaksiWallet(data PayloadLockTransaksiWallet, db *config.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "LockTransaksiWallet"

	for _, keranjang := range data.DataHold {
		if keranjang.IDSeller == 0 && keranjang.IDUser == 0 && keranjang.IdBarangInduk == 0 {
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Payload: response_transaction_pengguna.ResponseLockTransaksi{
					Message: "Data keranjang tidak valid.",
				},
			}
		}
	}

	pembayaran, ok := data.PaymentResult.Pembayaran()
	if !ok {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server terganggu akan dialihkan ke failed_transaksi",
		}
	}

	pembayaran.IdPengguna = data.DataHold[0].IDUser
	var transaksi_save []models.Transaksi = make([]models.Transaksi, 0, len(data.DataHold))

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pembayaran).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			var kategori models.KategoriBarang
			if err := db.Read.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: data.DataHold[i].IdKategoriBarang,
			}).Limit(1).Take(&kategori).Error; err != nil {
				return err
			}

			beratTotalKg := int32(kategori.BeratGram) * int32(data.DataHold[i].Dipesan) / 1000
			if beratTotalKg == 0 {
				beratTotalKg = 1
			}

			var kendaraan string = ""

			if beratTotalKg <= 10 {
				kendaraan = jenis_kendaraan_kurir.Motor
			}

			if beratTotalKg <= 20 {
				kendaraan = jenis_kendaraan_kurir.Mobil
			}

			if beratTotalKg > 20 {
				kendaraan = jenis_kendaraan_kurir.Pickup
			}

			transaksi_save = append(transaksi_save, models.Transaksi{
				IdPengguna:          data.DataHold[i].IDUser,
				IdSeller:            data.DataHold[i].IDSeller,
				IdBarangInduk:       int64(data.DataHold[i].IdBarangInduk),
				IdAlamatGudang:      data.DataHold[i].IdAlamatGudang,
				IdAlamatEkspedisi:   data.DataTransaksi[i].IdAlamatEkspedisi,
				IdKategoriBarang:    data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna:    data.IdAlamatUser,
				IdPembayaran:        pembayaran.ID,
				JenisPengiriman:     data.JenisLayananKurir,
				KendaraanPengiriman: kendaraan,
				JarakTempuh:         strconv.FormatFloat(data.DataTransaksi[i].Jarak, 'f', 2, 64),
				BeratTotalKg:        int16(beratTotalKg),
				KodeOrderSistem:     pembayaran.KodeOrderSistem,
				Status:              transaksi_enums.Dibayar,
				DibatalkanOleh:      nil,
				KuantitasBarang:     int32(data.DataHold[i].Dipesan),
				IsEkspedisi:         data.DataTransaksi[i].IsEkspedisi,
				SellerPaid:          data.DataTransaksi[i].HargaBarang,
				KurirPaid:           data.DataTransaksi[i].HargaBerat + data.DataTransaksi[i].HargaJarak,
				EkspedisiPaid:       data.DataTransaksi[i].HargaEkspedisi,
				Total:               data.DataTransaksi[i].TotalTagihan,
			})
		}

		if err := tx.CreateInBatches(&transaksi_save, len(transaksi_save)).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			if err := tx.Model(&models.VarianBarang{}).Where(&models.VarianBarang{
				IdBarangInduk: data.DataHold[i].IdBarangInduk,
				IdKategori:    data.DataHold[i].IdKategoriBarang,
				HoldBy:        data.DataHold[i].IDUser,
				HolderEntity:  entity_enums.Pengguna,
				Status:        "Dipesan",
			}).Updates(&models.VarianBarang{
				Status:      "Terjual",
				IdTransaksi: transaksi_save[i].ID,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		fmt.Printf("[ERROR] Transaction rollback | Err=%v\n", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Berhasil",
		}
	}

	go func(Dt []models.Transaksi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()
		for _, t := range Dt {
			thresholdPengguna := sot_threshold.PenggunaThreshold{
				IdPengguna: t.IdPengguna,
			}

			if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke pengguna threshold")
			}

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(t.IdSeller),
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke seller threshold")
			}

			thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
				IdBarangInduk: t.IdBarangInduk,
			}

			if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke varang induk threshold")
			}

			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: t.IdKategoriBarang,
			}

			if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke kategori barang threshold")
			}

			thresholdAlamatPengguna := sot_threshold.AlamatPenggunaThreshold{}
			if t.IdAlamatPengguna != 0 {
				thresholdAlamatPengguna.IdAlamatPengguna = t.IdAlamatPengguna

				if err := thresholdAlamatPengguna.Increment(konteks, Trh, stsk_alamat_pengguna.Transaksi); err != nil {
					fmt.Println("Gagal incr count transaksi ke alamat pengguna threshold")
				}
			}

			thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
				IdAlamatGudang: t.IdAlamatGudang,
			}

			if err := thresholdAlamatGudang.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke alamat gudang threshold")
			}

			thresholdAlamatEkspedisi := sot_threshold.AlamatEkspedisiThreshold{}
			if t.IdAlamatEkspedisi != 0 {
				thresholdAlamatEkspedisi.IdAlamatEkspedisi = t.IdAlamatEkspedisi

				if err := thresholdAlamatEkspedisi.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
					fmt.Println("Gagal incr count transaksi ke seller threshold")
				}

			}

			thresholdPembayaran := sot_threshold.PembayaranThreshold{
				IdPembayaran: t.IdPembayaran,
			}

			if err := thresholdPembayaran.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke pembayaran threshold")
			}

			newTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(t).SetTableName(t.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newTransaksiPublish); err != nil {
				fmt.Println("Gagal publish transaksi baru ke message broker")
			}

		}
	}(transaksi_save, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponseLockTransaksi{
			Message: "Transaksi berhasil dikunci.",
		},
	}
}

func PaidFailedTransaksiWallet(data PayloadPaidFailedTransaksiWallet, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "PaidFailedTransaksiWallet"

	var resp payment_in_wallet.Response = &data.PaymentResult
	standard_response, _ := resp.Pembayaran()

	standard_response.IdPengguna = data.DataHold[0].IDUser
	var TransaksiGagalBatch []models.TransaksiFailed = make([]models.TransaksiFailed, 0, len(data.DataHold))
	// --- Jalankan transaksi database ---
	err := db.Write.Transaction(func(tx *gorm.DB) error {
		// Simpan ke PaidFailed
		if err := tx.Create(&standard_response).Error; err != nil {
			return fmt.Errorf("gagal menyimpan PaidFailed: %w", err)
		}

		if standard_response.ID == 0 {
			return fmt.Errorf("id PaidFailed tidak ditemukan")
		}

		// Simpan TransaksiFailed per item
		for i := range data.DataHold {
			TransaksiGagalBatch = append(TransaksiGagalBatch, models.TransaksiFailed{
				IdPembayaran:     standard_response.ID,
				IdPengguna:       data.DataHold[i].IDUser,
				IdSeller:         data.DataHold[i].IDSeller,
				IdBarangInduk:    data.DataHold[i].IdBarangInduk,
				IdKategoriBarang: data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna: data.IdAlamatUser,
				Catatan:          data.DataHold[i].Message,
				KuantitasBarang:  data.DataHold[i].Dipesan,
				Total:            int64(data.DataHold[i].Dipesan) * int64(data.DataHold[i].HargaKategori),
			})
		}

		if err := tx.CreateInBatches(&TransaksiGagalBatch, len(TransaksiGagalBatch)).Error; err != nil {
			return fmt.Errorf("gagal menyimpan transaksi")
		}

		return nil
	})

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: fmt.Sprintf("Transaksi gagal: %v", err),
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
			Message: "Berhasil",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Lock Transaksi Gerai
// Befungsi saat sebuah transaksi sudah di bayar, setelah transaksi di bayar maka fungsi
// lock transaksi akan menjalankan rentetan yang perlu di jalankan ke db utama sesuai dengan
// jenis pembayaran yang dilakukan oleh pengguna disini adalah Gerai
// ////////////////////////////////////////////////////////////////////////////////////

func LockTransaksiGerai(data PayloadLockTransaksiGerai, db *config.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	services := "LockTransaksiGerai"

	for _, keranjang := range data.DataHold {
		if keranjang.IDSeller == 0 && keranjang.IDUser == 0 && keranjang.IdBarangInduk == 0 {
			return &response.ResponseForm{
				Status:   http.StatusBadRequest,
				Services: services,
				Payload: response_transaction_pengguna.ResponseLockTransaksi{
					Message: "Data keranjang tidak valid.",
				},
			}
		}
	}

	var (
		resp payment_in_gerai.Response
	)
	resp = &data.PaymentResult

	pembayaran, ok := resp.Pembayaran()
	if !ok {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal server terganggu akan dialihkan ke pembayaran dan transaksi failed",
		}
	}

	//
	// Sanitasi Id Pengguna
	//
	pembayaran.IdPengguna = data.DataHold[0].IDUser
	var transaksi_save []models.Transaksi = make([]models.Transaksi, 0, len(data.DataHold))

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pembayaran).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			var kategori models.KategoriBarang
			if err := db.Read.Model(&models.KategoriBarang{}).Where(&models.KategoriBarang{
				ID: data.DataHold[i].IdKategoriBarang,
			}).Limit(1).Take(&kategori).Error; err != nil {
				return err
			}

			beratTotalKg := int32(kategori.BeratGram) * int32(data.DataHold[i].Dipesan) / 1000
			if beratTotalKg == 0 {
				beratTotalKg = 1
			}

			var kendaraan string = ""

			if beratTotalKg <= 10 {
				kendaraan = jenis_kendaraan_kurir.Motor
			}

			if beratTotalKg <= 20 {
				kendaraan = jenis_kendaraan_kurir.Mobil
			}

			if beratTotalKg > 20 {
				kendaraan = jenis_kendaraan_kurir.Pickup
			}

			transaksi_save = append(transaksi_save, models.Transaksi{
				IdPengguna:          data.DataHold[i].IDUser,
				IdSeller:            data.DataHold[i].IDSeller,
				IdBarangInduk:       int64(data.DataHold[i].IdBarangInduk),
				IdAlamatGudang:      data.DataHold[i].IdAlamatGudang,
				IdAlamatEkspedisi:   data.DataTransaksi[i].IdAlamatEkspedisi,
				IdKategoriBarang:    data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna:    data.IdAlamatUser,
				IdPembayaran:        pembayaran.ID,
				JenisPengiriman:     data.JenisLayananKurir,
				KendaraanPengiriman: kendaraan,
				JarakTempuh:         strconv.FormatFloat(data.DataTransaksi[i].Jarak, 'f', 2, 64),
				BeratTotalKg:        int16(beratTotalKg),
				KodeOrderSistem:     pembayaran.KodeOrderSistem,
				Status:              transaksi_enums.Dibayar,
				DibatalkanOleh:      nil,
				KuantitasBarang:     int32(data.DataHold[i].Dipesan),
				IsEkspedisi:         data.DataTransaksi[i].IsEkspedisi,
				SellerPaid:          data.DataTransaksi[i].HargaBarang,
				KurirPaid:           data.DataTransaksi[i].HargaBerat + data.DataTransaksi[i].HargaJarak,
				EkspedisiPaid:       data.DataTransaksi[i].HargaEkspedisi,
				Total:               data.DataTransaksi[i].TotalTagihan,
			})
		}

		if err := tx.CreateInBatches(&transaksi_save, len(transaksi_save)).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			if err := tx.Model(&models.VarianBarang{}).Where(&models.VarianBarang{
				IdBarangInduk: data.DataHold[i].IdBarangInduk,
				IdKategori:    data.DataHold[i].IdKategoriBarang,
				HoldBy:        data.DataHold[i].IDUser,
				HolderEntity:  entity_enums.Pengguna,
				Status:        "Dipesan",
			}).Updates(&models.VarianBarang{
				Status:      "Terjual",
				IdTransaksi: transaksi_save[i].ID,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		fmt.Printf("[ERROR] Transaction rollback | Err=%v\n", err)
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Berhasil",
		}
	}

	go func(Dt []models.Transaksi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ctx_t := context.Background()
		konteks, cancel := context.WithTimeout(ctx_t, time.Second*5)
		defer cancel()
		for _, t := range Dt {
			thresholdPengguna := sot_threshold.PenggunaThreshold{
				IdPengguna: t.IdPengguna,
			}

			if err := thresholdPengguna.Increment(konteks, Trh, stsk_pengguna.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke pengguna threshold")
			}

			thresholdSeller := sot_threshold.SellerThreshold{
				IdSeller: int64(t.IdSeller),
			}

			if err := thresholdSeller.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke seller threshold")
			}

			thresholdBarangInduk := sot_threshold.BarangIndukThreshold{
				IdBarangInduk: t.IdBarangInduk,
			}

			if err := thresholdBarangInduk.Increment(konteks, Trh, stsk_baranginduk.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke varang induk threshold")
			}

			thresholdKategoriBarang := sot_threshold.KategoriBarangThreshold{
				IdKategoriBarang: t.IdKategoriBarang,
			}

			if err := thresholdKategoriBarang.Increment(konteks, Trh, stsk_kategori_barang.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke kategori barang threshold")
			}

			thresholdAlamatPengguna := sot_threshold.AlamatPenggunaThreshold{}
			if t.IdAlamatPengguna != 0 {
				thresholdAlamatPengguna.IdAlamatPengguna = t.IdAlamatPengguna

				if err := thresholdAlamatPengguna.Increment(konteks, Trh, stsk_alamat_pengguna.Transaksi); err != nil {
					fmt.Println("Gagal incr count transaksi ke alamat pengguna threshold")
				}
			}

			thresholdAlamatGudang := sot_threshold.AlamatGudangThreshold{
				IdAlamatGudang: t.IdAlamatGudang,
			}

			if err := thresholdAlamatGudang.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke alamat gudang threshold")
			}

			thresholdAlamatEkspedisi := sot_threshold.AlamatEkspedisiThreshold{}
			if t.IdAlamatEkspedisi != 0 {
				thresholdAlamatEkspedisi.IdAlamatEkspedisi = t.IdAlamatEkspedisi

				if err := thresholdAlamatEkspedisi.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
					fmt.Println("Gagal incr count transaksi ke seller threshold")
				}

			}

			thresholdPembayaran := sot_threshold.PembayaranThreshold{
				IdPembayaran: t.IdPembayaran,
			}

			if err := thresholdPembayaran.Increment(konteks, Trh, stsk_seller.Transaksi); err != nil {
				fmt.Println("Gagal incr count transaksi ke pembayaran threshold")
			}

			newTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(t).SetTableName(t.TableName())
			if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, newTransaksiPublish); err != nil {
				fmt.Println("Gagal publish transaksi baru ke message broker")
			}

		}
	}(transaksi_save, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponseLockTransaksi{
			Message: "Transaksi berhasil dikunci.",
		},
	}
}

func PaidFailedTransaksiGerai(data PayloadPaidFailedTransaksiGerai, db *config.InternalDBReadWriteSystem) *response.ResponseForm {
	services := "PaidFailedTransaksiGerai"

	var resp payment_in_gerai.Response = &data.PaymentResult
	standard_response, _ := resp.Pembayaran()

	standard_response.IdPengguna = data.DataHold[0].IDUser

	// --- Jalankan transaksi database ---
	err := db.Write.Transaction(func(tx *gorm.DB) error {
		// Simpan ke PaidFailed
		if err := tx.Create(&standard_response).Error; err != nil {
			return fmt.Errorf("gagal menyimpan PaidFailed: %w", err)
		}

		if standard_response.ID == 0 {
			return fmt.Errorf("id PaidFailed tidak ditemukan")
		}

		// Simpan TransaksiFailed per item
		for i := range data.DataHold {
			tf := models.TransaksiFailed{
				IdPembayaran:     standard_response.ID,
				IdPengguna:       data.DataHold[i].IDUser,
				IdSeller:         data.DataHold[i].IDSeller,
				IdBarangInduk:    data.DataHold[i].IdBarangInduk,
				IdKategoriBarang: data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna: data.IdAlamatUser,
				Catatan:          data.DataHold[i].Message,
				KuantitasBarang:  data.DataHold[i].Dipesan,
				Total:            int64(data.DataHold[i].Dipesan) * int64(data.DataHold[i].HargaKategori),
			}

			if err := tx.Create(&tf).Error; err != nil {
				return fmt.Errorf("gagal menyimpan transaksi ke-%d: %w", i+1, err)
			}
		}

		return nil
	})

	if err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
				Message: fmt.Sprintf("Transaksi gagal: %v", err),
			},
		}
	}

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Payload: response_transaction_pengguna.ResponsePaidFailedTransaksi{
			Message: "Berhasil",
		},
	}
}
