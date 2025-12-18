package kurir

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	kurir_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/media_services"
)

func PutKurirHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseMediaUpload
	ctx := r.Context()

	switch r.URL.Path {
	case "/kurir/media/ubah-foto-profil-kurir":
		var data kurir_media_services.PayloadUbahKurirProfilFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.UbahKurirProfilFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-informasi-kendaraan-kurir-kendaraan":
		var data kurir_media_services.PayloadTambahMediaInformasiKurirKendaraanFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahMediaInformasiKendaraanKurirKendaraanFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-informasi-kendaraan-kurir-bpkb":
		var data kurir_media_services.PayloadTambahMediaInformasiKendaraanKurirBPKBFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahInformasiKendaraanKurirBPKBFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-informasi-kendaraan-kurir-stnk":
		var data kurir_media_services.PayloadTambahMediaInformasiKendaraanKurirSTNKFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahInformasiKendaraanKurirSTNKFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-informasi-kurir-ktp":
		var data kurir_media_services.PayloadTambahMediaInformasiKurirKTPFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahMediaInformasiKurirKTPFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-pengiriman-non-ekspedisi-picked-up":
		var data kurir_media_services.PayloadTambahPengirimanPickedUpFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahMediaPengirimanPickedUpFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-pengiriman-non-ekspedisi-sampai":
		var data kurir_media_services.PayloadTambahPengirimanSampaiFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahMediaPengirimanSampaiFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-pengiriman-ekspedisi-picked-up":
		var data kurir_media_services.PayloadTambahPengirimanEkspedisiPickedUpFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahMediaPengirimanEkspedisiPickedUpFoto(ctx, data, db, ms)
	case "/kurir/media/tambah-foto-pengiriman-ekspedisi-sampai":
		var data kurir_media_services.PayloadTambahPengirimanEkspedisiSampaiAgentFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_media_services.TambahMediaPengirimanEkspedisiSampaiAgentFoto(ctx, data, db, ms)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(hasil.Status))
	json.NewEncoder(w).Encode(hasil)
}
