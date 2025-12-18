package seller

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	seller_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/alamat_services"
	seller_service "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/barang_services"
	seller_credential_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/credential_services"
	seller_diskon_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/diskon_services"
	seller_etalase_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/etalase_services"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/jenis_seller_services"
)

func PostSellerHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, rds_auth *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseForm

	ctx := r.Context()

	switch r.URL.Path {
	case "/seller/masukan_barang":
		var data seller_service.PayloadMasukanBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.MasukanBarangInduk(ctx, db, data)
	case "/seller/komentar-barang/tambah":
		var data seller_service.PayloadMasukanKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.MasukanKomentarBarang(ctx, data, db)
	case "/seller/komentar-child/tambah":
		var data seller_service.PayloadMasukanChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.MasukanChildKomentar(ctx, data, db)
	case "/seller/komentar-child-mention/tambah":
		var data seller_service.PayloadMentionChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.MentionChildKomentar(ctx, data, db)
	case "/seller/tambah_kategori_barang":
		var data seller_service.PayloadTambahKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.TambahKategoriBarang(ctx, db, data)
	case "/seller/credential/tambah-rekening":
		var data seller_credential_services.PayloadTambahkanNorekSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.TambahRekeningSeller(ctx, data, db)
	case "/seller/alamat/tambah-alamat-gudang":
		var data seller_alamat_services.PayloadTambahAlamatGudang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_alamat_services.TambahAlamatGudang(ctx, data, db)
	case "/seller/diskon/tambah-diskon":
		var data seller_diskon_services.PayloadTambahDiskonProduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.TambahDiskonProduk(ctx, data, db)
	case "/seller/diskon/masukan-barang":
		var data seller_diskon_services.PayloadTetapkanDiskonPadaBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.TetapKanDiskonPadaBarang(ctx, data, db)
	case "/seller/etalase/tambah-etalase":
		var data seller_etalase_services.PayloadMenambahEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.TambahEtalaseSeller(ctx, data, db)
	case "/seller/etalase/tambah-barang-ke-etalase":
		var data seller_etalase_services.PayloadTambahkanBarangKeEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.TambahkanBarangKeEtalase(ctx, data, db)
	case "/seller/jenis/ajukan-data-distributor":
		var data jenis_seller_services.PayloadMasukanDataDistributor
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.MasukanDataDistributor(ctx, data, db)
	case "/seller/jenis/ajukan-data-brand":
		var data jenis_seller_services.PayloadMasukanDataBrand
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.MasukanDataBrand(ctx, data, db)
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
