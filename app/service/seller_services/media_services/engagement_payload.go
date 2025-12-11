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
	IdBarangInduk   int64                          `json:"id_barang_induk"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusVideoBarangInduk struct {
	IdentitasSeller         identity_seller.IdentitySeller `json:"identitas_seller"`
	IdMediaBarangIndukVideo int64                          `json:"id_media_barang_induk_video"`
	KeyFoto                 string                         `json:"key_foto"`
}

type PayloadUbahKategoriBarangFoto struct {
	IdentitasSeller  identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBarangInduk    int64                          `json:"id_barang_induk"`
	IdKategoriBarang int64                          `json:"id_kategori_barang"`
	Ekstensi         string                         `json:"ekstensi"`
}

type PayloadHapusKategoriBarangFoto struct {
	IdentitasSeller           identity_seller.IdentitySeller `json:"identitas_seller"`
	IdMediaKategoriBarangFoto int64                          `json:"id_media_kategori_barang_foto"`
	KeyFoto                   string                         `json:"key_foto"`
}

type PayloadMediaDistributorDataDokumen struct {
	IdentitasSeller   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData int64                          `json:"id_distributor_data"`
	Ekstensi          string                         `json:"ekstensi"`
}

type PayloadHapusMediaDistributorDataDokumen struct {
	IdentitasSeller               identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData             int64                          `json:"id_distributor_data"`
	IdMediaDistributorDataDokumen int64                          `json:"id_media_distributor_data_dokumen"`
	KeyFoto                       string                         `json:"key_foto"`
}

type PayloadTambahMediaDistributorDataNPWPFoto struct {
	IdentitasSeller   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData int64                          `json:"id_distributor_data"`
	Ekstensi          string                         `json:"ekstensi"`
}

type PayloadHapusMediaDistributorDataNPWPFoto struct {
	IdentitasSeller                identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData              int64                          `json:"id_distributor_data"`
	IdMediaDistributorDataNPWPFoto int64                          `json:"id_media_distributor_data_npwp_foto"`
	KeyFoto                        string                         `json:"key_foto"`
}

type PayloadTambahDistributorDataNIBFoto struct {
	IdentitasSeller   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData int64                          `json:"id_distributor_data"`
	Ekstensi          string                         `json:"ekstensi"`
}

type PayloadHapusDistributorDataNIBFoto struct {
	IdentitasSeller               identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData             int64                          `json:"id_distributor_data"`
	IdMediaDistributorDataNIBFoto int64                          `json:"id_media_distributor_data_nib_foto"`
	KeyFoto                       string                         `json:"key_foto"`
}

type PayloadTambahDistributorDataSuratKerjasamaDokumen struct {
	IdentitasSeller   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData int64                          `json:"id_distributor_data"`
	Ekstensi          string                         `json:"ekstensi"`
}

type PayloadHapusDistributorDataSuratKerjasamaDokumen struct {
	IdentitasSeller                             identity_seller.IdentitySeller `json:"identitas_seller"`
	IdDistributorData                           int64                          `json:"id_distributor_data"`
	IdMediaDistributorDataSuratkerjasamaDokumen int64                          `json:"id_media_distributor_data_surate_kerjasama_dokumen"`
	KeyFoto                                     string                         `json:"key_foto"`
}

type PayloadTambahBrandDataPerwakilanDokumen struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData     int64                          `json:"id_brand_data"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusBrandDataPerwakilanDokumen struct {
	IdentitasSeller                   identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData                       int64                          `json:"id_brand_data"`
	IdMediaBrandDataPerwakilanDokumen int64                          `json:"id_media_brand_data_perwakilan_dokumen"`
	KeyFoto                           string                         `json:"key_foto"`
}

type PayloadTambahBrandDataSertifikatFoto struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData     int64                          `json:"id_brand_data"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusBrandDataSertifikatFoto struct {
	IdentitasSeller                identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData                    int64                          `json:"id_brand_data"`
	IdMediaBrandDataSertifikatFoto int64                          `json:"id_media_brand_data_sertifikat_foto"`
	KeyFoto                        string                         `json:"key_foto"`
}

type PayloadTambahMediaBrandDataNIBFoto struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData     int64                          `json:"id_brand_data"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusMediaBrandDataNIBFoto struct {
	IdentitasSeller         identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData             int64                          `json:"id_brand_data"`
	IdMediaBrandDataNIBFoto int64                          `json:"id_media_brand_data_nib_foto"`
	KeyFoto                 string                         `json:"key_foto"`
}

type PayloadTambahMediaBrandDataNPWPFoto struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData     int64                          `json:"id_brand_data"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusMediaBrandDataNPWPFoto struct {
	IdentitasSeller          identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData              int64                          `json:"id_brand_data"`
	IdMediaBrandDataNPWPFoto int64                          `json:"id_media_brand_data_npwp_foto"`
	KeyFoto                  string                         `json:"key_foto"`
}

type PayloadTambahMediaBrandDataLogoFoto struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData     int64                          `json:"id_brand_data"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusMediaBrandDataLogoFoto struct {
	IdentitasSeller               identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData                   int64                          `json:"id_brand_data"`
	IdMediaBrandDataLogoBrandFoto int64                          `json:"id_media_brand_data_logo_foto"`
	KeyFoto                       string                         `json:"key_foto"`
}

type PayloadTambahBrandDataSuratKerjasamaDokumen struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData     int64                          `json:"id_brand_data"`
	Ekstensi        string                         `json:"ekstensi"`
}

type PayloadHapusBrandDataSuratKerjasamaDokumen struct {
	IdentitasSeller                       identity_seller.IdentitySeller `json:"identitas_seller"`
	IdBrandData                           int64                          `json:"id_brand_data"`
	IdMediaBrandDataSuratKerjasamaDokumen int64                          `json:"id_media_brand_data_surat_kerjasama_dokumen"`
}
