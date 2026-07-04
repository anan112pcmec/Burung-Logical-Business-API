package cache_db_entity_sessioning_seeders

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	sot_models "github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
)

func UpdateCacheSessionKey[
	T *sot_models.Pengguna | *sot_models.Seller | *sot_models.Kurir,
](
	ctx context.Context,
	data T,
	keySebelum string,
	rds_session *redis.Client,
) error {

	switch t := any(data).(type) {

	case *sot_models.Pengguna:
		if err := rds_session.Del(ctx, keySebelum).Err(); err != nil {
			fmt.Println("Gagal hapus key lama session")
			return err
		}

		cacheSessionKey := SetSessionKey[*sot_models.Pengguna](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data pengguna")
			return err
		}

	case *sot_models.Seller:
		if err := rds_session.Del(ctx, keySebelum).Err(); err != nil {
			fmt.Println("Gagal hapus key lama session")
			return err
		}

		cacheSessionKey := SetSessionKey[*sot_models.Seller](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data seller")
			return err
		}

	case *sot_models.Kurir:
		if err := rds_session.Del(ctx, keySebelum).Err(); err != nil {
			fmt.Println("Gagal hapus key lama session")
			return err
		}

		cacheSessionKey := SetSessionKey[*sot_models.Kurir](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data kurir")
			return err
		}

	default:
		return fmt.Errorf("gagal tipe tidak dikenal")
	}

	return nil
}

func UpdateCacheSessionData[
	T *sot_models.Pengguna | *sot_models.Seller | *sot_models.Kurir,
](
	ctx context.Context,
	data T,
	rds_session *redis.Client,
) error {

	switch t := any(data).(type) {

	case *sot_models.Pengguna:
		cacheSessionKey := SetSessionKey[*sot_models.Pengguna](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data pengguna")
			return err
		}

	case *sot_models.Seller:
		cacheSessionKey := SetSessionKey[*sot_models.Seller](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data seller")
			return err
		}

	case *sot_models.Kurir:
		cacheSessionKey := SetSessionKey[*sot_models.Kurir](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data kurir")
			return err
		}

	default:
		return fmt.Errorf("gagal tipe tidak dikenal")
	}

	return nil
}
