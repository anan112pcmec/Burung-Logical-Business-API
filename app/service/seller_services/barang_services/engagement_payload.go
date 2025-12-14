package seller_barang_service

import (
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Masukan Barang Induk
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMasukanBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	BarangInduk     models.BarangInduk             `json:"data_barang_induk"`
	KategoriBarang  []models.KategoriBarang        `json:"data_kategori_barang_induk"`
	IdAlamatGudang  int64                          `json:"id_alamat_gudang"`
	IdRekening      int64                          `json:"id_rekening"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Barang Induk
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int64                          `json:"id_barang_induk"`
	NamaBarang      string                         `json:"nama_barang"`
	JenisBarang     string                         `json:"jenis_barang"`
	Deskripsi       string                         `json:"deskripsi"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Barang Induk
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int64                          `json:"id_barang_induk"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Tambah Kategori
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadTambahKategori struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int32                          `json:"id_barang_induk_tambah_kategori"`
	KategoriBarang  []models.KategoriBarang        `json:"tambah_kategori_barang"`
	IdAlamatGudang  int64                          `json:"id_alamat_gudang"`
	IdRekening      int64                          `json:"id_rekening"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Kategori
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditKategori struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int32                          `json:"id_barang_induk_edit_kategori"`
	IdKategoriBarang int64                          `json:"id_kategori_barang"`
	Nama             string                         `json:"nama"`
	Deskripsi        string                         `json:"deskripsi"`
	Warna            string                         `json:"warna"`
	DimensiPanjang   int16                          `json:"dimensi_panjang"`
	DimensiLebar     int16                          `json:"dimensi_lebar"`
	Sku              string                         `json:"sku"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Kategori
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusKategori struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int32                          `json:"id_barang_induk"`
	IdKategoriBarang int64                          `json:"id_kategori_barang"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Stok Kategori
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Edit Stok Barang
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditStokKategoriBarang struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int32                          `json:"id_barang_induk"`
	IdKategoriBarang int64                          `json:"id_kategori"`
	UpdateStok       int64                          `json:"update_stok"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Down Barang Induk
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadDownBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int32                          `json:"id_barang_induk_down"`
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Down Kategori Barang
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadDownKategoriBarang struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int32                          `json:"id_barang_induk"`
	IdKategoriBarang int64                          `json:"id_kategori_barang"`
}

// ////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Rekening Barang Induk
// ////////////////////////////////////////////////////////////////////////////////

type PayloadEditRekeningBarangInduk struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int32                          `json:"id_barang_induk"`
	IdRekeningSeller int64                          `json:"id_rekening_seller"`
}

// ////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Alamat Barang Induk
// ////////////////////////////////////////////////////////////////////////////////

type PayloadEditAlamatBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int32                          `json:"id_barang_induk"`
	IdAlamatGudang  int64                          `json:"id_alamat_gudang"`
}

// ////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Alamat Barang Kategori
// ////////////////////////////////////////////////////////////////////////////////

type PayloadEditAlamatBarangKategori struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int32                          `json:"id_barang_induk"`
	IdKategoriBarang int64                          `json:"id_kategori_barang"`
	IdAlamatGudang   int64                          `json:"id_alamat_gudang"`
}

type PayloadMasukanKomentarBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int32                          `json:"id_barang_induk_masukan_komentar"`
	Komentar        string                         `json:"komentar_masukan_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Komentar Barang Induk
// ////////////////////////////////////////////////////////////////////////////////////////////////////////
type PayloadEditKomentarBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdKomentar      int64                          `json:"id_komentar_edit_komentar"`
	Komentar        string                         `json:"komentar_edit_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Komentar Barang Induk
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusKomentarBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdKomentar      int64                          `json:"id_komentar_hapus_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Masukan Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMasukanChildKomentar struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdKomentarBarang int64                          `json:"id_komentar_masukan_komentar"`
	IdBarangInduk    int32                          `json:"id_barang_induk_child_komentar"`
	Komentar         string                         `json:"komentar_masukan_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Mention Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMentionChildKomentar struct {
	IdentitasSeller   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk     int32                          `json:"id_barang_induk_child_komentar"`
	IdKomentar        int64                          `json:"id_komentar_child_komentar"`
	UsernameMentioned string                         `json:"username_mention_komentar"`
	Komentar          string                         `json:"komentar_mention_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditChildKomentar struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdKomentar      int64                          `json:"id_child_komentar"`
	Komentar        string                         `json:"komentar_child_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusChildKomentar struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdKomentar      int64                          `json:"id_child_komentar"`
}
