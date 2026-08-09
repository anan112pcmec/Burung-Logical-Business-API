package kurir_pengiriman_services

import (
	"github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/identity_kurir"
)

type PayloadAktifkanBidKurir struct {
	IdentitasKurir  identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	JenisPengiriman string                        `json:"jenis_pengiriman"`
	Mode            string                        `json:"mode"`
	Provinsi        string                        `json:"provinsi"`
	Kota            string                        `json:"kota"`
	IsEkspedisi     bool                          `json:"is_ekspedisi"`
	Alamat          string                        `json:"alamat"`
	Longitude       float64                       `json:"longitude"`
	Latitude        float64                       `json:"latitude"`
	MaxKg           int32                         `json:"max_kg"`
}

type PayloadUpdatePosisiBid struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir     int64                         `json:"id_bid_kurir"`
	Longitude      float64                       `json:"longitude"`
	Latitude       float64                       `json:"latitude"`
}

type PayloadAmbilPengirimanNonEks struct {
	IdentitasKurir  identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdPengiriman    int64                         `json:"id_pengiriman"`
	IdBid           int64                         `json:"id_bid"`
	JenisPengiriman string                        `json:"jenis_pengiriman"`
}

type PayloadAmbilPengirimanEks struct {
	IdentitasKurir  identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdPengiriman    int64                         `json:"id_pengiriman"`
	IdBid           int64                         `json:"id_bid"`
	JenisPengiriman string                        `json:"jenis_pengiriman"`
}

type PayloadLockSiapAntar struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir     int64                         `json:"id_bid_kurir"`
}

type PayloadPickedUpPengirimanNonEks struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir     int64                         `json:"id_bid_kurir"`
	IdPengiriman   int64                         `json:"id_pengiriman"`
	Lokasi         string                        `json:"lokasi"`
	Keterangan     string                        `json:"keterangan"`
	Latitude       float64                       `json:"latitude"`
	Longitude      float64                       `json:"longitude"`
}

type PayloadKirimPengirimanNonEks struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir     int64                         `json:"id_bid"`
	IdPengiriman   int64                         `json:"id_pengiriman"`
	Lokasi         string                        `json:"lokasi"`
	Keterangan     string                        `json:"keterangan"`
	Latitude       float64                       `json:"latitude"`
	Longitude      float64                       `json:"longitude"`
}

type PayloadUpdateInformasiPerjalananPengiriman struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdPengiriman   int64                         `json:"id_pengiriman"`
	Lokasi         string                        `json:"lokasi"`
	Keterangan     string                        `json:"keterangan"`
	Latitude       float64                       `json:"latitude"`
	Longitude      float64                       `json:"longitude"`
}

type PayloadSampaiPengirimanNonEks struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir     int64                         `json:"id_bid"`
	IdPengiriman   int64                         `json:"id_pengiriman"`
	Lokasi         string                        `json:"lokasi"`
	Keterangan     string                        `json:"keterangan"`
	Latitude       float64                       `json:"latitude"`
	Longitude      float64                       `json:"longitude"`
}

type PayloadPickedUpPengirimanEks struct {
	IdentitasKurir  identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir      int64                         `json:"id_bid_kurir"`
	IdPengirimanEks int64                         `json:"id_pengiriman"`
	Lokasi          string                        `json:"lokasi"`
	Keterangan      string                        `json:"keterangan"`
	Latitude        float64                       `json:"latitude"`
	Longitude       float64                       `json:"longitude"`
}

type PayloadKirimPengirimanEks struct {
	IdentitasKurir  identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir      int64                         `json:"id_bid_kurir"`
	IdPengirimanEks int64                         `json:"id_pengiriman"`
	Lokasi          string                        `json:"lokasi"`
	Keterangan      string                        `json:"keterangan"`
	Latitude        float64                       `json:"latitude"`
	Longitude       float64                       `json:"longitude"`
}

type PayloadUpdateInformasiPerjalananPengirimanEks struct {
	IdenititasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBid           int64                         `json:"id_bid"`
	IdPengirimanEks int64                         `json:"id_pengiriman_eks"`
	Lokasi          string                        `json:"lokasi"`
	Keterangan      string                        `json:"keterangan"`
	Latitude        float64                       `json:"latitude"`
	Longitude       float64                       `json:"longitude"`
}

type PayloadSampaiPengirimanEks struct {
	IdentitasKurir  identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	NoResiEkspedisi string                        `json:"no_resi_ekspedisi"`
	IdBidKurir      int64                         `json:"id_bid_kurir"`
	IdPengirimanEks int64                         `json:"id_pengiriman"`
	Lokasi          string                        `json:"lokasi"`
	Keterangan      string                        `json:"keterangan"`
	Latitude        float64                       `json:"latitude"`
	Longitude       float64                       `json:"longitude"`
}

type PayloadNonaktifkanBidKurir struct {
	IdentitasKurir identity_kurir.IdentitasKurir `json:"identitas_kurir"`
	IdBidKurir     int64                         `json:"id_bid_kurir"`
}
