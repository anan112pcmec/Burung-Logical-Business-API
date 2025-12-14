package pengguna_media_services

import "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"

type PayloadUbahFotoProfilPengguna struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	Ekstensi          string                             `json:"ekstensi"`
}

type PayloadHapusFotoProfilPengguna struct {
	IdentitasPengguna   identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdMediaDataPengguna int64                              `json:"id_media_foto_profil_pengguna"`
	KeyFoto             string                             `json:"key_foto"`
}

type PayloadTambahMediaReviewFoto struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdReviewData      int64                              `json:"id_review_data"`
	Ekstensi          []string                           `json:"ekstensi"`
}

type PayloadHapusMediaReviewFoto struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdReviewData      int64                              `json:"id_review_data"`
}

type PayloadTambahMediaReviewVideo struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdReviewData      int64                              `json:"id_review_data"`
	Ekstensi          string                             `json:"ekstensi"`
}

type PayloadHapusMediaReviewVideo struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	KeyVideo          string                             `json:"key_video"`
}
