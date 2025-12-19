package cache_db_entity_sessioning_seeders

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
)

func UpdateCacheSessionKey[
	T *models.Pengguna | *models.Seller | *models.Kurir,
](
	ctx context.Context,
	data T,
	keySebelum string,
	rds_session *redis.Client,
) error {

	switch t := any(data).(type) {

	case *models.Pengguna:
		if err := rds_session.Del(ctx, keySebelum).Err(); err != nil {
			fmt.Println("Gagal hapus key lama session")
			return err
		}

		cacheSessionKey := SetSessionKey[*models.Pengguna](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data pengguna")
			return err
		}

	case *models.Seller:
		if err := rds_session.Del(ctx, keySebelum).Err(); err != nil {
			fmt.Println("Gagal hapus key lama session")
			return err
		}

		cacheSessionKey := SetSessionKey[*models.Seller](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data seller")
			return err
		}

	case *models.Kurir:
		if err := rds_session.Del(ctx, keySebelum).Err(); err != nil {
			fmt.Println("Gagal hapus key lama session")
			return err
		}

		cacheSessionKey := SetSessionKey[*models.Kurir](t)
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
	T *models.Pengguna | *models.Seller | *models.Kurir,
](
	ctx context.Context,
	data T,
	rds_session *redis.Client,
) error {

	switch t := any(data).(type) {

	case *models.Pengguna:
		cacheSessionKey := SetSessionKey[*models.Pengguna](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data pengguna")
			return err
		}

	case *models.Seller:
		cacheSessionKey := SetSessionKey[*models.Seller](t)
		cacheData := helper.StructToJSONMap(t)

		if err := rds_session.HSet(ctx, cacheSessionKey, cacheData).Err(); err != nil {
			fmt.Println("Gagal update session data seller")
			return err
		}

	case *models.Kurir:
		cacheSessionKey := SetSessionKey[*models.Kurir](t)
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
