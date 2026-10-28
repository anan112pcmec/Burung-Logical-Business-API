package jenis_seller_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Payload Struct Ajukan Ubah Jenis Seller
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadMasukanDataDistributor struct {
	IdentitasSeller           identity_seller.IdentitySeller `json:"identitas_seller"`
	NamaPerusahaan            string                         `json:"nama_perusahaan"`
	NIB                       string                         `json:"kode_nib" `
	NPWP                      string                         `json:"kode_npwp"`
	DokumenIzinDistributorUrl string                         `json:"dokumen_izin_distributor_url"`
	Alasan                    string                         `json:"alasan"`
}

type PayloadEditDataDistributor struct {
	IdentitasSeller           identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData         int64                          `json:"id_data_distributor"`
	NamaPerusahaan            string                         `json:"nama_perusahaan"`
	NIB                       string                         `json:"kode_nib" `
	NPWP                      string                         `json:"kode_npwp"`
	DokumenIzinDistributorUrl string                         `json:"dokumen_izin_distributor_url"`
	Alasan                    string                         `json:"alasan"`
}

type PayloadHapusDataDistributor struct {
	IdentitasSeller   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData int64                          `json:"id_data_distributor"`
}

type PayloadMasukanDataBrand struct {
	IdentitasSeller       identity_seller.IdentitySeller `json:"identitas_seller"`
	NamaPerusahaan        string                         `json:"nama_perusahaan"`
	NegaraAsal            string                         `json:"negara_asal"`
	LembagaPendaftaran    string                         `json:"lembaga_pendaftaran"`
	NomorPendaftaranMerek string                         `json:"nomor_pendaftaran_merek"`
	SertifikatMerekUrl    string                         `json:"sertifikat_merek_url"`
	DokumenPerwakilanUrl  string                         `json:"dokumen_perwakilan_url"`
	NIB                   string                         `json:"kode_nib"`
	NPWP                  string                         `json:"kode_npwp"`
	Alasan                string                         `json:"alasan"`
}

type PayloadEditDataBrand struct {
	IdentitasSeller       identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDataBrand           int64                          `json:"id_data_brand"`
	NamaPerusahaan        string                         `json:"nama_perusahaan"`
	NegaraAsal            string                         `json:"negara_asal"`
	LembagaPendaftaran    string                         `json:"lembaga_pendaftaran"`
	NomorPendaftaranMerek string                         `json:"nomor_pendaftaran_merek"`
	SertifikatMerekUrl    string                         `json:"sertifikat_merek_url"`
	DokumenPerwakilanUrl  string                         `json:"dokumen_perwakilan_url"`
	NIB                   string                         `json:"kode_nib"`
	NPWP                  string                         `json:"kode_npwp"`
	Alasan                string                         `json:"alasan"`
}

type PayloadHapusDataBrand struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDataBrand     int64                          `json:"id_data_brand"`
}
