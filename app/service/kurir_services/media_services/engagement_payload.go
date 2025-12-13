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

type PayloadTambahMediaInformasiKurirKendaraanFoto struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	Ekstensi       string                        `json:"ekstensi"`
}

type PayloadHapusMediaInformasiKurirKendaraanFoto struct {
	IdentitasKurir                     identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdMediaKendaraanKurirKendaraanFoto int64                         `json:"id_media_informasi_kendaraan_kurir_kendaraan_foto"`
	KeyFoto                            string                        `json:"key_foto"`
}

type PayloadTambahMediaInformasiKendaraanKurirBPKBFoto struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	Ekstensi       string                        `json:"ekstensi"`
}

type PayloadHapusMediaInformasiKendaraanKurirBPKBFoto struct {
	IdentitasKurir                         identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdMediaInformasiKendaraanKurirBPKBFoto int64                         `json:"id_media_informasi_kendaraan_kurir_bpkb_foto"`
	KeyFoto                                string                        `json:"key_foto"`
}

type PayloadTambahMediaInformasiKendaraanKurirSTNKFoto struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	Ekstensi       string                        `json:"ekstensi"`
}

type PayloadHapusMediaInformasiKendaraanKurirSTNKFoto struct {
	IdentitasKurir                         identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdMediaInformasiKendaraanKurirSTNKFoto int64                         `json:"id_media_informasi_kendaraan_kurir_stnk_foto"`
	KeyFoto                                string                        `json:"key_foto"`
}

type PayloadTambahMediaInformasiKurirKTPFoto struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	Ekstensi       string                        `json:"ekstensi"`
}

type PayloadHapusMediaInformasiKurirKTPFoto struct {
	IdentitasKurir               identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdMediaInformasiKurirKTPFoto string                        `json:"id_media_informasi_kurir_ktp_foto"`
	KeyFoto                      string                        `json:"key_foto"`
}
