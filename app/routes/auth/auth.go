package auth

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/service/authservices"
)

type OTPkey struct {
	Value string `json:"otp_key"`
}

func HandleAuth(db *environment.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, rds_auth *redis.Client, rds_session *redis.Client, cud_publisher *mb_cud_publisher.Publisher) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/auth/user/registration":
		if r.Method == http.MethodPost {
			var data sot_models.Pengguna
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.PreUserRegistration(r.Context(), db, data.Username, data.Nama, data.Email, data.PasswordHash, rds_auth)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/user/login":
		if r.Method == http.MethodPost {
			var data sot_models.Pengguna
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.PenggunaLogin(r.Context(), db, data.Email, data.PasswordHash, rds_session, cud_publisher)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/user/registration/validate":
		if r.Method == http.MethodPost {
			var data OTPkey
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.ValidateUserRegistration(r.Context(), db, data.Value, rds_auth, cud_publisher)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/seller/registration":
		if r.Method == http.MethodPost {
			var data sot_models.Seller
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.PreSellerRegistration(r.Context(), db, data.Username, data.Nama, data.Email, data.Jenis, data.SellerDedication, data.Password, rds_auth)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/seller/login":
		if r.Method == http.MethodPost {
			var data sot_models.Seller
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.SellerLogin(r.Context(), db, data.Email, data.Password, rds_session, cud_publisher)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/seller/registration/validate":
		if r.Method == http.MethodPost {
			var data OTPkey
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.ValidateSellerRegistration(r.Context(), db, data.Value, rds_auth, cud_publisher)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/kurir/registration":
		if r.Method == http.MethodPost {
			var data sot_models.Kurir
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.PreKurirRegistration(r.Context(), db, data.Nama, data.Email, data.PasswordHash, data.Username, rds_auth)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/kurir/login":
		if r.Method == http.MethodPost {
			var data sot_models.Kurir
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.KurirLogin(r.Context(), db, data.Email, data.PasswordHash, rds_session, cud_publisher)
			json.NewEncoder(w).Encode(hasil)
			return
		}

	case "/auth/kurir/registration/validate":
		if r.Method == http.MethodPost {
			var data OTPkey
			if err := helper.DecodeJSONBody(r, &data); err != nil {
				http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasil := authservices.ValidateKurirRegistration(r.Context(), db, data.Value, rds_auth, cud_publisher)
			json.NewEncoder(w).Encode(hasil)
			return
		}
	}

	http.Error(w, "Endpoint tidak ditemukan", http.StatusNotFound)
}
