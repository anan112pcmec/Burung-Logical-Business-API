package seller_profiling_services

import "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"

type PayloadUpdateProfilePersonalSeller struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	Username        string                         `json:"update_username_seller"`
	Nama            string                         `json:"update_nama_seller"`
	Email           string                         `json:"update_email_seller"`
}

type PayloadUpdateInfoGeneralSeller struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	JamOperasional  string                         `json:"update_jam_operasional_seller"`
	Punchline       string                         `json:"update_punchline_seller"`
	Deskripsi       string                         `json:"update_deskripsi_seller"`
	Dedication      string                         `json:"update_dedication_seller"`
}
