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
	seller_profiling_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/profiling_services"
	seller_social_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/social_media_services"
	seller_transaksi_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/transaksi_services"
)

func PatchSellerHandler(db *config.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, rds_auth *redis.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseForm

	ctx := r.Context()

	switch r.URL.Path {
	case "/seller/edit_barang":
		var data seller_service.PayloadEditBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditBarangInduk(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/edit_kategori_barang":
		var data seller_service.PayloadEditKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditKategoriBarang(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/edit/stok-barang":
		var data seller_service.PayloadEditStokKategoriBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditStokKategoriBarang(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/profiling/personal-update":
		var data seller_profiling_services.PayloadUpdateProfilePersonalSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_profiling_services.UpdatePersonalSeller(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/profiling/info-general-update":
		var data seller_profiling_services.PayloadUpdateInfoGeneralSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_profiling_services.UpdateInfoGeneralPublic(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/credential/update-password":
		var data seller_credential_services.PayloadPreUbahPasswordSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.PreUbahPasswordSeller(ctx, data, db, rds_auth, rds_session, mb_cud_publisher)
	case "/seller/credential/validate-password-otp":
		var data seller_credential_services.PayloadValidateUbahPasswordSellerOTP
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.ValidateUbahPasswordSeller(data, db, rds_auth, rds_session, mb_cud_publisher)

	case "/seller/alamat/edit-alamat-gudang":
		var data seller_alamat_services.PayloadEditAlamatGudang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_alamat_services.EditAlamatGudang(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/barang/edit-alamat-barang-induk":
		var data seller_service.PayloadEditAlamatBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditAlamatGudangBarangInduk(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/social-media/social-media-engage":
		var data seller_social_media_services.PayloadEngageSocialMedia
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_social_media_services.EngageSocialMediaSeller(ctx, data, db, rds_session)
	case "/seller/barang/down-barang-induk":
		var data seller_service.PayloadDownBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.DownStokBarangInduk(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/down-kategori-barang":
		var data seller_service.PayloadDownKategoriBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.DownKategoriBarang(ctx, db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/edit-rekening-barang":
		var data seller_service.PayloadEditRekeningBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditRekeningBarangInduk(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/barang/edit-alamat-kategori":
		var data seller_service.PayloadEditAlamatBarangKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditAlamatGudangBarangKategori(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/komentar-barang/edit":
		var data seller_service.PayloadEditKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditKomentarBarang(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/komentar-child/edit":
		var data seller_service.PayloadEditChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditChildKomentar(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/rekening/edit-rekening":
		var data seller_credential_services.PayloadEditNorekSeler
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.EditRekeningSeller(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/rekening/set-default-rekening":
		var data seller_credential_services.PayloadSetDefaultRekeningSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.SetDefaultRekeningSeller(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/diskon/edit-diskon":
		var data seller_diskon_services.PayloadEditDiskonProduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.EditDiskonProduk(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/etalase/edit-etalase":
		var data seller_etalase_services.PayloadEditEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.EditEtalaseSeller(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/jenis/edit-data-distributor":
		var data jenis_seller_services.PayloadEditDataDistributor
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.EditDataDistributor(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/jenis/edit-data-brand":
		var data jenis_seller_services.PayloadEditDataBrand
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.EditDataBrand(ctx, data, db, rds_session, mb_cud_publisher)
	case "/seller/transaction/approve-order":
		var data seller_transaksi_services.PayloadApproveOrderTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_transaksi_services.ApproveOrderTransaksi(ctx, data, db, rds_auth, rds_session, mb_cud_publisher)
	case "/seller/transaction/kirim-barang":
		var data seller_transaksi_services.PayloadKirimOrderTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_transaksi_services.KirimOrderTransaksi(ctx, data, db, rds_session, mb_cud_publisher)
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
