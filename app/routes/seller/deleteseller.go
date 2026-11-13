package seller

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
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

func DeleteSellerHandler(db *environment.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseForm

	switch r.URL.Path {
	case "/seller/hapus_barang":
		var data seller_service.PayloadHapusBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
		}
		hasil = seller_service.HapusBarangInduk(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/hapus_kategori_barang":
		var data seller_service.PayloadHapusKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
		}
		hasil = seller_service.HapusKategoriBarang(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/credential/hapus-rekening":
		var data seller_credential_services.PayloadHapusNorekSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
		}
		hasil = seller_credential_services.HapusRekeningSeller(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/alamat/hapus-alamat-gudang":
		var data seller_alamat_services.PayloadHapusAlamatGudang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_alamat_services.HapusAlamatGudang(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/komentar-barang/hapus":
		var data seller_service.PayloadHapusKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.HapusKomentarBarang(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/komentar-child/hapus":
		var data seller_service.PayloadHapusChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.HapusChildKomentar(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/diskon/hapus-diskon":
		var data seller_diskon_services.PayloadHapusDiskonProduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.HapusDiskonProduk(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/diskon/hapus-diskon-barang":
		var data seller_diskon_services.PayloadHapusDiskonPadaBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.HapusDiskonPadaBarang(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/etalase/hapus-etalase":
		var data seller_etalase_services.PayloadHapusEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.HapusEtalaseSeller(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/etalase/hapus-barang-dari-etalase":
		var data seller_etalase_services.PayloadHapusBarangDiEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.HapusBarangDariEtalase(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/jenis/hapus-data-distributor":
		var data jenis_seller_services.PayloadHapusDataDistributor
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.HapusDataDistributor(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/jenis/hapus-data-brand":
		var data jenis_seller_services.PayloadHapusDataBrand
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.HapusDataBrand(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/transaction/unapprove-order":
		var data seller_transaksi_services.PayloadUnApproveOrderTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_transaksi_services.UnApproveOrderTransaksi(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-profile":
		var data seller_media_services.PayloadHapusFotoProfilSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoProfilSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-banner":
		var data seller_media_services.PayloadHapusFotoBannerSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoBannerSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-toko-fisik":
		var data seller_media_services.PayloadHapusFotoTokoFisikSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoTokoFisikSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-etalase":
		var data seller_media_services.PayloadHapusFotoEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusFotoEtalaseSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-barang-induk":
		var data seller_media_services.PayloadHapusBarangIndukFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBarangIndukFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-video-barang-induk":
		var data seller_media_services.PayloadHapusVideoBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusBarangIndukVideo(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-kategori-barang":
		var data seller_media_services.PayloadHapusKategoriBarangFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusKategoriBarangFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-dokumen-distributor-data":
		var data seller_media_services.PayloadHapusMediaDistributorDataDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaDistributorDataDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-distributor-data-npwp":
		var data seller_media_services.PayloadHapusMediaDistributorDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaDistributorDataNPWPFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-distributor-data-nib":
		var data seller_media_services.PayloadHapusDistributorDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusDistributorDataNIBFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-dokumen-distributor-data-surat-kerjasama":
		var data seller_media_services.PayloadHapusDistributorDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusDistributorDataSuratKerjasamaDataDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-dokumen-brand-data-perwakilan":
		var data seller_media_services.PayloadHapusBrandDataPerwakilanDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataPerwakilanDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-brand-data-sertifikat":
		var data seller_media_services.PayloadHapusBrandDataSertifikatFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataSertifikatFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-brand-data-nib":
		var data seller_media_services.PayloadHapusMediaBrandDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataNIBFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-brand-data-npwp":
		var data seller_media_services.PayloadHapusMediaBrandDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandNPWPFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus-foto-brand-data-logo":
		var data seller_media_services.PayloadHapusMediaBrandDataLogoFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusMediaBrandDataLogo(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/hapus/-dokumen-brand-data-surat-kerjasama":
		var data seller_media_services.PayloadHapusBrandDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.HapusBrandDataSuratKerjasamaDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
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
