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
	barang_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/barang"
	enums_barang_di_diskon "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/barang_di_diskon"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/jenis_kendaraan_kurir"
	transaksi_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/transaksi"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
	stsk_alamat_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/alamat_pengguna"
	stsk_baranginduk "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/barang_induk"
	stsk_kategori_barang "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kategori_barang"
	stsk_kurir "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/kurir"
	stsk_pengguna "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengguna"
	stsk_pengiriman "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/pengiriman"
	stsk_seller "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold/seeders/nama_kolom/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/transaction_services/response_transaction_pengguna"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

func CheckoutBarangUser(ctx context.Context, data PayloadCheckoutBarang, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "CheckoutBarangUser"
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
	dataLen := len(data.DataCheckout)
	idKeranjang := make([]int64, 0, dataLen)
	KeranjangData := make([]sot_models.Keranjang, 0, dataLen)

	// Loop menggunakan dataLen agar aman, tidak baca len() berulang
	for i := 0; i < dataLen; i++ {
		// tambahan defensive: pastikan index valid sebelum akses
		if i < 0 || i >= dataLen {
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

	if err := db.Read.WithContext(ctx).Model(&sot_models.Keranjang{}).Where("id IN ?", idKeranjang).Limit(dataLen).Take(&KeranjangData).Error; err != nil {
		fmt.Println("Gagal mendapatkan data seluruh keranjang")
	}

	responseData := make([]response_transaction_pengguna.CheckoutData, 0, dataLen)
	varianUpdates := make([]int64, 0, totalDipesan)
	kategoriUpdates := make(map[int32]int32, dataLen)
	BarangInduk := make(map[int64]sot_models.BarangInduk, dataLen)
	KategoriBarang := make(map[int64]sot_models.KategoriBarang, dataLen)
	NamaSeller := make(map[int64]string, dataLen)

	for i := 0; i < dataLen; i++ {
		if i < 0 || i >= dataLen {
			continue
		}

		var idsVarianStok []int64 = make([]int64, 0, data.DataCheckout[i].Jumlah)
		if err := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").Where(&sot_models.VarianBarang{
			IdBarangInduk: data.DataCheckout[i].IdBarangInduk,
			IdKategori:    data.DataCheckout[i].IdKategori,
			Status:        barang_enums.Ready,
		}).Limit(int(data.DataCheckout[i].Jumlah)).Scan(&idsVarianStok).Error; err != nil {
			return &response.ResponseForm{
				Status:   http.StatusInternalServerError,
				Services: services,
				Message:  "Gagal server sedang sibuk coba lagi lain waktu",
			}
		}

		if len(idsVarianStok) < int(data.DataCheckout[i].Jumlah) {
			return &response.ResponseForm{
				Status:   http.StatusUnauthorized,
				Services: services,
				Message:  "Gagal barang lebih sedikit dibanding yang kamu pesan",
			}
		}

		varianUpdates = append(varianUpdates, idsVarianStok...)

		if BarangInduk[int64(data.DataCheckout[i].IdBarangInduk)].NamaBarang == "" {
			barang := sot_models.BarangInduk{}

			if err := db.Read.WithContext(ctx).Model(&sot_models.BarangInduk{}).Select("nama_barang", "id_seller", "jenis_barang").Where(&sot_models.BarangInduk{
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
			var kategori sot_models.KategoriBarang = sot_models.KategoriBarang{Nama: ""}
			if err := db.Read.Model(&sot_models.KategoriBarang{}).Select("nama", "harga", "stok", "id_barang_induk", "id_alamat_gudang", "berat_gram").
				Where(&sot_models.KategoriBarang{ID: data.DataCheckout[i].IdKategori}).Limit(1).Scan(&kategori).Error; err != nil {
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
			if err := db.Read.Model(&sot_models.Seller{}).Select("nama").
				Where(&sot_models.Seller{ID: data.DataCheckout[i].IdSeller}).
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

		var IdDiskon int64 = 0
		if err := db.Read.WithContext(ctx).Model(&sot_models.BarangDiDiskon{}).Select("id_diskon").Where(&sot_models.BarangDiDiskon{
			SellerId:         data.DataCheckout[i].IdSeller,
			IdBarangInduk:    data.DataCheckout[i].IdBarangInduk,
			IdKategoriBarang: data.DataCheckout[i].IdKategori,
			Status:           enums_barang_di_diskon.Applied,
		}).Limit(1).Take(&IdDiskon).Error; err != nil {
			fmt.Println(err)
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
			IdDiskon:         IdDiskon,
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
			if err := tx.Model(&sot_models.VarianBarang{}).
				Where("id IN ?", varianUpdates).
				Updates(&sot_models.VarianBarang{
					Status:       barang_enums.Dipesan,
					HoldBy:       data.IdentitasPengguna.ID,
					HolderEntity: entity_enums.Pengguna,
				}).Error; err != nil {
				return err
			}
		}

		// Update stok kategori secara atomic
		for kategoriID, totalDipesan := range kategoriUpdates {
			if err := tx.Model(&sot_models.KategoriBarang{}).
				Where("id = ? AND stok >= ?", kategoriID, totalDipesan).
				UpdateColumn("stok", gorm.Expr("stok - ?", totalDipesan)).Error; err != nil {
				return err
			}
		}

		// Hapus keranjang menggunakan tx agar konsisten dengan transaksi
		if len(idKeranjang) > 0 {
			if err := tx.WithContext(ctx).Model(&sot_models.Keranjang{}).Where("id IN ?", idKeranjang).Delete(&sot_models.Keranjang{}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	go func(Dk []sot_models.Keranjang, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		var wg sync.WaitGroup
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer cancel()
		for _, k := range Dk {
			wg.Add(1)
			go func(datakeranjang sot_models.Keranjang, t *gorm.DB, p *mb_cud_publisher.Publisher) {
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

				deleteKeranjangPublish := mb_cud_serializer.NewJsonPayload().SetPayload(datakeranjang).SetTableName(datakeranjang.TableName()).SetRole(mb_cud_seeders.Pengguna)
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

func BatalCheckoutUser(data response_transaction_pengguna.ResponseDataCheckout, db *environment.InternalDBReadWriteSystem) *response.ResponseForm {
	const services string = "BatalCheckoutKeranjang"

	var varianIDs []int64
	kategoriUpdates := make(map[int32]int32)

	for _, keranjang := range data.DataResponse {
		var varian_id []int64
		if err := db.Read.Model(&sot_models.VarianBarang{}).
			Where(sot_models.VarianBarang{
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
			if err := tx.Model(&sot_models.VarianBarang{}).
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
			if err := tx.Model(&sot_models.KategoriBarang{}).
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

func SnapTransaksi(ctx context.Context, data PayloadSnapTransaksiRequest, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client) *response.ResponseForm {
	const services string = "SnapTransaksiUser"
	fmt.Println("[TRACE] Start SnapTransaksi")

	model, status := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session)
	if !status {
		return &response.ResponseForm{
			Status:   http.StatusNotFound,
			Services: services,
			Payload:  "Gagal Validasi User Tidak Valid",
		}
	}

	lenData := len(data.DataCheckout.DataResponse)

	if lenData == 0 {
		return &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: services,
			Message:  "Data checkout kosong",
		}
	}

	var sellerTransaction map[int32]sot_models.Seller = make(map[int32]sot_models.Seller, lenData)

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
			var seller sot_models.Seller
			if err := db.Read.WithContext(ctx).Model(&sot_models.Seller{}).Where(&sot_models.Seller{
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
		if err := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").Where(&sot_models.VarianBarang{
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

	var KebijakanSistem sot_models.KebijakanSistem
	if err := db.Read.WithContext(ctx).Model(&sot_models.KebijakanSistem{}).Where(&sot_models.KebijakanSistem{
		StatusActive: true,
	}).Limit(1).Take(&KebijakanSistem).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Ada kegagalan perhitungan di server",
		}
	}

	var AlamatGudang map[int64]sot_models.AlamatGudang = make(map[int64]sot_models.AlamatGudang, lenData)
	var dataTransaksi []response_transaction_pengguna.DataTransaksi = make([]response_transaction_pengguna.DataTransaksi, 0, lenData)
	var fee_platform int64 = 0

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
		totalHargaBerat := int64(KebijakanSistem.TarifPerkg) * int64(beratTotal)

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
			var alamat sot_models.AlamatGudang
			if err := db.Read.WithContext(ctx).Model(&sot_models.AlamatGudang{}).Where(&sot_models.AlamatGudang{
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
			if err := db.Read.WithContext(ctx).Model(&sot_models.AlamatEkspedisi{}).Select("id").Where(&sot_models.AlamatEkspedisi{
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

		if Jarak >= float64(KebijakanSistem.JarakMasukEkspedisi) {
			isEkspedisi = true
		}

		// Defensive: Validate delivery service type
		validLayanan := map[string]bool{"reguler": true, "express": true, "instant": true}
		if !validLayanan[data.LayananPengirimanKurir] {
			data.LayananPengirimanKurir = "reguler"
		}

		var Tarif int64 = int64(KebijakanSistem.TarifPengirimanRegulerPerKm)
		switch data.LayananPengirimanKurir {
		case "express":
			if Jarak > float64(KebijakanSistem.MaxJarakKmExpress) {
				data.LayananPengirimanKurir = "reguler"
			}
			Tarif = int64(KebijakanSistem.TarifPengirimanExpressPerKm)
		case "instant":
			if Jarak > float64(KebijakanSistem.MaxJarakKmInstant) {
				data.LayananPengirimanKurir = "reguler"
			}
			Tarif = int64(KebijakanSistem.TarifPengirimanInstantPerKm)
		}

		hargaJarak += Tarif * int64(Jarak)

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

		totalTagihanTransaksi := int64(totalHargapembelian) + totalHargaBerat + hargaJarak + hargaEkspedisi
		if data.DataCheckout.DataResponse[i].IdDiskon != 0 {

			var diskonPersen float32
			if err := db.Read.WithContext(ctx).Model(&sot_models.DiskonProduk{}).Select("diskon_persen").Where(&sot_models.DiskonProduk{
				ID: data.DataCheckout.DataResponse[i].IdDiskon}).Limit(1).Take(&diskonPersen).Error; err != nil {
				return &response.ResponseForm{
					Status:   http.StatusInternalServerError,
					Services: services,
					Message:  "Terjadi kesalahan pada sistem",
				}
			}

			totalTagihanTransaksi = totalTagihanTransaksi - int64(float32(totalTagihanTransaksi)*diskonPersen)
		}

		fee_platform += totalTagihanTransaksi
		datafee := int64(float32(KebijakanSistem.KomisiSistemPerTransaksi) * float32(totalTagihanTransaksi))

		dataTransaksiIterasi := response_transaction_pengguna.DataTransaksi{
			IdAlamatEkspedisi: IdAlamatEkspedisi,
			HargaBarang:       int64(totalHargapembelian),
			HargaBerat:        totalHargaBerat,
			IdDIskon:          data.DataCheckout.DataResponse[i].IdDiskon,
			HargaJarak:        hargaJarak,
			HargaEkspedisi:    hargaEkspedisi,
			IsEkspedisi:       isEkspedisi,
			KomisiSistem:      datafee,
			Jarak:             Jarak,
			TotalTagihan:      totalTagihanTransaksi + datafee,
		}

		dataTransaksi = append(dataTransaksi, dataTransaksiIterasi)
	}

	fee_platform = int64(float32(KebijakanSistem.KomisiSistemPerTransaksi) * float32(fee_platform))

	var harga_kirim int64 = 0
	for i := 0; i < len(dataTransaksi); i++ {
		harga_kirim += dataTransaksi[i].HargaBerat + dataTransaksi[i].HargaEkspedisi + dataTransaksi[i].HargaJarak
	}

	hasil = append(hasil, midtrans.ItemDetails{
		ID:           "fee-courier",
		Price:        harga_kirim,
		Qty:          1,
		Name:         fmt.Sprintf("Biaya Kurir - %s", data.LayananPengirimanKurir),
		MerchantName: "Courier",
		Category:     "fee",
	})
	total := harga_kirim

	// ==== Biaya aplikasi ====
	hasil = append(hasil, midtrans.ItemDetails{
		ID:           "fee-app",
		Price:        fee_platform,
		Qty:          1,
		Name:         "Biaya Aplikasi",
		MerchantName: "Platform",
		Category:     "fee",
	})
	total += fee_platform

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

func BatalTransaksi(ctx context.Context, data response_transaction_pengguna.SnapTransaksi, db *environment.InternalDBReadWriteSystem) *response.ResponseForm {
	const services string = "BatalTransaksi"

	var total_varian int64 = 0
	for i := 0; i < len(data.DataCheckout); i++ {
		total_varian += int64(data.DataCheckout[i].Dipesan)
	}

	var varianIds []int64 = make([]int64, 0, total_varian)
	var idkategori map[int64]int64 = make(map[int64]int64, len(data.DataCheckout))

	for i := 0; i < len(data.DataCheckout); i++ {
		idkategori[data.DataCheckout[i].IdKategoriBarang] = int64(data.DataCheckout[i].Dipesan)
		if err := db.Read.WithContext(ctx).Model(&sot_models.VarianBarang{}).Select("id").Where(&sot_models.VarianBarang{
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
		if err := tx.Model(&sot_models.VarianBarang{}).
			Where("id IN ?", varianIds).
			Updates(map[string]interface{}{
				"status":        barang_enums.Ready,
				"hold_by":       0,
				"holder_entity": "",
			}).Error; err != nil {
			return err
		}

		for ind, jumlah := range idkategori {
			if err := tx.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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

func LockTransaksiVa(data PayloadLockTransaksiVa, db *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "LockTransaksiVa"

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
	var transaksi_save []sot_models.Transaksi = make([]sot_models.Transaksi, 0, len(data.DataHold))

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pembayaran).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			var kategori sot_models.KategoriBarang
			if err := db.Read.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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

			transaksi_save = append(transaksi_save, sot_models.Transaksi{
				IdPengguna:          data.DataHold[i].IDUser,
				IdSeller:            data.DataHold[i].IDSeller,
				IdBarangInduk:       int64(data.DataHold[i].IdBarangInduk),
				IdAlamatGudang:      data.DataHold[i].IdAlamatGudang,
				IdAlamatEkspedisi:   data.DataTransaksi[i].IdAlamatEkspedisi,
				IdKategoriBarang:    data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna:    data.IdAlamatUser,
				IdPembayaran:        pembayaran.ID,
				IdDiskon:            data.DataHold[i].IdDiskon,
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
				SistemPaid:          data.DataTransaksi[i].KomisiSistem,
				EkspedisiPaid:       data.DataTransaksi[i].HargaEkspedisi,
				Total:               data.DataTransaksi[i].TotalTagihan,
			})
		}

		if err := tx.CreateInBatches(&transaksi_save, len(transaksi_save)).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			if err := tx.Model(&sot_models.VarianBarang{}).Where(&sot_models.VarianBarang{
				IdBarangInduk: data.DataHold[i].IdBarangInduk,
				IdKategori:    data.DataHold[i].IdKategoriBarang,
				HoldBy:        data.DataHold[i].IDUser,
				HolderEntity:  entity_enums.Pengguna,
				Status:        "Dipesan",
			}).Updates(&sot_models.VarianBarang{
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

	go func(Dt []sot_models.Transaksi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
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

			newTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(t).SetTableName("LockTransaksiVa").SetRole(mb_cud_seeders.Pengguna)
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

func PaidFailedTransaksiVa(data PayloadPaidFailedTransaksiVa, db *environment.InternalDBReadWriteSystem) *response.ResponseForm {
	const services string = "PaidFailedTransaksiVa"

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
			tf := sot_models.TransaksiFailed{
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

func LockTransaksiWallet(data PayloadLockTransaksiWallet, db *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "LockTransaksiWallet"

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
	var transaksi_save []sot_models.Transaksi = make([]sot_models.Transaksi, 0, len(data.DataHold))

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pembayaran).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			var kategori sot_models.KategoriBarang
			if err := db.Read.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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

			transaksi_save = append(transaksi_save, sot_models.Transaksi{
				IdPengguna:          data.DataHold[i].IDUser,
				IdSeller:            data.DataHold[i].IDSeller,
				IdBarangInduk:       int64(data.DataHold[i].IdBarangInduk),
				IdAlamatGudang:      data.DataHold[i].IdAlamatGudang,
				IdAlamatEkspedisi:   data.DataTransaksi[i].IdAlamatEkspedisi,
				IdKategoriBarang:    data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna:    data.IdAlamatUser,
				IdPembayaran:        pembayaran.ID,
				IdDiskon:            data.DataHold[i].IdDiskon,
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
				SistemPaid:          data.DataTransaksi[i].KomisiSistem,
				EkspedisiPaid:       data.DataTransaksi[i].HargaEkspedisi,
				Total:               data.DataTransaksi[i].TotalTagihan,
			})
		}

		if err := tx.CreateInBatches(&transaksi_save, len(transaksi_save)).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			if err := tx.Model(&sot_models.VarianBarang{}).Where(&sot_models.VarianBarang{
				IdBarangInduk: data.DataHold[i].IdBarangInduk,
				IdKategori:    data.DataHold[i].IdKategoriBarang,
				HoldBy:        data.DataHold[i].IDUser,
				HolderEntity:  entity_enums.Pengguna,
				Status:        "Dipesan",
			}).Updates(&sot_models.VarianBarang{
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

	go func(Dt []sot_models.Transaksi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {

		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
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

			newTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(t).SetTableName("LockTransaksiWallet").SetRole(mb_cud_seeders.Pengguna)
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

func PaidFailedTransaksiWallet(data PayloadPaidFailedTransaksiWallet, db *environment.InternalDBReadWriteSystem) *response.ResponseForm {
	const services string = "PaidFailedTransaksiWallet"

	var resp payment_in_wallet.Response = &data.PaymentResult
	standard_response, _ := resp.Pembayaran()

	standard_response.IdPengguna = data.DataHold[0].IDUser
	var TransaksiGagalBatch []sot_models.TransaksiFailed = make([]sot_models.TransaksiFailed, 0, len(data.DataHold))
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
			TransaksiGagalBatch = append(TransaksiGagalBatch, sot_models.TransaksiFailed{
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

func LockTransaksiGerai(data PayloadLockTransaksiGerai, db *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "LockTransaksiGerai"

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
	var transaksi_save []sot_models.Transaksi = make([]sot_models.Transaksi, 0, len(data.DataHold))

	if err := db.Write.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pembayaran).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			var kategori sot_models.KategoriBarang
			if err := db.Read.Model(&sot_models.KategoriBarang{}).Where(&sot_models.KategoriBarang{
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

			transaksi_save = append(transaksi_save, sot_models.Transaksi{
				IdPengguna:          data.DataHold[i].IDUser,
				IdSeller:            data.DataHold[i].IDSeller,
				IdBarangInduk:       int64(data.DataHold[i].IdBarangInduk),
				IdAlamatGudang:      data.DataHold[i].IdAlamatGudang,
				IdAlamatEkspedisi:   data.DataTransaksi[i].IdAlamatEkspedisi,
				IdKategoriBarang:    data.DataHold[i].IdKategoriBarang,
				IdAlamatPengguna:    data.IdAlamatUser,
				IdPembayaran:        pembayaran.ID,
				IdDiskon:            data.DataHold[i].IdDiskon,
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
				SistemPaid:          data.DataTransaksi[i].KomisiSistem,
				EkspedisiPaid:       data.DataTransaksi[i].HargaEkspedisi,
				Total:               data.DataTransaksi[i].TotalTagihan,
			})
		}

		if err := tx.CreateInBatches(&transaksi_save, len(transaksi_save)).Error; err != nil {
			return err
		}

		for i := 0; i < len(data.DataHold); i++ {
			if err := tx.Model(&sot_models.VarianBarang{}).Where(&sot_models.VarianBarang{
				IdBarangInduk: data.DataHold[i].IdBarangInduk,
				IdKategori:    data.DataHold[i].IdKategoriBarang,
				HoldBy:        data.DataHold[i].IDUser,
				HolderEntity:  entity_enums.Pengguna,
				Status:        "Dipesan",
			}).Updates(&sot_models.VarianBarang{
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

	go func(Dt []sot_models.Transaksi, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
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

			newTransaksiPublish := mb_cud_serializer.NewJsonPayload().SetPayload(t).SetTableName("LockTransaksiGerai").SetRole(mb_cud_seeders.Pengguna)
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

func PaidFailedTransaksiGerai(data PayloadPaidFailedTransaksiGerai, db *environment.InternalDBReadWriteSystem) *response.ResponseForm {
	const services string = "PaidFailedTransaksiGerai"

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
			tf := sot_models.TransaksiFailed{
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

func PenggunaRatingPengirimanKurir(ctx context.Context, data PayloadPenggunaRatingPengirimanKurir, db *environment.InternalDBReadWriteSystem, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) *response.ResponseForm {
	const services string = "PenggunaRatingPengirimanKurir"
	if _, valid := data.IdentitasPengguna.Validating(ctx, db.Read, rds_session); !valid {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data pengguna tidak valid",
		}
	}

	var data_rating int = 0
	if err := db.Read.WithContext(ctx).Model(&sot_models.ReviewPengirimanKurir{}).Select("id").Where(&sot_models.ReviewPengirimanKurir{
		IdPengiriman: data.IdPengiriman,
		IdKurir:      data.IdKurir,
	}).Limit(1).Take(&data_rating).Error; err != nil {
		fmt.Println("Gagal mengambil data rating kurir")
	}

	if data_rating != 0 {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal kamu sudah memberikan rating",
		}
	}

	var jenis_pengiriman string = ""
	if err := db.Read.WithContext(ctx).Model(&sot_models.Pengiriman{}).Select("jenis_pengiriman").Where(&sot_models.Pengiriman{
		ID:      data.IdPengiriman,
		IdKurir: &data.IdKurir,
	}).Limit(1).Take(&data_rating).Error; err != nil {
		fmt.Println("Gagal mengambil data id pengiriman")
	}

	if jenis_pengiriman == "" {
		return &response.ResponseForm{
			Status:   http.StatusUnauthorized,
			Services: services,
			Message:  "Gagal data tidak valid",
		}
	}

	createReviewPengirimanKurir := sot_models.ReviewPengirimanKurir{
		IdPengiriman:    data.IdPengiriman,
		JenisPengiriman: fmt.Sprintf("non ekspedisi - %s", jenis_pengiriman),
		IdRater:         data.IdentitasPengguna.ID,
		RaterEntityType: entity_enums.Pengguna,
		IdKurir:         data.IdKurir,
		Ulasan:          data.Ulasan,
		Rating:          data.Rating,
		CreatedAt:       time.Now(),
	}

	if err := db.Write.WithContext(ctx).Create(&createReviewPengirimanKurir).Error; err != nil {
		return &response.ResponseForm{
			Status:   http.StatusInternalServerError,
			Services: services,
			Message:  "Gagal memasukan data review coba lagi beberapa saat",
		}
	}

	go func(RPK sot_models.ReviewPengirimanKurir, Trh *gorm.DB, publisher *mb_cud_publisher.Publisher) {
		ThresholdKurir := sot_threshold.KurirThreshold{
			ID: RPK.IdKurir,
		}

		ThresholdPengiriman := sot_threshold.PengirimanNonEkspedisiThreshold{
			ID: RPK.IdPengiriman,
		}

		konteks, batal := context.WithTimeout(context.Background(), settings.TimeoutContext)
		defer batal()

		if err := ThresholdKurir.Increment(konteks, Trh, stsk_kurir.ReviewPengirimanKurir); err != nil {
			fmt.Println("Gagal increment threshold Kurir")
		}

		if err := ThresholdPengiriman.Increment(konteks, Trh, stsk_pengiriman.PengirimanRated); err != nil {
			fmt.Println("Gagal increment threshold pengiriman")
		}

		createRatingPengirimanKurirPublish := mb_cud_serializer.NewJsonPayload().SetPayload(RPK).SetTableName(RPK.TableName()).SetRole(mb_cud_seeders.Pengguna)
		if err := mb_cud_publisher.CreatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, createRatingPengirimanKurirPublish); err != nil {
			fmt.Println("Gagal publish create rating pengiriman kurir")
		}

	}(createReviewPengirimanKurir, db.Write, cud_publisher)

	return &response.ResponseForm{
		Status:   http.StatusOK,
		Services: services,
		Message:  "Berhasil menambahkan rating pengiriman kurir",
	}
}
