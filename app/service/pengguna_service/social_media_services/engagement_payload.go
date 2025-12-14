package pengguna_social_media_service

import (
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/identity_pengguna"
)

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Struct Payload Tautkan Social Media
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEngageTautkanSocialMedia struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	Data              models.EntitySocialMedia           `json:"data_social_media"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Struct Payload Hapus Social Media Tertaut
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadEngageHapusSocialMedia struct {
	IdentitasPengguna identity_pengguna.IdentityPengguna `json:"identitas_pengguna"`
	IdSocialMedia     int64                              `json:"id_entity_social_media"`
	HapusSocialMedia  string                             `json:"delete_social_media"`
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Struct Payload Follow Atau Unfollow Seller
// ////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PayloadFollowOrUnfollowSeller struct {
	IdentitasUser identity_pengguna.IdentityPengguna `json:"data_identitas_pengguna"`
	IdSeller      int32                              `json:"id_seller_follow"`
}
