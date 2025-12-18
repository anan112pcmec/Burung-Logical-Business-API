package kurir

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	kurir_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/alamat_services"
	kurir_informasi_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/informasi_services"
	kurir_rekening_services "github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/rekening_services"
)

func PostKurirHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	ctx := r.Context()
	var hasil *response.ResponseForm
	switch r.URL.Path {
	case "/kurir/informasi/ajukan-informasi-kendaraan":
		var data kurir_informasi_services.PayloadInformasiDataKendaraan
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_informasi_services.AjukanInformasiKendaraan(ctx, data, db)

	case "/kurir/informasi/ajukan-informasi-kurir":
		var data kurir_informasi_services.PayloadInformasiDataKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_informasi_services.AjukanInformasiKurir(ctx, data, db)

	case "/kurir/alamat/masukan-alamat":
		var data kurir_alamat_services.PayloadMasukanAlamatKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_alamat_services.MasukanAlamatKurir(ctx, data, db)
	case "/kurir/rekening/masukan-rekening":
		var data kurir_rekening_services.PayloadMasukanRekeningKurir
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = kurir_rekening_services.MasukanRekeningKurir(ctx, data, db)

	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
