package userroute

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	pengguna_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/alamat_services"
	pengguna_service "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/barang_services"
	pengguna_credential_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/credential_services"
	pengguna_social_media_service "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/social_media_services"
	pengguna_transaction_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/transaction_services"
	pengguna_wishlist_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/wishlist_services"
)

func PostUserHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseForm
	ctx := r.Context()

	switch r.URL.Path {
	case "/user/barang/komentar-barang/tambah":
		var data pengguna_service.PayloadMasukanKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.MasukanKomentarBarang(ctx, data, db, rds_session, mb_cud_publisher)
	case "/user/barang/komentar-child/tambah":
		var data pengguna_service.PayloadMasukanChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.MasukanChildKomentar(ctx, data, db, rds_session, mb_cud_publisher)

	case "/user/barang/komentar-child-mention/tambah":
		var data pengguna_service.PayloadMentionChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.MentionChildKomentar(ctx, data, db, rds_session, mb_cud_publisher)
	case "/user/barang/keranjang-barang/tambah":
		var data pengguna_service.PayloadTambahDataKeranjangBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_service.TambahKeranjangBarang(ctx, data, db, rds_session, mb_cud_publisher)
	case "/user/credential/membuat-pin":
		var data pengguna_credential_services.PayloadMembuatPinPengguna
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_credential_services.MembuatSecretPinPengguna(ctx, data, db)
	case "/user/alamat/membuat-alamat":
		var data pengguna_alamat_services.PayloadMasukanAlamatPengguna
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_alamat_services.MasukanAlamatPengguna(ctx, data, db, rds_session, mb_cud_publisher)
	case "/user/transaksi/checkout-barang":
		var data pengguna_transaction_services.PayloadCheckoutBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_transaction_services.CheckoutBarangUser(ctx, data, db)
	case "/user/social-media/follow-seller":
		var data pengguna_social_media_service.PayloadFollowOrUnfollowSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_social_media_service.FollowSeller(ctx, data, db)
	case "/user/wishlist/tambah-barang":
		var data pengguna_wishlist_services.PayloadTambahBarangKeWishlist
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = pengguna_wishlist_services.TambahBarangKeWishlist(ctx, data, db)
	default:
		hasil = &response.ResponseForm{
			Status:   http.StatusBadRequest,
			Services: "Seller Services",
			Payload:  "Gagal Coba Lagi Nanti",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hasil)
}
