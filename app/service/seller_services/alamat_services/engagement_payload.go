package seller_alamat_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Tambah Alamat Gudang
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadTambahAlamatGudang struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	PanggilanAlamat string                         `json:"panggilan_alamat"`
	NomorTelefon    string                         `json:"nomor_telefon"`
	NamaAlamat      string                         `json:"nama_alamat"`
	Provinsi        string                         `json:"provinsi"`
	Kota            string                         `json:"kota"`
	KodePos         string                         `json:"kode_pos"`
	KodeNegara      string                         `json:"kode_negara"`
	Deskripsi       string                         `json:"deskripsi"`
	Longitude       float64                        `json:"longitude"`
	Latitude        float64                        `json:"latitude"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Alamat Gudang
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditAlamatGudang struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdAlamatGudang  int64                          `json:"id_alamat_gudang"`
	PanggilanAlamat string                         `json:"panggilan_alamat"`
	NomorTelefon    string                         `json:"nomor_telefon"`
	NamaAlamat      string                         `json:"nama_alamat"`
	Provinsi        string                         `json:"provinsi"`
	Kota            string                         `json:"kota"`
	KodePos         string                         `json:"kode_pos"`
	KodeNegara      string                         `json:"kode_negara"`
	Deskripsi       string                         `json:"deskripsi"`
	Longitude       float64                        `json:"longitude"`
	Latitude        float64                        `json:"latitude"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Alamat Gudang
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusAlamatGudang struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdGudang        int64                          `json:"id_hapus_alamat_gudang"`
}
