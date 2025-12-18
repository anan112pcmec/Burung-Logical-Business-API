package userroute

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	pengguna_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/media_services"
)

func PutUserHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {

	var hasil *response.ResponseMediaUpload
	ctx := r.Context()

	switch r.URL.Path {
	case "/user/media/ubah-foto-profile":
		var data pengguna_media_services.PayloadUbahFotoProfilPengguna
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_media_services.UbahFotoProfilPengguna(ctx, data, db, ms)
	case "/user/media/tambah-foto-review-barang":
		var data pengguna_media_services.PayloadTambahMediaReviewFoto
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_media_services.TambahMediaReviewFoto(ctx, data, db, ms)
	case "/user/media/tambah-video-review-barang":
		var data pengguna_media_services.PayloadTambahMediaReviewVideo
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_media_services.TambahMediaReviewVideo(ctx, data, db, ms)

	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
