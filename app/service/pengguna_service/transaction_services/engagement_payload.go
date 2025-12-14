package pengguna_transaction_services

import (
	payment_gerai "github.com/anan112pcmec/Burung-backend-1/app/api/payment_in_midtrans/gerai"
	payment_wallet "github.com/anan112pcmec/Burung-backend-1/app/api/payment_in_midtrans/wallet"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/transaction_services/response_transaction_pengguna"
)

type PendingTransactionModel struct {
	FinishRedirectUrl string `json:"finish_redirect_url"`
	FraudStatus       string `json:"fraud_status"`
	GrossAmout        string `json:"gross_amount"`
	OrderId           string `json:"order_id"`
	PaymentType       string `json:"payment_type"`
	StatusCode        string `json:"status_code"`
	StatusMessage     string `json:"status_message"`
	TransactionId     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	TransactionTime   string `json:"transaction_time"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Checkout Barang Dan Batal Checkout Barang
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadCheckoutBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	DataCheckout      []models.Keranjang                 `json:"data_checkout"`
	JenisLayananKurir string                             `json:"jenis_layanan_kurir_checkout_barang"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Snap Transaksi
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadSnapTransaksiRequest struct {
	IdentitasPengguna      identity_pengguna.IdentityPengguna                 `json:"identitas_pengguna"`
	AlamatInformation      models.AlamatPengguna                              `json:"data_alamat_transaksi"`
	DataCheckout           response_transaction_pengguna.ResponseDataCheckout `json:"data_transaksi_item"`
	PaymentMethod          string                                             `json:"pilihan_pembayaran"`
	LayananPengirimanKurir string                                             `json:"layanan_pengiriman_kurir"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Snap Transaksi
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// type PayloadPendingTransaksi struct {
// 	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"data_identitas_pengguna"`
// 	DataPending       PendingTransactionModel            `json:"data_pending_transaksi"`
// }

// type PayloadCallPendingTransaksi struct {
// 	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"data_identitas_pengguna"`
// 	PendingKey        string                             `json:"data_key_pending"`
// }

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Lock Transaksi VA
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadLockTransaksiVa struct {
	DataHold          []response_transaction_pengguna.CheckoutData  `json:"checkout_data_hold"`
	PaymentResult     any                                           `json:"payment_result"`
	DataTransaksi     []response_transaction_pengguna.DataTransaksi `json:"data_jarak"`
	IdAlamatUser      int64                                         `json:"alamat_data_hold"`
	JenisLayananKurir string                                        `json:"jenis_layanan_kurir_keranjang"`
}

type PayloadPaidFailedTransaksiVa struct {
	DataHold          []response_transaction_pengguna.CheckoutData `json:"checkout_data_hold"`
	PaymentResult     any                                          `json:"payment_result"`
	IdAlamatUser      int64                                        `json:"alamat_data_hold"`
	JenisLayananKurir string                                       `json:"jenis_layanan_kurir_keranjang"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Lock Transaksi Wallet
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadLockTransaksiWallet struct {
	DataHold          []response_transaction_pengguna.CheckoutData  `json:"checkout_data_hold"`
	PaymentResult     payment_wallet.WalletResponse                 `json:"payment_result"`
	DataTransaksi     []response_transaction_pengguna.DataTransaksi `json:"data_jarak"`
	IdAlamatUser      int64                                         `json:"alamat_data_hold"`
	JenisLayananKurir string                                        `json:"jenis_layanan_kurir_keranjang"`
}

type PayloadPaidFailedTransaksiWallet struct {
	DataHold      []response_transaction_pengguna.CheckoutData `json:"checkout_data_hold"`
	PaymentResult payment_wallet.WalletResponse                `json:"payment_result"`
	IdAlamatUser  int64                                        `json:"alamat_data_hold"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Lock Transaksi Gerai
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadLockTransaksiGerai struct {
	DataHold          []response_transaction_pengguna.CheckoutData  `json:"checkout_data_hold"`
	PaymentResult     payment_gerai.GeraiResponse                   `json:"payment_result"`
	DataTransaksi     []response_transaction_pengguna.DataTransaksi `json:"data_jarak"`
	IdAlamatUser      int64                                         `json:"alamat_data_hold"`
	JenisLayananKurir string                                        `json:"jenis_layanan_kurir_keranjang"`
}

type PayloadPaidFailedTransaksiGerai struct {
	DataHold          []response_transaction_pengguna.CheckoutData  `json:"checkout_data_hold"`
	PaymentResult     payment_gerai.GeraiResponse                   `json:"payment_result"`
	DataTransaksi     []response_transaction_pengguna.DataTransaksi `json:"data_jarak"`
	IdAlamatUser      int64                                         `json:"alamat_data_hold"`
	JenisLayananKurir string                                        `json:"jenis_layanan_kurir_keranjang"`
}

type PayloadMemberikanUlasan struct {
}
