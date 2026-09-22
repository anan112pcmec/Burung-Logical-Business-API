package seller_diskon_services

import (
	"time"

	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"

)

type PayloadTambahDiskonProduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	Nama            string                         `json:"nama"`
	Deskripsi       string                         `json:"deksripsi"`
	DiskonPersen    float64                        `json:"diskon_persen"`
	BerlakuMulai    time.Time                      `json:"berlaku_mulai"`
	BerlakuSampai   time.Time                      `json:"berlaku_sampai"`
}


type PayloadEditDiskonProduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDiskonProduk  int64                          `json:"id_diskon_produk"`
	Nama            string                         `json:"nama"`
	Deskripsi       string                         `json:"deksripsi"`
	DiskonPersen    float64                        `json:"diskon_persen"`
	BerlakuMulai    time.Time                      `json:"berlaku_mulai"`
	BerlakuSampai   time.Time                      `json:"berlaku_sampai"`
}

type PayloadHapusDiskonProduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDiskonProduk  int64                          `json:"id_diskon_produk"`
}

type PayloadTetapkanDiskonPadaBarang struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDiskonProduk   int64                          `json:"id_diskon_produk"`
	IdBarangInduk    int32                          `json:"id_barang_induk"`
	IdKategoriBarang int64                          `json:"id_kategori_barang"`
}

type PayloadHapusDiskonPadaBarang struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangDiDiskon int64                          `json:"id_barang_di_diskon"`
}
