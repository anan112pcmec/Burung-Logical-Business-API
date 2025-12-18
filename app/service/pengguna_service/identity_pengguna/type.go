package identity_pengguna

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

func (i *IdentityPengguna) GetSessionKey() string {
	return fmt.Sprintf(
		"session_user_%d_%s_%s",
		i.ID,
		i.Username,
		i.Email,
	)
}

type IdentityPengguna struct {
	ID       int64  `json:"id_pengguna"`
	Username string `json:"username_pengguna"`
	Email    string `json:"email_pengguna"`
}

func (i *IdentityPengguna) FallbackDB(ctx context.Context, db *gorm.DB) (model models.Pengguna, status bool) {
	var user models.Pengguna

	if err_validate := db.WithContext(ctx).Model(models.Pengguna{}).Where(models.Pengguna{
		ID:       i.ID,
		Username: i.Username,
		Email:    i.Email,
	}).Limit(1).Take(&user).Error; err_validate != nil {
		return user, false
	}

	return user, true
}

func (i *IdentityPengguna) Validating(ctx context.Context, db *gorm.DB, rds *redis.Client) (model models.Pengguna, status bool) {
	var user models.Pengguna

	if i.ID == 0 {
		return user, false
	}

	if i.Username == "" {
		return user, false
	}

	if i.Email == "" {
		return user, false
	}

	cacheSession := rds.HGetAll(ctx, i.GetSessionKey()).Val()
	id_pengguna, err_id := strconv.Atoi(cacheSession["id_user"])
	if err_id != nil {
		return i.FallbackDB(ctx, db)
	}

	user = models.Pengguna{
		ID:             int64(id_pengguna),
		Username:       cacheSession["username"],
		Nama:           cacheSession["nama"],
		Email:          cacheSession["email"],
		PasswordHash:   cacheSession["password_hash"],
		PinHash:        cacheSession["pin_hash"],
		StatusPengguna: cacheSession["status"],
	}

	return user, true
}
