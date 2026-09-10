package transaksi_enums

const (
	Dibayar    string = "Dibayar"
	Diproses   string = "Diproses"
	Waiting    string = "Waiting"
	Dikirim    string = "Dikirim"
	Selesai    string = "Selesai"
	Dibatalkan string = "Dibatalkan"
)

func NamaStatusTransaksiEnums() string {
	return "status_transaksi"
}

func StatusTransaksiEnums() []string {
	return []string{Dibayar, Diproses, Waiting, Dikirim, Selesai, Dibatalkan}
}

// Untuk failed transaksi

const (
	Pending string = "Pending"
	Batal   string = "Batal"
	Lanjut  string = "Lanjut"
)

func NamaStatusPaidFailedEnums() string {
	return "status_paid_failed"
}

func StatusPaidFailedEnums() []string {
	return []string{Pending, Batal, Lanjut}
}
