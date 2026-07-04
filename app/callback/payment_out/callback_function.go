package callback_payment_out

import (
	"context"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
	mb_cud_serializer "github.com/anan112pcmec/Burung-backend-1/app/message_broker/serializer/cud_serializer"
	"github.com/anan112pcmec/Burung-backend-1/app/settings"
)

func UpdateStatusPaymentOut(ctx context.Context, data PayloadUpdateStatusPaymentOut, db *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) int16 {
	var id_payout int64 = 0
	var untuk string = ""

	// 1. Cek di PayOutKurir
	if err := db.Read.WithContext(ctx).Model(&sot_models.PayOutKurir{}).Select("id").Where(&sot_models.PayOutKurir{
		IdDisbursment: data.ID,
	}).Limit(1).Take(&id_payout).Error; err != nil {

		// 2. Cek di PayOutSeller jika di Kurir tidak ada
		if err := db.Read.WithContext(ctx).Model(&sot_models.PayOutSeller{}).Select("id").Where(&sot_models.PayOutSeller{
			IdDisbursment: data.ID,
		}).Limit(1).Take(&id_payout).Error; err != nil {

			// 3. Cek di PayOutSistem jika di Seller tidak ada
			if err := db.Read.WithContext(ctx).Model(&sot_models.PayOutSistem{}).Select("id").Where(&sot_models.PayOutSistem{
				IdDisburstment: data.ID,
			}).Limit(1).Take(&id_payout).Error; err != nil {
				return http.StatusNotFound
			} else {
				untuk = "sistem" // Pastikan enum ini ada di package Anda, misal: "SISTEM" atau "SYSTEM"
			}

		} else {
			untuk = entity_enums.Seller
		}
	} else {
		untuk = entity_enums.Kurir
	}

	if id_payout == 0 {
		return http.StatusNotFound
	}

	// Proses Update Status berdasarkan tipe 'untuk'
	switch untuk {
	case entity_enums.Kurir:
		if err := db.Write.WithContext(ctx).Model(&sot_models.PayOutKurir{}).Where(&sot_models.PayOutKurir{
			ID: id_payout,
		}).Update("status", data.Status).Error; err != nil {
			return http.StatusInternalServerError
		} else {
			go func(IPK int64, read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
				konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
				defer cancel()

				var dataPayoutKurir sot_models.PayOutKurir
				if err := read.WithContext(konteks).Model(&sot_models.PayOutKurir{}).Where(&sot_models.PayOutKurir{
					ID: IPK,
				}).Limit(1).Scan(&dataPayoutKurir).Error; err != nil && err == gorm.ErrRecordNotFound {
					fmt.Println("Gagal mendapatkan data payout")
					return
				}
				updatePayoutKurirPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPayoutKurir).SetTableName(dataPayoutKurir.TableName()).SetRole(mb_cud_seeders.Sistem)
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, publisher, updatePayoutKurirPublish); err != nil {
					fmt.Println("Gagal update payout gagal publish")
				}
			}(id_payout, db.Read, cud_publisher)
		}
	case entity_enums.Seller:
		if err := db.Write.WithContext(ctx).Model(&sot_models.PayOutSeller{}).Where(&sot_models.PayOutSeller{
			ID: id_payout,
		}).Update("status", data.Status).Error; err != nil {
			return http.StatusInternalServerError
		} else {
			go func(IPS int64, read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
				konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
				defer cancel()

				var dataPayoutSeller sot_models.PayOutSeller
				if err := read.WithContext(konteks).Model(&sot_models.PayOutSeller{}).Where(&sot_models.PayOutSeller{
					ID: IPS,
				}).Limit(1).Scan(&dataPayoutSeller).Error; err != nil && err == gorm.ErrRecordNotFound {
					fmt.Println("Gagal menemukan data payout")
					return
				}

				updatePayoutSellerPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPayoutSeller).SetTableName(dataPayoutSeller.TableName()).SetRole(mb_cud_seeders.Sistem)
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, cud_publisher, updatePayoutSellerPublish); err != nil {
					fmt.Println("Gagal publish update payout seller")
				}
			}(id_payout, db.Read, cud_publisher)
		}
	case "sistem":
		if err := db.Write.WithContext(ctx).Model(&sot_models.PayOutSistem{}).Where(&sot_models.PayOutSistem{
			ID: id_payout,
		}).Update("status", data.Status).Error; err != nil {
			return http.StatusInternalServerError
		} else {
			go func(IPS int64, read *gorm.DB, publisher *mb_cud_publisher.Publisher) {
				konteks, cancel := context.WithTimeout(context.Background(), settings.TimeoutContext)
				defer cancel()

				var dataPayoutSistem sot_models.PayOutSistem
				if err := read.WithContext(konteks).Model(&sot_models.PayOutSistem{}).Where(&sot_models.PayOutSistem{
					ID: IPS,
				}).Limit(1).Scan(&dataPayoutSistem).Error; err != nil && err == gorm.ErrRecordNotFound {
					fmt.Println("Gagal menemukan data payout")
					return
				}

				updatePayoutSellerPublish := mb_cud_serializer.NewJsonPayload().SetPayload(dataPayoutSistem).SetTableName(dataPayoutSistem.TableName()).SetRole(mb_cud_seeders.Sistem)
				if err := mb_cud_publisher.UpdatePublish[*mb_cud_serializer.PublishPayloadJson](konteks, cud_publisher, updatePayoutSellerPublish); err != nil {
					fmt.Println("Gagal publish update payout seller")
				}
			}(id_payout, db.Read, cud_publisher)

		}
	}

	return http.StatusOK
}
