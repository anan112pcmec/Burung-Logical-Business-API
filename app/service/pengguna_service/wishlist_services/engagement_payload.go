package pengguna_wishlist_services

import "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"

type PayloadTambahWishlist struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	Nama              string                             `json:"nama"`
	Deskripsi         string                             `json:"deskripsi"`
	Visibility        string                             `json:"visibility"`
}
type PayloadTambahBarangKeWishlist struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdWishlist        int64                              `json:"id_wishlist"`
	IdBarangInduk     int32                              `json:"id_barang_induk"`
}

type PayloadHapusBarangDariWishlist struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdWishlistProduct int64                              `json:"id_wishlist"`
	IdBarangInduk     int32                              `json:"id_barang_induk"`
}

type PayloadHapusWishlist struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdWishlist        int64                              `json:"id_wishlist"`
}
