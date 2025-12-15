package seller

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	seller_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/alamat_services"
	seller_service "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/barang_services"
	seller_credential_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/credential_services"
	seller_diskon_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/diskon_services"
	seller_etalase_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/etalase_services"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/jenis_seller_services"
	seller_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/media_services"
	seller_transaksi_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/transaksi_services"
)

func DeleteSellerHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client) {
	var hasil *response.ResponseForm

	ctx := r.Context()

	switch r.URL.Path {
	case "/seller/hapus_barang":
		var data seller_service.PayloadHapusBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
		}
		hasil = seller_service.HapusBarangInduk(ctx, db, data)
	case "/seller/hapus_kategori_barang":
		var data seller_service.PayloadHapusKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
		}
		hasil = seller_service.HapusKategoriBarang(ctx, db, data)
	case "/seller/credential/hapus-rekening":
		var data seller_credential_services.PayloadHapusNorekSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
		}
		hasil = seller_credential_services.HapusRekeningSeller(ctx, data, db)
	case "/seller/alamat/hapus-alamat-gudang":
		var data seller_alamat_services.PayloadHapusAlamatGudang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_alamat_services.HapusAlamatGudang(ctx, data, db)
	case "/seller/komentar-barang/hapus":
		var data seller_service.PayloadHapusKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.HapusKomentarBarang(ctx, data, db)
	case "/seller/komentar-child/hapus":
		var data seller_service.PayloadHapusChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.HapusChildKomentar(ctx, data, db)
	case "/seller/diskon/hapus-diskon":
		var data seller_diskon_services.PayloadHapusDiskonProduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.HapusDiskonProduk(ctx, data, db)
	case "/seller/diskon/hapus-diskon-barang":
		var data seller_diskon_services.PayloadHapusDiskonPadaBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.HapusDiskonPadaBarang(ctx, data, db)
	case "/seller/etalase/hapus-etalase":
		var data seller_etalase_services.PayloadHapusEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.HapusEtalaseSeller(ctx, data, db)
	case "/seller/etalase/hapus-barang-dari-etalase":
		var data seller_etalase_services.PayloadHapusBarangDiEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.HapusBarangDariEtalase(ctx, data, db)
	case "/seller/jenis/hapus-data-distributor":
		var data jenis_seller_services.PayloadHapusDataDistributor
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.HapusDataDistributor(ctx, data, db)
	case "/seller/jenis/hapus-data-brand":
		var data jenis_seller_services.PayloadHapusDataBrand
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.HapusDataBrand(ctx, data, db)
	case "/seller/transaction/unapprove-order":
		var data seller_transaksi_services.PayloadUnApproveOrderTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_transaksi_services.UnApproveOrderTransaksi(ctx, data, db)
	case "/seller/media/hapus-foto-profile":
		var data seller_media_services.PayloadHapusFotoProfilSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoProfilSeller(ctx, data, db, ms)
	case "/seller/media/hapus-foto-banner":
		var data seller_media_services.PayloadHapusFotoBannerSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoBannerSeller(ctx, data, db, ms)
	case "/seller/media/hapus-foto-toko-fisik":
		var data seller_media_services.PayloadHapusFotoTokoFisikSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoTokoFisikSeller(ctx, data, db, ms)
	case "/seller/media/hapus-foto-etalase":
		var data seller_media_services.PayloadHapusFotoEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoEtalaseSeller(ctx, data, db, ms)
	case "/seller/media/hapus-foto-barang-induk":
		var data seller_media_services.PayloadHapusBarangIndukFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBarangIndukFoto(ctx, data, db, ms)
	case "/seller/media/hapus-video-barang-induk":
		var data seller_media_services.PayloadHapusVideoBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusBarangIndukVideo(ctx, data, db, ms)
	case "/seller/media/hapus-foto-kategori-barang":
		var data seller_media_services.PayloadHapusKategoriBarangFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusKategoriBarangFoto(ctx, data, db, ms)
	case "/seller/media/hapus-dokumen-distributor-data":
		var data seller_media_services.PayloadHapusMediaDistributorDataDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaDistributorDataDokumen(ctx, data, db, ms)
	case "/seller/media/hapus-foto-distributor-data-npwp":
		var data seller_media_services.PayloadHapusMediaDistributorDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaDistributorDataNPWPFoto(ctx, data, db, ms)
	case "/seller/media/hapus-foto-distributor-data-nib":
		var data seller_media_services.PayloadHapusDistributorDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusDistributorDataNIBFoto(ctx, data, db, ms)
	case "/seller/media/hapus-dokumen-distributor-data-surat-kerjasama":
		var data seller_media_services.PayloadHapusDistributorDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusDistributorDataSuratKerjasamaDataDokumen(ctx, data, db, ms)
	case "/seller/media/hapus-dokumen-brand-data-perwakilan":
		var data seller_media_services.PayloadHapusBrandDataPerwakilanDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataPerwakilanDokumen(ctx, data, db, ms)
	case "/seller/media/hapus-foto-brand-data-sertifikat":
		var data seller_media_services.PayloadHapusBrandDataSertifikatFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataSertifikatFoto(ctx, data, db, ms)
	case "/seller/media/hapus-foto-brand-data-nib":
		var data seller_media_services.PayloadHapusMediaBrandDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataNIBFoto(ctx, data, db, ms)
	case "/seller/media/hapus-foto-brand-data-npwp":
		var data seller_media_services.PayloadHapusMediaBrandDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandNPWPFoto(ctx, data, db, ms)
	case "/seller/media/hapus-foto-brand-data-logo":
		var data seller_media_services.PayloadHapusMediaBrandDataLogoFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataLogo(ctx, data, db, ms)
	case "/seller/media/hapus/-dokumen-brand-data-surat-kerjasama":
		var data seller_media_services.PayloadHapusBrandDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusBrandDataSuratKerjasamaDokumen(ctx, data, db, ms)
	default:
		hasil = &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: "Seller Services",
			Payload:  "Gagal Coba Lagi Nanti",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
