package kurir_informasi_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/identity_kurir"
)

type PayloadInformasiDataKendaraan struct {
	DataIdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	JenisKendaraan     string                        `json:"jenis_kendaraan"`
	NamaKendaraan      string                        `json:"nama_kendaraan"`
	RodaKendaraan      string                        `json:"roda_kendaraan"`
	InformasiStnk      bool                          `json:"informasi_stnk"`
	InformasiBpkb      bool                          `json:"informasi_bpkb"`
	NomorRangka        string                        `json:"nomor_rangka"`
	NomorMesin         string                        `json:"nomor_mesin"`
}

type PayloadEditInformasiDataKendaraan struct {
	DataIdentitasKurir   identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdInformasiKendaraan int64                         `json:"id_informasi_kendaraan"`
	JenisKendaraan       string                        `json:"jenis_kendaraan"`
	NamaKendaraan        string                        `json:"nama_kendaraan"`
	RodaKendaraan        string                        `json:"roda_kendaraan"`
	InformasiStnk        bool                          `json:"informasi_stnk"`
	InformasiBpkb        bool                          `json:"informasi_bpkb"`
	NomorRangka          string                        `json:"nomor_rangka"`
	NomorMesin           string                        `json:"nomor_mesin"`
}

type PayloadHapusInformasiDataKendaraan struct {
	IdentitasKurir       identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdInformasiKendaraan int64                         `json:"id_informasi_kendaraan"`
}

type PayloadInformasiDataKurir struct {
	DataIdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	TanggalLahir       string                        `json:"tanggal_lahir"`
	Alasan             string                        `json:"alasan"`
	InformasiKtp       bool                          `json:"informasi_ktp"`
	InformasiSim       bool                          `json:"informasi_sim"`
}

type PayloadEditInformasiDataKurir struct {
	DataIdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdInformasiKurir   int64                         `json:"id_informasi_kurir"`
	TanggalLahir       string                        `json:"tanggal_lahir"`
	Alasan             string                        `json:"alasan"`
	InformasiKtp       bool                          `json:"informasi_ktp"`
	InformasiSim       bool                          `json:"informasi_sim"`
}

type PayloadHapusInformasiDataKurir struct {
	IdentitasKurir   identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdInformasiKurir int64                         `json:"id_informasi_kurir"`
}
