package models

import (
	"time"

	"gorm.io/gorm"
)

type AlamatEkspedisi struct {
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id_alamat_ekspedisi"`
	Kota            string         `gorm:"index;column:kota;type:nama_kota" json:"kota"`
	NamaAlamat      string         `gorm:"column:nama_alamat;type:text" json:"nama"`
	Lokasi          string         `gorm:"column:lokasi;type:text;not null" json:"lokasi"`
	Longitude       float64        `gorm:"column:longitude;type:numeric(11,8);not null" json:"longitude"`
	Latitude        float64        `gorm:"column:latitude;type:numeric(11,8);not null" json:"latitude"`
	PengirimanCount int64          `gorm:"column:pengiriman_count;type:int8;not null;default:0" json:"pengiriman_count" `
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (AlamatEkspedisi) TableName() string {
	return "alamat_ekspedisi"
}

type KebijakanSistem struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id_kebijakan_sistem"`

	DitetapkanOleh string `gorm:"column:ditetapkan_oleh;type:varchar(100);not null" json:"ditetapkan_oleh_kebijakan_sistem"`
	IDAdmin        int64  `gorm:"column:id_admin;type:bigint" json:"id_admin_kebijakan_sistem"`           // Boleh null
	NamaAdmin      string `gorm:"column:nama_admin;type:varchar(100)" json:"nama_admin_kebijakan_sistem"` // Boleh null

	KomisiSistemPerTransaksi float32 `gorm:"column:komisi_sistem_per_transaksi;type:decimal(10,2);not null" json:"komisi_sistem_per_transaksi_kebijakan_sistem"`

	// seller strict
	LimitMembuatDiskonPersonal    int32 `gorm:"column:limit_membuat_diskon_personal;type:int4;not null" json:"limit_membuat_diskon_personal_kebijakan_sistem"`
	LimitMembuatDiskonDistributor int32 `gorm:"column:limit_membuat_diskon_distributor;type:int4;not null" json:"limit_membuat_diskon_distributor_kebijakan_sistem"`
	LimitMembuatDiskonBrand       int32 `gorm:"column:limit_membuat_diskon_brand;type:int4;not null" json:"limit_membuat_diskon_brand_kebijakan_sistem"`

	// kurir strict
	MaxJarakKmReguler           int `gorm:"column:max_jarak_km_reguler;type:int4;not null" json:"max_jarak_km_reguler_kebijakan_sistem"`
	MaxJarakKmExpress           int `gorm:"column:max_jarak_km_express;type:int4;not null" json:"max_jarak_km_express_kebijakan_sistem"`
	MaxJarakKmInstant           int `gorm:"column:max_jarak_km_instant;type:int4;not null" json:"max_jarak_km_instant_kebijakan_sistem"`
	EstimasiHariReguler         int `gorm:"column:estimasi_hari_reguler;type:int4;not null" json:"estimasi_hari_reguler_kebijakan_sistem"`
	EstimasiHariExpress         int `gorm:"column:estimasi_hari_express;type:int4;not null" json:"estimasi_hari_express_kebijakan_sistem"`
	EstimasiHariInstant         int `gorm:"column:estimasi_hari_instant;type:int4;not null" json:"estimasi_hari_instant_kebijakan_sistem"`
	TarifPengirimanRegulerPerKm int `gorm:"column:tarif_pengiriman_reguler;type:int4;not null" json:"tarif_pengiriman_reguler_kebijakan_sistem"`
	TarifPengirimanExpressPerKm int `gorm:"column:tarif_pengiriman_express;type:int4;not null" json:"tarif_pengiriman_express_kebijakan_sistem"`
	TarifPengirimanInstantPerKm int `gorm:"column:tarif_pengiriman_instant;type:int4;not null" json:"tarif_pengiriman_instant_kebijakan_sistem"`
	MaksimalBidKurirReguler     int `gorm:"column:maksimal_bid_kurir_reguler;type:int4;not null" json:"maksimal_bid_kurir_reguler_kebijakan_sistem"`
	MaksimalBidKurirExpress     int `gorm:"column:maksimal_bid_kurir_express;type:int4;not null" json:"maksimal_bid_kurir_express_kebijakan_sistem"`
	MaksimalBidKurirInstant     int `gorm:"column:maksimal_bid_kurir_instant;type:int4;not null" json:"maksimal_bid_kurir_instant_kebijakan_sistem"`
	JarakMasukEkspedisi         int `gorm:"column:jarak_masuk_ekspedisi;type:int4;not null" json:"jarak_masuk_ekspedisi_kebijakan_sistem"`
	TarifPerkg                  int `gorm:"column:tarif_per_kg;type:int4;not null" json:"tarif_per_kg_kebijakan_sistem"`

	Catatan       string    `gorm:"column:catatan;type:text;not null" json:"catatan_kebijakan_sistem"`
	BerlakuMulai  time.Time `gorm:"column:berlaku_mulai;type:timestamp;not null" json:"berlaku_mulai_kebijakan_sistem"`
	BerlakuSampai time.Time `gorm:"column:berlaku_sampai;type:timestamp" json:"berlaku_sampai_kebijakan_sistem"` // Boleh null
	StatusActive  bool      `gorm:"column:status_active;type:boolean;not null;default:true" json:"status_active_kebijakan_sistem"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (KebijakanSistem) TableName() string {
	return "kebijakan_sistem"
}
