package kurir

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	kurir_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/alamat_services"
	kurir_informasi_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/informasi_services"
	kurir_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/media_services"
	kurir_pengiriman_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/pengiriman_services"
	kurir_rekening_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/rekening_services"
)

func DeleteKurirHandler(db *environment.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseForm
	switch r.URL.Path {
	case "/kurir/alamat/hapus-alamat":
		var data kurir_alamat_services.PayloadHapusAlamatKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_alamat_services.HapusAlamatKurir(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/kurir/informasi/hapus-informasi-kendaraan":
		var data kurir_informasi_services.PayloadHapusInformasiDataKendaraan
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_informasi_services.HapusInformasiKendaraan(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/kurir/informasi/hapus-informasi-kurir":
		var data kurir_informasi_services.PayloadHapusInformasiDataKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_informasi_services.HapusInformasiKurir(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/kurir/rekening/hapus-rekening":
		var data kurir_rekening_services.PayloadHapusRekeningKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_rekening_services.HapusRekeningKurir(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/kurir/pengiriman/non-aktifkan-bid":
		var data kurir_pengiriman_services.PayloadNonaktifkanBidKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_pengiriman_services.NonaktifkanBidKurir(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/kurir/media/hapus-foto-profil-kurir":
		var data kurir_media_services.PayloadHapusKurirProfilFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusKurirProfilFoto(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/kurir/media/hapus-foto-informasi-kendaraan-kurir-kendaraan":
		var data kurir_media_services.PayloadHapusMediaInformasiKurirKendaraanFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusMediaInformasiKendaraanKurirKendaraanFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/kurir/media/hapus-foto-informasi-kendaraan-kurir-bpkb":
		var data kurir_media_services.PayloadHapusMediaInformasiKendaraanKurirBPKBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusInformasiKendaraanKurirBPKBFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/kurir/media/hapus-foto-informasi-kendaraan-kurir-stnk":
		var data kurir_media_services.PayloadHapusMediaInformasiKendaraanKurirSTNKFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusInformasiKendaraanKurirSTNKFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/kurir/media/hapus-foto-informasi-kurir-ktp":
		var data kurir_media_services.PayloadHapusMediaInformasiKurirKTPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusMediaInformasiKurirKTPFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	case "/kurir/media/hapus-foto-informasi-kurir-sim":
		var data kurir_media_services.PayloadHapusMediaInformasiKurirKTPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.HapusMediaInformasiKurirKTPFoto(r.Context(), data, db, ms, rds_session, mb_cud_publisher)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
