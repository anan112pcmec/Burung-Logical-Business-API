package seller_transaksi_services

import (
	"time"

	"github.com/anan112pcmec/Burung-backend-1/app/service/seller_services/identity_seller"
)

type PayloadApproveOrderTransaksi struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdTransaksi     int64                          `json:"id_transaksi"`
	Catatan         string                         `json:"catatan_approve"`
	IsAuto          bool                           `json:"auto_pengiriman"`
	AutoPengiriman  *time.Time                     `json:"waktu_auto_pengiriman"`
}

type PayloadKirimOrderTransaksi struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdTransaksi     int64                          `json:"id_transaksi"`
}

type PayloadUnApproveOrderTransaksi struct {
	IdentitasSeller identity_seller.IdentitySeller `json:"identitas_seller"`
	IdTransaksi     int64                          `json:"id_transaksi"`
	Catatan         string                         `json:"catatan_unapprove"`
}
