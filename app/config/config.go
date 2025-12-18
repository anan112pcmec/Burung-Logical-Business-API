package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mb_cud_queue_provisioning "github.com/anan112pcmec/Burung-backend-1/app/message_broker/provisioning/cud_exchange/queue"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	mb_cud_seeders "github.com/anan112pcmec/Burung-backend-1/app/message_broker/seeders/cud_exchange"
)

const (
	ENVFILE = "env"
	YAML    = "yaml"
	JSON    = "json"
)

type Environment struct {
	DB_MASTER_HOST, DB_MASTER_USER, DB_MASTER_PASS, DB_MASTER_NAME, DB_MASTER_PORT                                         string
	DB_REPLICA_SYSTEM_HOST, DB_REPLICA_SYSTEM_USER, DB_REPLICA_SYSTEM_PASS, DB_REPLICA_SYSTEM_NAME, DB_REPLICA_SYSTEM_PORT string
	DB_REPLICA_CLIENT_HOST, DB_REPLICA_CLIENT_USER, DB_REPLICA_CLIENT_PASS, DB_REPLICA_CLIENT_NAME, DB_REPLICA_CLIENT_PORT string
	RDSHOST, RDSPORT                                                                                                       string
	RDSAUTHDB, RDSSESSIONDB, RDSENGAGEMENTDB                                                                               int
	MEILIHOST, MEILIKEY, MEILIPORT                                                                                         string
	RMQ_HOST, RMQ_USER, RMQ_PASS, EXCHANGE, RMQ_PORT                                                                       string
	MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_SIGNED_URL_EXPIRE_SEC                                        string
	MINIO_USE_SSL                                                                                                          bool
}

type InternalDBReadWriteSystem struct {
	Write *gorm.DB
	Read  *gorm.DB
}

