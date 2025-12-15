package seller

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	seller_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/media_services"
)

func PutSellerHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client) {
	var hasil *response.ResponseMediaUpload
	ctx := r.Context()

	switch r.URL.Path {
	case "/seller/media/ubah-foto-profile":
		var data seller_media_services.PayloadUbahFotoProfilSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoProfilSeller(ctx, data, db, ms)
	case "/seller/media/ubah-foto-banner":
		var data seller_media_services.PayloadUbahFotoBannerSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoBannerSeller(ctx, data, db, ms)
	case "/seller/media/tambah-foto-toko-fisik":
		var data seller_media_services.PayloadTambahkanFotoTokoFisikSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahkanFotoTokoFisikSeller(ctx, data, db, ms)
	case "/seller/media/ubah-foto-etalase":
		var data seller_media_services.PayloadUbahFotoEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoEtalaseSeller(ctx, data, db, ms)
	case "/seller/media/tambah-foto-barang-induk":
		var data seller_media_services.PayloadTambahBarangIndukFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahkanMediaBarangIndukFoto(ctx, data, db, ms)
	case "/seller/media/ubah-video-barang-induk":
		var data seller_media_services.PayloadUbahVideoBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahBarangIndukVideo(ctx, data, db, ms)
	case "/seller/media/ubah-foto-kategori-barang":
		var data seller_media_services.PayloadUbahKategoriBarangFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahKategoriBarangFoto(ctx, data, db, ms)
	case "/seller/media/tambah-dokumen-distributor-data":
		var data seller_media_services.PayloadMediaDistributorDataDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahDistributorDataDokumen(ctx, data, db, ms)
	case "/seller/media/tambah-foto-distributor-data-npwp":
		var data seller_media_services.PayloadTambahMediaDistributorDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaDistributorDataNPWPFoto(ctx, data, db, ms)
	case "/seller/media/tambah-foto-distributor-data-nib":
		var data seller_media_services.PayloadTambahDistributorDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahDistributorDataNIBFoto(ctx, data, db, ms)
	case "/seller/media/tambah-dokumen-distributor-data-surat-kerjasama":
		var data seller_media_services.PayloadTambahDistributorDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahDistributorDataSuratKerjasamaDokumen(ctx, data, db, ms)
	case "/seller/media/tambah-dokumen-brand-data-perwakilan":
		var data seller_media_services.PayloadTambahBrandDataPerwakilanDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahBrandDataPerwakilanDokumen(ctx, data, db, ms)
	case "/seller/media/tambah-foto-brand-data-sertifikat":
		var data seller_media_services.PayloadTambahBrandDataSertifikatFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandDataSertifikatFoto(ctx, data, db, ms)
	case "/seller/media/tambah-foto-brand-data-nib":
		var data seller_media_services.PayloadTambahMediaBrandDataNIBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandDataNIBFoto(ctx, data, db, ms)
	case "/seller/media/tambah-foto-brand-data-npwp":
		var data seller_media_services.PayloadTambahMediaBrandDataNPWPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandNPWPFoto(ctx, data, db, ms)
	case "/seller/media/tambah-foto-brand-data-logo":
		var data seller_media_services.PayloadTambahMediaBrandDataLogoFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaBrandDataLogoFoto(ctx, data, db, ms)
	case "/seller/media/tambah-dokumen-brand-data-surat-kerjasama":
		var data seller_media_services.PayloadTambahBrandDataSuratKerjasamaDokumen
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahBrandDataSuratKerjasamaDokumen(ctx, data, db, ms)
	case "/seller/media/tambah-foto-approve-transaksi":
		var data seller_media_services.PayloadTambahMediaTransaksiApprovedFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahMediaTransaksiApprovedFoto(ctx, data, db, ms)
	case "/seller/media/tambah-video-approve-transaksi":
		var data seller_media_services.PayloadTambahMediaTransaksiApprovedVideo
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.TambahTransaksiApprovedVideo(ctx, data, db, ms)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(hasil.Status))
	json.NewEncoder(w).Encode(hasil)
}
