package pengguna_barang_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Uncritical
// Tidak terlalu membutuhkan identitas pengguna karna akan memperlambat dan bukan untuk kepentingan yang absah
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct View Barang
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadViewBarang struct {
	ID int32 `json:"id_barang_induk"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Likes Barang
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadLikesBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IDBarangInduk     int32                              `json:"id_barang_induk_likes"`
}

type PayloadUnlikeBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdBarangDisukai   int64                              `json:"id_barang_disukai"`
	IdBarangInduk     int32                              `json:"id_barang_induk"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Critical
// Dibutuhkan Identitas Pengguna untuk kepentingan yang absah
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Masukan Komentar Barang Induk
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMasukanKomentarBarangInduk struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdBarangInduk     int32                              `json:"id_barang_induk_masukan_komentar"`
	Komentar          string                             `json:"komentar_masukan_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Komentar Barang Induk
// ////////////////////////////////////////////////////////////////////////////////////////////////////////
type PayloadEditKomentarBarangInduk struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKomentar        int64                              `json:"id_komentar_edit_komentar"`
	Komentar          string                             `json:"komentar_edit_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Komentar Barang Induk
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusKomentarBarangInduk struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKomentar        int64                              `json:"id_komentar_hapus_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Masukan Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMasukanChildKomentar struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKomentarBarang  int64                              `json:"id_komentar_masukan_komentar"`
	Komentar          string                             `json:"komentar_masukan_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Mention Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMentionChildKomentar struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKomentar        int64                              `json:"id_komentar_child_komentar"`
	UsernameMentioned string                             `json:"username_mention_komentar"`
	Komentar          string                             `json:"komentar_mention_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditChildKomentar struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKomentar        int64                              `json:"id_child_komentar"`
	Komentar          string                             `json:"komentar_child_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Child Komentar
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusChildKomentar struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKomentar        int64                              `json:"id_child_komentar"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Tambah Keranjang
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadTambahDataKeranjangBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdSeller          int32                              `json:"id_seller"`
	IdBarangInduk     int32                              `json:"id_barang_induk"`
	IdKategori        int64                              `json:"id_kategori_barang"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Edit Keranjang
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEditDataKeranjangBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKeranjang       int64                              `json:"id_keranjang"`
	IdBarangInduk     int32                              `json:"id_barang_induk"`
	IdKategori        int64                              `json:"id_kategori_barang"`
	Jumlah            int64                              `json:"jumlah_di_keranjang"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Hapus Keranjang
// ////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadHapusDataKeranjangBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdKeranjang       int64                              `json:"id_keranjang"`
}

type PayloadBerikanReviewBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdBarangInduk     int64                              `json:"id_barang_induk"`
	Rating            float32                            `json:"rating"`
	Ulasan            string                             `json:"ulasan"`
}

type PayloadLikeReviewBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdReview          int64                              `json:"id_review"`
}

type PayloadUnlikeReviewBarang struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdReview          int64                              `json:"id_review"`
}
