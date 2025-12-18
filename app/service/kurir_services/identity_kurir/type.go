package identity_kurir

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	entity_enums "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums/entity"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

func (i *IdentitasKurir) GetSessionKey() string {
	return fmt.Sprintf(
		"session_kurir_%d_%s_%s",
		i.IdKurir,
		i.UsernameKurir,
		i.EmailKurir,
	)
}

type IdentitasKurir struct {
	IdKurir       int64  `json:"id_kurir"`
	UsernameKurir string `json:"username_kurir"`
	EmailKurir    string `json:"email_kurir"`
}

func (i *IdentitasKurir) FallbackDB(ctx context.Context, db *gorm.DB) (model models.Kurir, status bool) {
	var kurir models.Kurir

	_ = db.WithContext(ctx).Model(models.Kurir{}).Where(models.Kurir{
		ID:          i.IdKurir,
		Username:    i.UsernameKurir,
		Email:       i.EmailKurir,
		StatusKurir: entity_enums.Online,
	}).Take(&kurir)

	if kurir.ID == 0 {
		return kurir, false
	}

	return kurir, true
}

func (i *IdentitasKurir) Validating(ctx context.Context, db *gorm.DB, rds *redis.Client) (model models.Kurir, status bool) {
	var kurir models.Kurir
	if i.IdKurir == 0 {
		return kurir, false
	}

	if i.UsernameKurir == "" {
		return kurir, false
	}

	if i.EmailKurir == "" {
		return kurir, false
	}

	cacheSession := rds.HGetAll(ctx, i.GetSessionKey()).Val()
	id_kurir, err_id := strconv.Atoi(cacheSession["id_kurir"])

	if err_id != nil {
		return i.FallbackDB(ctx, db)
	}

	verified, err_verified := strconv.ParseBool(cacheSession["verified"])
	if err_verified != nil {
		return i.FallbackDB(ctx, db)
	}

	rawRating, err_rate := strconv.ParseFloat(cacheSession["rating"], 32)
	if err_rate != nil {
		return i.FallbackDB(ctx, db)
	}

	Rating := float32(rawRating)

	kurir = models.Kurir{
		ID:            int64(id_kurir),
		Nama:          cacheSession["nama"],
		Username:      cacheSession["session"],
		Email:         cacheSession["email"],
		Jenis:         cacheSession["jenis"],
		PasswordHash:  cacheSession["password_hash"],
		Deskripsi:     cacheSession["deskripsi"],
		StatusKurir:   cacheSession["status"],
		StatusBid:     cacheSession["status_bid"],
		VerifiedKurir: verified,
		Rating:        Rating,
		TipeKendaraan: cacheSession["tipe_kendaraan"],
	}

	return kurir, true
}
