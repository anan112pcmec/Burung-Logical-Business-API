package threshold

import "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"

type ThresholdTransaksiSeller struct {
	ID           int64         `gorm:"primaryKey;autoIncrement" json:"id_threshold_transaksi_seller"`
	IdSeller     int32         `gorm:"column:id_seller;not null" json:"id_seller_threshold_transaksi_seller"`
	Seller       models.Seller `gorm:"foreignKey:IdSeller;references:ID" json:"-"`
	BelumSelesai int64         `gorm:"column:belum_selesai;type:int8;not null;default:0" json:"belum_selesai_threshold_transaksi_seller"`
	Selesai      int64         `gorm:"column:selesai;type:int8;not null;default:0" json:"selesai_threshold_transaksi_seller"`
	Total        int64         `gorm:"column:total;type:int8;not null;default:0" json:"total_threshold_transaksi_seller"`
}

type ThresholdOrderSeller struct {
	ID         int64         `gorm:"primaryKey;autoIncrement" json:"id_threshold_order_seller"`
	IdSeller   int32         `gorm:"column:id_seller;not null" json:"id_seller_threshold_order_seller"`
	Seller     models.Seller `gorm:"foreignKey:IdSeller;references:ID" json:"-"`
	Dibatalkan int64         `gorm:"column:dibatalkan;type:int8;not null;default:0" json:"dibatalkan_threshold_order_seller"`
	Dibayar    int64         `gorm:"column:dibayar;type:int8;not null;default:0" json:"dibayar_threshold_order_seller"`
	Diproses   int64         `gorm:"column:diproses;type:int8;not null;default:0" json:"diproses_threshold_order_seller"`
	Waiting    int64         `gorm:"column:waiting;type:int8;not null;default:0" json:"waiting_threshold_order_seller"`
	Dikirim    int64         `gorm:"column:dikirim;type:int8;not null;default:0" json:"dikirim_threshold_order_seller"`
	Sampai     int64         `gorm:"column:sampai;type:int8;not null;default:0" json:"sampai_threshold_order_seller"`
	Total      int64         `gorm:"column:total;type:int8;not null;default:0" json:"total_threshold_order_seller"`
}
