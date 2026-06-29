package pengguna_transaction_services

import (
	"fmt"
	"os"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

)

// ////////////////////////////////////////////////////////////////////////////////////
// Fungsi Prosedur Validating Transaksi
// Befungsi Untuk membantu snap transaksi untuk membuat sebuah transaksi
// ////////////////////////////////////////////////////////////////////////////////////

func ValidateTransaksi(snapReq *snap.Request) (*snap.Response, bool) {
	fmt.Println("[TRACE] Start ValidateTransaksi")

	var s snap.Client
	s.New(os.Getenv("MIDTRANS_SERVER_KEY"), midtrans.Sandbox)

	fmt.Println("[TRACE] Membuat transaksi dengan Snap SDK")
	snapResp, err := s.CreateTransaction(snapReq)
	if err != nil {
		fmt.Printf("[TRACE] Gagal membuat transaksi: %v\n", err)
		return nil, false
	}

	fmt.Println("[TRACE] Selesai ValidateTransaksi, return response")
	return snapResp, true
}
