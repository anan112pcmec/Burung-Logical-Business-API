package pengguna_media_services

import "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"

type PayloadUbahFotoProfilPengguna struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	Ekstensi          string                             `json:"ekstensi"`
}
