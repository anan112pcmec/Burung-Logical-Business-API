package callback_payment_out

import (
	"context"
	"net/http"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

func UpdateStatusPaymentOut(ctx context.Context, data PayloadUpdateStatusPaymentOut, db *config.InternalDBReadWriteSystem) int16 {
	var id_payout int64 = 0
	var untuk string = ""
	if err := db.Read.WithContext(ctx).Model(&models.PayOutKurir{}).Select("id").Where(&models.PayOutKurir{
		IdDisbursment: data.ID,
	}).Limit(1).Take(&id_payout).Error; err != nil {
		if err := db.Read.WithContext(ctx).Model(&models.PayOutSeller{}).Select("id").Where(&models.PayOutSeller{
			IdDisbursment: data.ID,
		}).Limit(1).Take(&id_payout).Error; err != nil {
			return http.StatusNotFound
		} else {
			untuk = entity_enums.Seller
		}
	} else {
		untuk = entity_enums.Kurir
	}

	if id_payout == 0 {
		return http.StatusNotFound
	}

	switch untuk {
	case entity_enums.Kurir:
		if err := db.Write.WithContext(ctx).Model(&models.PayOutKurir{}).Where(&models.PayOutKurir{
			ID: id_payout,
		}).Update("status", data.Status).Error; err != nil {
			return http.StatusInternalServerError
		}
	case entity_enums.Seller:
		if err := db.Write.WithContext(ctx).Model(&models.PayOutSeller{}).Where(&models.PayOutSeller{
			ID: id_payout,
		}).Update("status", data.Status).Error; err != nil {
			return http.StatusInternalServerError
		}
	}

	return http.StatusOK
}
