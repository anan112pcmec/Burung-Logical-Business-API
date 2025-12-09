package seller_media_services

import "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"

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
