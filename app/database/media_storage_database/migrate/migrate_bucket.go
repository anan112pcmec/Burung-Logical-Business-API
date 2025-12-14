package media_storage_database_migrate

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"

	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
)

func MigrateBucketMediaStorage(ms *minio.Client) bool {
	ctx := context.Background()
	var bucketPublicName = []string{
		media_storage_database_seeders.BucketFotoName,
		media_storage_database_seeders.BucketVideoName,
		media_storage_database_seeders.BucketDokumenName,
	}

	for i := 0; i < len(bucketPublicName); i++ {
		exists, err := ms.BucketExists(ctx, bucketPublicName[i])
		if err != nil {
			fmt.Printf("Bucket %s tidak ada", bucketPublicName[i])
			fmt.Println(err)
		}

		if !exists {
			if err := ms.MakeBucket(ctx, bucketPublicName[i], minio.MakeBucketOptions{}); err != nil {
				fmt.Printf("Gagal membuat bucket %s", bucketPublicName[i])
				fmt.Println(err)
				continue
			}

		} else {
			fmt.Printf("Bucket %s sudah ada", bucketPublicName[i])
		}

		policy := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": { "AWS": ["*"] },
						"Action": ["s3:GetObject"],
						"Resource": ["arn:aws:s3:::%s/*"]
					}
				]
			}`, bucketPublicName[i])

		err = ms.SetBucketPolicy(ctx, bucketPublicName[i], policy)
		if err != nil {
			log.Fatal(err)
			return false
		}
	}

	return true
}
