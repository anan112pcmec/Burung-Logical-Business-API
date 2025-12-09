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
