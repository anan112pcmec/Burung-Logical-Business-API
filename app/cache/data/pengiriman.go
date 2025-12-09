package data_cache

import "github.com/anan112pcmec/Burung-backend-1/app/database/models"

// Map nested untuk alamat ekspedisi
var DataAlamatEkspedisi map[string]map[int64]models.AlamatEkspedisi

type JenisPengiriman struct {
	Nama         string `json:"nama"`
	Rules        string `json:"rules"`
	MaxJarakKm   int64  `json:"max_jarak_km"`
	EstimasiHari int8   `json:"estimasi_hari"`
	Harga        int64  `json:"harga"`
}

var OperationalPengirimanData struct {
	Version   string `json:"version"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt string `json:"updated_at"`

	CommittedOperationalData struct {
		DataTarifPengiriman struct {
			TarifSistem     int64 `json:"tarif_sistem"`
			TarifKurirPerKm int64 `json:"tarif_kurir_per_km"`
			TarifKurirPerKg int64 `json:"tarif_kurir_per_kg"`
		} `json:"data_tarif_pengiriman"`

		DataJenisPengiriman []JenisPengiriman `json:"data_jenis_pengiriman"`
	} `json:"committed_operational_data"` // perbaiki typo
	Dokumentasi string `json:"dokumentasi"`
}

var DataTarifPengiriman struct {
	TarifSistem     int64
	TarifKurirPerKm int64
	TarifKurirPerKg int64
}

var DataTarifJenisPengiriman map[string]JenisPengiriman

var BucketFotoName string
var BucketVideoName string
var BucketDokumenName string
