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
	seller_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/media_services"
)

func PutSellerHandler(db *environment.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseMediaUpload

	switch r.URL.Path {
	case "/seller/media/ubah-foto-profile":
		var data seller_media_services.PayloadUbahFotoProfilSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoProfilSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/ubah-foto-banner":
		var data seller_media_services.PayloadUbahFotoBannerSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoBannerSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-toko-fisik":
		var data seller_media_services.PayloadTambahkanFotoTokoFisikSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahkanFotoTokoFisikSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/ubah-foto-etalase":
		var data seller_media_services.PayloadUbahFotoEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoEtalaseSeller(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-barang-induk":
		var data seller_media_services.PayloadTambahBarangIndukFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahkanMediaBarangIndukFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/ubah-video-barang-induk":
		var data seller_media_services.PayloadUbahVideoBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahBarangIndukVideo(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/ubah-foto-kategori-barang":
		var data seller_media_services.PayloadUbahKategoriBarangFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahKategoriBarangFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-dokumen-distributor-data":
		var data seller_media_services.PayloadMediaDistributorDataDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahDistributorDataDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-distributor-data-npwp":
		var data seller_media_services.PayloadTambahMediaDistributorDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaDistributorDataNPWPFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-distributor-data-nib":
		var data seller_media_services.PayloadTambahDistributorDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahDistributorDataNIBFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-dokumen-distributor-data-surat-kerjasama":
		var data seller_media_services.PayloadTambahDistributorDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahDistributorDataSuratKerjasamaDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-dokumen-brand-data-perwakilan":
		var data seller_media_services.PayloadTambahBrandDataPerwakilanDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahBrandDataPerwakilanDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-brand-data-sertifikat":
		var data seller_media_services.PayloadTambahBrandDataSertifikatFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandDataSertifikatFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-brand-data-nib":
		var data seller_media_services.PayloadTambahMediaBrandDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandDataNIBFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-brand-data-npwp":
		var data seller_media_services.PayloadTambahMediaBrandDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandNPWPFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-brand-data-logo":
		var data seller_media_services.PayloadTambahMediaBrandDataLogoFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandDataLogoFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-dokumen-brand-data-surat-kerjasama":
		var data seller_media_services.PayloadTambahBrandDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahBrandDataSuratKerjasamaDokumen(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-foto-approve-transaksi":
		var data seller_media_services.PayloadTambahMediaTransaksiApprovedFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaTransaksiApprovedFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/seller/media/tambah-video-approve-transaksi":
		var data seller_media_services.PayloadTambahMediaTransaksiApprovedVideo
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahTransaksiApprovedVideo(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(hasil.Status))
	json.NewEncoder(w).Encode(hasil)
}
