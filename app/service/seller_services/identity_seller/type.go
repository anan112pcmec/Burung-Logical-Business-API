package identity_seller

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	sot_threshold "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/threshold"
)

func (i *IdentitySeller) GetSessionKey() string {
	return fmt.Sprintf(
		"session_seller_%d_%s_%s",
		i.IdSeller,
		i.Username,
		i.EmailSeller,
	)
}

type IdentitySeller struct {
	IdSeller    int32  `json:"id_seller"`
	Username    string `json:"username_seller"`
	EmailSeller string `json:"email_seller"`
}

func (i IdentitySeller) UpThreshold(ctx context.Context, db *gorm.DB) bool {

	var id_data_order int64 = 0
	if err := db.WithContext(ctx).Model(&sot_threshold.ThresholdOrderSeller{}).Select("id").Where(&sot_threshold.ThresholdOrderSeller{
		IdSeller: i.IdSeller,
	}).Limit(1).Scan(&id_data_order).Error; err != nil {
		return false
	}

	if id_data_order == 0 {
		if err := db.WithContext(ctx).Create(&sot_threshold.ThresholdOrderSeller{
			IdSeller: i.IdSeller,
		}).Error; err != nil {
			return false
		}
	}

	return true
}

func (i *IdentitySeller) FallbackDB(ctx context.Context, db *gorm.DB) (model models.Seller, status bool) {
	var seller models.Seller

	_ = db.WithContext(ctx).Model(&models.Seller{}).Where(&models.Seller{
		ID:       i.IdSeller,
		Username: i.Username,
		Email:    i.EmailSeller,
	}).Limit(1).Take(&seller)

	if seller.ID == 0 {
		return seller, false
	}

	if seller.StatusSeller != "Online" {
		return seller, false
	}

	return seller, true
}

func (i *IdentitySeller) Validating(ctx context.Context, db *gorm.DB, rds *redis.Client) (model models.Seller, status bool) {
	var seller models.Seller
	if i.IdSeller == 0 {
		return seller, false
	}

	if i.Username == "" {
		return seller, false
	}

	if i.EmailSeller == "" {
		return seller, false
	}

	cacheSession := rds.HGetAll(ctx, i.GetSessionKey()).Val()
	id_seller, err_id := strconv.Atoi(cacheSession["id_seller"])
	if err_id != nil {
		return i.FallbackDB(ctx, db)
	}
	seller = models.Seller{
		ID:               int32(id_seller),
		Username:         cacheSession["username"],
		Nama:             cacheSession["nama"],
		Email:            cacheSession["email"],
		SellerDedication: cacheSession["seller_dedication"],
		JamOperasional:   cacheSession["jam_operasional"],
		Punchline:        cacheSession["punchline"],
		Password:         cacheSession["password_hash"],
		Deskripsi:        cacheSession["deskripsi"],
		StatusSeller:     cacheSession["status"],
	}

	return seller, true
}
