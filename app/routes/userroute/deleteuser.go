package userroute

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	pengguna_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/alamat_services"
	pengguna_service "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/barang_services"
	pengguna_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/media_services"
	pengguna_social_media_service "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/social_media_services"
	pengguna_transaction_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/transaction_services"
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/transaction_services/response_transaction_pengguna"
	pengguna_wishlist_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/wishlist_services"

)

func DeleteUserHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, ms *minio.Client) {
	var hasil *response.ResponseForm
	ctx := r.Context()

	switch r.URL.Path {
	case "/user/barang/komentar-barang/hapus":
		var data pengguna_service.PayloadHapusKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.HapusKomentarBarang(ctx, data, db)
	case "/user/barang/komentar-child/hapus":
		var data pengguna_service.PayloadHapusChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.HapusChildKomentar(ctx, data, db)
	case "/user/barang/keranjang-barang/hapus":
		var data pengguna_service.PayloadHapusDataKeranjangBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.HapusKeranjangBarang(ctx, data, db)
	case "/user/alamat/hapus-alamat":
		var data pengguna_alamat_services.PayloadHapusAlamatPengguna
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_alamat_services.HapusAlamatPengguna(ctx, data, db)
	case "/user/transaksi/batal-checkout-barang":
		var data response_transaction_pengguna.ResponseDataCheckout
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_transaction_services.BatalCheckoutUser(data, db)
	case "/user/transaksi/payment-gateaway-snap-gagal":
		var data response_transaction_pengguna.SnapTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_transaction_services.BatalTransaksi(ctx, data, db)
	case "/user/social-media/unfollow-seller":
		var data pengguna_social_media_service.PayloadFollowOrUnfollowSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_social_media_service.UnfollowSeller(ctx, data, db)
	case "/user/wishlist/hapus-barang":
		var data pengguna_wishlist_services.PayloadHapusBarangDariWishlist
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_wishlist_services.HapusBarangDariWishlist(ctx, data, db)
	case "/user/media/hapus-foto-profile":
		var data pengguna_media_services.PayloadHapusFotoProfilPengguna
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_media_services.HapusFotoProfilPengguna(ctx, data, db, ms)
	default:
		hasil = &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: "User Services",
			Payload:  "Gagal Coba Lagi Nanti",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
