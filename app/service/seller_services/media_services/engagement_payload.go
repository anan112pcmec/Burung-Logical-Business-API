package seller_media_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"
)

type PayloadUbahFotoProfilSeller struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusFotoProfilSeller struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdMediaSeller   int64                          `json:"id_media_seller_profil_foto"`
	KeyFoto         string                         `json:"key_foto"`
}

type PayloadUbahFotoBannerSeller struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusFotoBannerSeller struct {
	IdentitasSeller         identity_seller.IdentitySeller `json:"identitas_seller"`
	IdMediaBannerSellerFoto int64                          `json:"id_media_seller_banner_foto"`
	KeyFoto                 string                         `json:"key_foto"`
}

type PayloadTambahkanFotoTokoFisikSeller struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	Ekstensi        []string                       `json:"ekstensi"`
}

type PayloadHapusFotoTokoFisikSeller struct {
	IdentitasSeller        identity_seller.IdentitySeller `json:"identitas_seller"`
	DataMediaFotoTokoFisik []struct {
		IdMedia  int64  `json:"id_media_seller_fisik_toko"`
		KeyMedia string `json:"key"`
	} `json:"data_media_dan_key"`
}

type PayloadUbahFotoEtalase struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdEtalase       int64                          `json:"id_etalase"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusFotoEtalase struct {
	IdentitasSeller    identity_seller.IdentitySeller `json:"identitas_seller"`
	IdMediaEtalaseFoto int64                          `json:"id_media_etalase_foto"`
	KeyFoto            string                         `json:"key_foto"`
}

type PayloadTambahBarangIndukFoto struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk   int32                          `json:"id_barang_induk"`
	Ekstensi        []string                       `json:"ekstensi"`
}

type PayloadHapusBarangIndukFoto struct {
	IdentitasSeller          identity_seller.IdentitySeller `json:"identitas_seller"`
	DataMediaBarangIndukFoto []struct {
		IdMedia  int64  `json:"id_media_barang_induk_foto"`
		KeyMedia string `json:"key"`
	} `json:"data_media_dan_key"`
}

type PayloadUbahVideoBarangInduk struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	Ekstensi        string                         `json:"ekstensi"`
}
