package seller

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/response"
	seller_alamat_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/alamat_services"
	seller_barang_service "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/barang_services"
	seller_service "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/barang_services"
	seller_credential_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/credential_services"
	seller_diskon_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/diskon_services"
	seller_etalase_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/etalase_services"
	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/jenis_seller_services"
	seller_profiling_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/profiling_services"
	seller_social_media_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/social_media_services"
	seller_transaksi_services "github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/transaksi_services"
)

func PatchSellerHandler(db *environment.InternalDBReadWriteSystem, w http.ResponseWriter, r *http.Request, rds_auth *redis.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) {
	var hasil *response.ResponseForm

	switch r.URL.Path {
	case "/seller/edit_barang":
		var data seller_service.PayloadEditBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditBarangInduk(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/edit_kategori_barang":
		var data seller_service.PayloadEditKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditKategoriBarang(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/edit/stok-barang":
		var data seller_service.PayloadEditStokKategoriBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditStokKategoriBarang(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/profiling/personal-update":
		var data seller_profiling_services.PayloadUpdateProfilePersonalSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_profiling_services.UpdatePersonalSeller(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/profiling/info-general-update":
		var data seller_profiling_services.PayloadUpdateInfoGeneralSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_profiling_services.UpdateInfoGeneralPublic(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/credential/update-password":
		var data seller_credential_services.PayloadPreUbahPasswordSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.PreUbahPasswordSeller(r.Context(), data, db, rds_auth, rds_session, mb_cud_publisher)
	case "/seller/credential/validate-password-otp":
		var data seller_credential_services.PayloadValidateUbahPasswordSellerOTP
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.ValidateUbahPasswordSeller(r.Context(), data, db, rds_auth, rds_session, mb_cud_publisher)

	case "/seller/alamat/edit-alamat-gudang":
		var data seller_alamat_services.PayloadEditAlamatGudang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_alamat_services.EditAlamatGudang(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "seller/barang/set-as-origin-kategori-barang":
		var data seller_barang_service.PayloadSetAsOriginKategoriBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_barang_service.SetAsOriginKategoriBarang(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/ubah-harga-kategori-barang":
		var data seller_barang_service.PayloadUbahHargaKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_barang_service.UbahHargaKategoriBarang(r.Context(), db, data, rds_auth, mb_cud_publisher)
	case "/seller/barang/edit-alamat-barang-induk":
		var data seller_service.PayloadEditAlamatBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditAlamatGudangBarangInduk(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/social-media/social-media-engage":
		var data seller_social_media_services.PayloadEngageSocialMedia
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_social_media_services.EngageSocialMediaSeller(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/barang/down-barang-induk":
		var data seller_service.PayloadDownBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.DownStokBarangInduk(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/down-kategori-barang":
		var data seller_service.PayloadDownKategoriBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.DownKategoriBarang(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/up-barang-induk":
		var data seller_service.PayloadUpBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.UpBarangInduk(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/up-kategori-barang":
		var data seller_service.PayloadUpKategoriBarang
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.UpKategoriBarang(r.Context(), db, data, rds_session, mb_cud_publisher)
	case "/seller/barang/edit-rekening-barang":
		var data seller_service.PayloadEditRekeningBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditRekeningBarangInduk(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/barang/edit-alamat-kategori":
		var data seller_service.PayloadEditAlamatBarangKategori
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditAlamatGudangBarangKategori(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/komentar-barang/edit":
		var data seller_service.PayloadEditKomentarBarangInduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditKomentarBarang(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/komentar-child/edit":
		var data seller_service.PayloadEditChildKomentar
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_service.EditChildKomentar(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/rekening/edit-rekening":
		var data seller_credential_services.PayloadEditNorekSeler
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.EditRekeningSeller(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/rekening/set-default-rekening":
		var data seller_credential_services.PayloadSetDefaultRekeningSeller
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_credential_services.SetDefaultRekeningSeller(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/diskon/edit-diskon":
		var data seller_diskon_services.PayloadEditDiskonProduk
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_diskon_services.EditDiskonProduk(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/etalase/edit-etalase":
		var data seller_etalase_services.PayloadEditEtalase
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_etalase_services.EditEtalaseSeller(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/jenis/edit-data-distributor":
		var data jenis_seller_services.PayloadEditDataDistributor
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.EditDataDistributor(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/jenis/edit-data-brand":
		var data jenis_seller_services.PayloadEditDataBrand
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = jenis_seller_services.EditDataBrand(r.Context(), data, db, rds_session, mb_cud_publisher)
	case "/seller/transaction/approve-order":
		var data seller_transaksi_services.PayloadApproveOrderTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_transaksi_services.ApproveOrderTransaksi(r.Context(), data, db, rds_auth, rds_session, mb_cud_publisher)
	case "/seller/transaction/kirim-barang":
		var data seller_transaksi_services.PayloadKirimOrderTransaksi
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		hasil = seller_transaksi_services.KirimOrderTransaksi(r.Context(), data, db, rds_session, mb_cud_publisher)
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
