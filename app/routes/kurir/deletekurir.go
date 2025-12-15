package kurir

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	kurir_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/alamat_services"
	kurir_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/media_services"
	kurir_pengiriman_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/pengiriman_services"
	kurir_rekening_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/rekening_services"
)

func DeleteKurirHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client) {
	ctx := r.Context()
	var hasil *response.ResponseForm
	switch r.URL.Path {
	case "/kurir/alamat/hapus-alamat":
		var data kurir_alamat_services.PayloadHapusAlamatKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_alamat_services.HapusAlamatKurir(ctx, data, db)
	case "/kurir/rekening/hapus-rekening":
		var data kurir_rekening_services.PayloadHapusRekeningKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_rekening_services.HapusRekeningKurir(ctx, data, db)
	case "/kurir/pengiriman/non-aktifkan-bid":
		var data kurir_pengiriman_services.PayloadNonaktifkanBidKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_pengiriman_services.NonaktifkanBidKurir(ctx, data, db)
	case "/kurir/media/hapus-foto-profil-kurir":
		var data kurir_media_services.PayloadHapusKurirProfilFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusKurirProfilFoto(ctx, data, db)
	case "/kurir/media/hapus-foto-informasi-kendaraan-kurir-kendaraan":
		var data kurir_media_services.PayloadHapusMediaInformasiKurirKendaraanFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusMediaInformasiKendaraanKurirKendaraanFoto(ctx, data, db, ms)
	case "/kurir/media/hapus-foto-informasi-kendaraan-kurir-bpkb":
		var data kurir_media_services.PayloadHapusMediaInformasiKendaraanKurirBPKBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusInformasiKendaraanKurirBPKBFoto(ctx, data, db, ms)
	case "/kurir/media/hapus-foto-informasi-kendaraan-kurir-stnk":
		var data kurir_media_services.PayloadHapusMediaInformasiKendaraanKurirSTNKFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusInformasiKendaraanKurirSTNKFoto(ctx, data, db, ms)
	case "/kurir/media/hapus-foto-informasi-kurir-ktp":
		var data kurir_media_services.PayloadHapusMediaInformasiKurirKTPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusMediaInformasiKurirKTPFoto(ctx, data, db, ms)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
