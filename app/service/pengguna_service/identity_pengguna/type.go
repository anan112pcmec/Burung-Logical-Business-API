package identity_pengguna

import (
	"context"

	"gorm.io/gorm"

	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

type IdentityPengguna struct {
	ID       int64  `json:"id_pengguna"`
	Username string `json:"username_pengguna"`
	Email    string `json:"email_pengguna"`
}

func (i IdentityPengguna) Validating(ctx context.Context, db *gorm.DB) (model models.Pengguna, status bool) {
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

	if err_validate := db.WithContext(ctx).Model(models.Pengguna{}).Where(models.Pengguna{
		ID:       i.ID,
		Username: i.Username,
		Email:    i.Email,
	}).Limit(1).Take(&user).Error; err_validate != nil {
		return user, false
	}

	return user, true
}
