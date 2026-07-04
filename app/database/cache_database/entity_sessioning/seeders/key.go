package cache_db_entity_sessioning_seeders

import (
	"fmt"

	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
)

func SetSessionKey[T *sot_models.Pengguna | *sot_models.Seller | *sot_models.Kurir](i T) string {
	switch v := any(i).(type) {

	case *sot_models.Pengguna:
		return fmt.Sprintf(
			"session_user_%d_%s_%s",
			v.ID,
			v.Username,
			v.Email,
		)

	case *sot_models.Seller:
		return fmt.Sprintf(
			"session_seller_%d_%s_%s",
			v.ID,
			v.Username,
			v.Email,
		)

	case *sot_models.Kurir:
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
