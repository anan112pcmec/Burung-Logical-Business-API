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
	case "/seller/media/upload-foto-profile":
		var data seller_media_services.PayloadUbahFotoProfilSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_media_services.UbahFotoProfilSeller(ctx, data, db, ms)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(hasil.Status))
	json.NewEncoder(w).Encode(hasil)
}
