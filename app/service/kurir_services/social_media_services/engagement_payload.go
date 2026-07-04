package kurir_social_media_services

import (
	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/service/kurir_services/identity_kurir"
)

type PayloadEngageSocialMedia struct {
	DataIdentitas identity_kurir.IdentitasKurir `json:"data_identitas_kurir"`
	Data          sot_models.EntitySocialMedia  `json:"data_social_media"`
}