func (e *Environment) RunConnectionEnvironment() (
	db_system *InternalDBReadWriteSystem,
	db_replica_client *gorm.DB,
	redis_auth *redis.Client,
	redis_session *redis.Client,
	redis_engagement *redis.Client,
	search_engine meilisearch.ServiceManager,
	cud_publisher *mb_cud_publisher.Publisher,
	media_storage *minio.Client,
) {

	getDsn := func(host, user, pass, name, port string) string {
		return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
			host, user, pass, name, port)
	}

	dsn_master := getDsn(e.DB_MASTER_HOST, e.DB_MASTER_USER, e.DB_MASTER_PASS, e.DB_MASTER_NAME, e.DB_MASTER_PORT)
	dsn_replica_system := getDsn(e.DB_REPLICA_SYSTEM_HOST, e.DB_REPLICA_SYSTEM_USER, e.DB_REPLICA_SYSTEM_PASS, e.DB_REPLICA_SYSTEM_NAME, e.DB_REPLICA_SYSTEM_PORT)
	dsn_replica_client := getDsn(e.DB_REPLICA_CLIENT_HOST, e.DB_REPLICA_CLIENT_USER, e.DB_REPLICA_CLIENT_PASS, e.DB_REPLICA_CLIENT_NAME, e.DB_REPLICA_CLIENT_PORT)

	log.Println("🔍 Mencoba koneksi ke PostgreSQL...")
	log.Println("🔗 DSN:", dsn_master)

	var err error
	db_master, err := gorm.Open(postgres.Open(dsn_master), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // pakai level Warn agar log tidak terlalu ramai
	})
	if err != nil {
		log.Fatalf("❌ koneksi master Gagal konek ke PostgreSQL: %v", err)
	}

	// Coba koneksi langsung
	sqlDB, err := db_master.DB()
	if err != nil {
		log.Fatalf("❌ Gagal mendapatkan *sql.DB dari GORM: %v", err)
	}

	// Coba ping database untuk memastikan koneksi aktif
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Gagal ping ke PostgreSQL: %v", err)
	}

	// Atur pool koneksi
	sqlDB.SetMaxOpenConns(1000)
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	var currentDB string
	if err := db_master.Raw("SELECT current_database();").Scan(&currentDB).Error; err != nil {
		log.Printf("⚠️ Tidak bisa membaca nama database: %v", err)
	} else {
		log.Println("✅ Berhasil terkoneksi ke database:", currentDB)
	}

	// Koneksi ke replica_system
	db_replica_system, err := gorm.Open(postgres.Open(dsn_replica_system), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("❌ koneksi replica_system Gagal konek ke PostgreSQL: %v", err)
	}

	sqlDBReplicaSystem, err := db_replica_system.DB()
	if err != nil {
		log.Fatalf("❌ Gagal mendapatkan *sql.DB dari GORM (replica_system): %v", err)
	}

	if err := sqlDBReplicaSystem.Ping(); err != nil {
		log.Fatalf("❌ Gagal ping ke PostgreSQL (replica_system): %v", err)
	}

	sqlDBReplicaSystem.SetMaxOpenConns(1000)
	sqlDBReplicaSystem.SetMaxIdleConns(50)
	sqlDBReplicaSystem.SetConnMaxLifetime(time.Hour)

	var currentReplicaSystem string
	if err := db_replica_system.Raw("SELECT current_database();").Scan(&currentReplicaSystem).Error; err != nil {
		log.Printf("⚠️ Tidak bisa membaca nama database replica_system: %v", err)
	} else {
		log.Println("✅ Berhasil terkoneksi ke database replica_system:", currentReplicaSystem)
	}

	db_system = &InternalDBReadWriteSystem{
		Write: db_master,
		Read:  db_replica_system,
	}

	// Koneksi ke replica_client
	db_replica_client, err = gorm.Open(postgres.Open(dsn_replica_client), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("❌ koneksi replica_client Gagal konek ke PostgreSQL: %v", err)
	}

	sqlDBReplicaClient, err := db_replica_client.DB()
	if err != nil {
		log.Fatalf("❌ Gagal mendapatkan *sql.DB dari GORM (replica_client): %v", err)
	}

	if err := sqlDBReplicaClient.Ping(); err != nil {
		log.Fatalf("❌ Gagal ping ke PostgreSQL (replica_client): %v", err)
	}

	sqlDBReplicaClient.SetMaxOpenConns(100)
	sqlDBReplicaClient.SetMaxIdleConns(50)
	sqlDBReplicaClient.SetConnMaxLifetime(time.Hour)

	var currentReplicaClient string
	if err := db_replica_client.Raw("SELECT current_database();").Scan(&currentReplicaClient).Error; err != nil {
		log.Printf("⚠️ Tidak bisa membaca nama database replica_client: %v", err)
	} else {
		log.Println("✅ Berhasil terkoneksi ke database replica_client:", currentReplicaClient)
	}

	redis_auth = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", e.RDSHOST, e.RDSPORT),
		Password: "",
		DB:       e.RDSAUTHDB,
	})

	redis_session = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", e.RDSHOST, e.RDSPORT),
		Password: "",
		DB:       e.RDSSESSIONDB,
	})

	redis_engagement = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", e.RDSHOST, e.RDSPORT),
		Password: "",
		DB:       e.RDSENGAGEMENTDB,
	})

	connStr := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/internal_system_burung",
		e.RMQ_USER,
		e.RMQ_PASS,
		e.RMQ_HOST,
		e.RMQ_PORT,
	)
	message_broker, _ := amqp091.Dial(connStr)
	cud_ch, err := message_broker.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	cud_publisher = &mb_cud_publisher.Publisher{
		Ch: cud_ch,
		QueueCreate: &mb_cud_queue_provisioning.CreateQueue{
			ExchangeName: "mb.cud",
			QueueName:    mb_cud_seeders.Create,
			QueueBind:    mb_cud_queue_provisioning.CreateQueue{}.BindingName(),
			Durable:      true,
			AutoDelete:   false,
			Internal:     false,
			NoWait:       false,
			Exclusive:    false,
		},
		QueueUpdate: &mb_cud_queue_provisioning.UpdateQueue{
			ExchangeName: "mb.cud",
			QueueName:    mb_cud_seeders.Update,
			QueueBind:    mb_cud_queue_provisioning.UpdateQueue{}.BindingName(),
			Durable:      true,
			AutoDelete:   false,
			Internal:     false,
			NoWait:       false,
			Exclusive:    false,
		},
		QueueDelete: &mb_cud_queue_provisioning.DeleteQueue{
			ExchangeName: "mb.cud",
			QueueName:    mb_cud_seeders.Delete,
			QueueBind:    mb_cud_queue_provisioning.DeleteQueue{}.BindingName(),
			Durable:      true,
			AutoDelete:   false,
			Internal:     false,
			NoWait:       false,
			Exclusive:    false,
		},
		Mu: sync.Mutex{},
	}

	search_engine = meilisearch.New(fmt.Sprintf("http://%s:%s", e.MEILIHOST, e.MEILIPORT), meilisearch.WithAPIKey(e.MEILIKEY))

	media_storage, err_media := minio.New(e.MINIO_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(e.MINIO_ACCESS_KEY, e.MINIO_SECRET_KEY, ""),
		Secure: e.MINIO_USE_SSL,
	})
	if err_media != nil {
		log.Fatal("MinIO init error:", err)
	}

	return
}
