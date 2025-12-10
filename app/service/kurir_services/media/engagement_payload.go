package kurir_media_services

import "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/identity_kurir"

type PayloadUbahKurirProfilFoto struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	Ekstensi       string                        `json:"ekstensi"`
}

type PayloadHapusKurirProfilFoto struct {
	IdentitasKurir         identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdMediaKurirProfilFoto int64                         `json:"id_media_kurir_profil_foto"`
	KeyFoto                string                        `json:"key_foto"`
}
