package cache_db_entity_sessioning_seeders

import (
	"fmt"

	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

func SetSessionKey[T *models.Pengguna | *models.Seller | *models.Kurir](i T) string {
	switch v := any(i).(type) {

	case *models.Pengguna:
		return fmt.Sprintf(
			"session_user_%d_%s_%s",
			v.ID,
			v.Username,
			v.Email,
		)

	case *models.Seller:
		return fmt.Sprintf(
			"session_seller_%d_%s_%s",
			v.ID,
			v.Username,
			v.Email,
		)

	case *models.Kurir:
		return fmt.Sprintf(
			"session_kurir_%d_%s_%s",
			v.ID,
			v.Username,
			v.Email,
		)

	default:
		panic("unsupported identity type")
	}
}
