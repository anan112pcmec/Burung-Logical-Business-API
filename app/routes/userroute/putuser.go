package userroute

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	pengguna_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/media_services"
)

func PutUserHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client) {

	var hasil *response.ResponseMediaUpload
	ctx := r.Context()

	switch r.URL.Path {
	case "/user/media/upload-foto-profile":
		var data pengguna_media_services.PayloadUbahFotoProfilPengguna
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_media_services.UbahFotoProfilPengguna(ctx, data, db, ms)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(hasil.Status))
	json.NewEncoder(w).Encode(hasil)
}
