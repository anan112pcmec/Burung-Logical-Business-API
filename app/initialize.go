package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	routes "github.com/anan112pcmec/Burung-backend-1/app/Routes"
	maintain_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/maintain"
	media_storage_database_migrate "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/migrate"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/migrate"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/models"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_exchange_provisioning "github.com/anan112pcmec/Burung-backend-1/app/message_broker/provisioning/cud_exchange"
)

func Getenvi(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Run() {
	// 1. Load Environment & Configuration
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	rdsauth, _ := strconv.Atoi(Getenvi("RDSENTITY", "0"))
	rdssession, _ := strconv.Atoi(Getenvi("RDSAUTH", "0"))
	minioSSl, _ := strconv.ParseBool(Getenvi("MINIO_USE_SSL", "NIL"))

	env := environment.Environment{
		DB_MASTER_HOST: Getenvi("DB_MASTER_HOST", "NIL"),
		DB_MASTER_USER: Getenvi("DB_MASTER_USER", "NIL"),
		DB_MASTER_PASS: Getenvi("DB_MASTER_PASS", "NIL"),
		DB_MASTER_NAME: Getenvi("DB_MASTER_NAME", "NIL"),
		DB_MASTER_PORT: Getenvi("DB_MASTER_PORT", "NIL"),

		DB_REPLICA_SYSTEM_HOST: Getenvi("DB_REPLICA_SYSTEM_HOST", "NIL"),
		DB_REPLICA_SYSTEM_USER: Getenvi("DB_REPLICA_SYSTEM_USER", "NIL"),
		DB_REPLICA_SYSTEM_PASS: Getenvi("DB_REPLICA_SYSTEM_PASS", "NIL"),
		DB_REPLICA_SYSTEM_NAME: Getenvi("DB_REPLICA_SYSTEM_NAME", "NIL"),
		DB_REPLICA_SYSTEM_PORT: Getenvi("DB_REPLICA_SYSTEM_PORT", "NIL"),

		RDSHOST:      Getenvi("RDSHOST", "NIL"),
		RDSPORT:      Getenvi("RDSPORT", "NIL"),
		RDSAUTHDB:    rdsauth,
		RDSSESSIONDB: rdssession,
		MEILIHOST:    Getenvi("MEILIHOST", "NIL"),
		MEILIPORT:    Getenvi("MEILIPORT", "NIL"),
		MEILIKEY:     Getenvi("MEILIKEY", "NIL"),

		RMQ_HOST: Getenvi("RMQ_HOST", "NIL"),
		RMQ_USER: Getenvi("RMQ_USER", "NIL"),
		RMQ_PASS: Getenvi("RMQ_PASS", "NIL"),
		RMQ_PORT: Getenvi("RMQ_PORT", "NIL"),

		MINIO_ENDPOINT:              Getenvi("MINIO_ENDPOINT", "NIL"),
		MINIO_USE_SSL:               minioSSl,
		MINIO_ACCESS_KEY:            Getenvi("MINIO_ACCESS_KEY", "NIL"),
		MINIO_SECRET_KEY:            Getenvi("MINIO_SECRET_KEY", "NIL"),
		MINIO_SIGNED_URL_EXPIRE_SEC: Getenvi("MINIO_SIGNED_URL_EXPIRE_SEC", "NIL"),
	}

	// 2. Initialize Infrastructure Stack Connection
	db_system, redis_auth, redis_session, cud_publisher, media_storage := env.RunConnectionEnvironment()
	defer cud_publisher.Ch.Close()

	// ==========================================
	// PHASE: MIGRATIONS & PROVISIONING
	// ==========================================

	// Database Enums & Tables Migration
	if err := enums.UpEnumsEntity(db_system.Write); err != nil {
		log.Printf("❌ Gagal UpEnumsEntity: %v", err)
	}
	if err := enums.UpBarangEnums(db_system.Write); err != nil {
		log.Printf("❌ Gagal UpBarangEnums: %v", err)
	}
	if err := enums.UpEnumsTransaksi(db_system.Write); err != nil {
		log.Printf("❌ Gagal UpEnumsTransaksi: %v", err)
	}

	migrate.UpEntity(db_system.Write)
	migrate.UpBarang(db_system.Write)
	migrate.UpTransaksi(db_system.Write)
	migrate.UpEngagementEntity(db_system.Write)
	migrate.UpSystemData(db_system.Write)
	migrate.UpTresholdData(db_system.Write)
	migrate.UpMediaData(db_system.Write)
	migrate.UpThresholdTable(db_system.Write)

	// Message Broker Provisioning (Exchange & Queues)
	if err := mb_cud_exchange_provisioning.ProvisionExchangeCUD(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat exchange create update delete: ", err)
		return
	}
	if err := cud_publisher.QueueCreate.ProvisioningQueues(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat Queue Create: ", err)
		return
	}
	if err := cud_publisher.QueueUpdate.ProvisioningQueues(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat update queue: ", err)
		return
	}
	if err := cud_publisher.QueueDelete.ProvisioningQueues(cud_publisher.Ch); err != nil {
		fmt.Println("Gagal membuat queue Delete: ", err)
		return
	}

	// Media Storage Bucket Provisioning
	media_storage_database_seeders.BucketFotoName = Getenvi("MINIO_PHOTOS_BUCKET", "NIL")
	media_storage_database_seeders.BucketVideoName = Getenvi("MINIO_VIDEOS_BUCKET", "NIL")
	media_storage_database_seeders.BucketDokumenName = Getenvi("MINIO_DOKUMENS_BUCKET", "NIL")
	media_storage_database_migrate.MigrateBucketMediaStorage(media_storage)

	// ==========================================
	// PHASE: DATA CACHING & SEEDING
	// ==========================================

	// Maintain Caches
	maintain_cache.DataAlamatEkspedisiUp(db_system.Write)
	maintain_cache.DataOperasionalPengirimanUp()

	// Database Seeding (JSON Location Data)

	var KebijakanSistemdata models.KebijakanSistem
	if err := db_system.Read.Model(&models.KebijakanSistem{}).Where(&models.KebijakanSistem{
		StatusActive: true,
	}).Limit(1).Take(&KebijakanSistemdata).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("Error DB saat read KebjakanSistemData", err)
		return
	} else {
		byteValue, err := os.ReadFile("./operational_data/kebijakan_sistem.json")
		if err != nil {
			fmt.Println("Gagal baca file:", err)
			return
		}

		if err := json.Unmarshal(byteValue, &KebijakanSistemdata); err != nil {
			fmt.Println("Gagal unmarshal JSON:", err)
			return
		}

		if err := db_system.Write.Create(&KebijakanSistemdata).Error; err != nil {
			fmt.Println("gagal memasukan data kebijakan sistem ke dalam sistem:", err)
		}

	}
	var dump_ekspedisi models.AlamatEkspedisi
	err := db_system.Read.Model(&models.AlamatEkspedisi{}).Limit(1).Take(&dump_ekspedisi).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("Error DB saat read:", err)
		return
	}

	if dump_ekspedisi.ID == 0 {
		var dataAlamatEks []models.AlamatEkspedisi

		byteValue, err := os.ReadFile("../jne_location.json")
		if err != nil {
			fmt.Println("Gagal baca file:", err)
			return
		}

		if err := json.Unmarshal(byteValue, &dataAlamatEks); err != nil {
			fmt.Println("Gagal unmarshal JSON:", err)
			return
		}

		if err := db_system.Write.CreateInBatches(&dataAlamatEks, 2000).Error; err != nil {
			fmt.Println("Gagal insert batches:", err)
		}
	}

	// ==========================================
	// PHASE: ROUTER & HTTP SERVER HTTP
	// ==========================================

	Router := mux.NewRouter()
	Router.Use(enableCORS)
	// Router.Use(rateLimitMiddleware)
	// Router.Use(blockBadRequestsMiddleware)

	// Setup Routes
	Router.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	Router.PathPrefix("/").Handler(http.HandlerFunc(
		routes.PostHandler(db_system, redis_auth, redis_session, cud_publisher),
	)).Methods("POST")

	Router.PathPrefix("/").Handler(http.HandlerFunc(
		routes.PutHandler(db_system, media_storage, redis_session, cud_publisher),
	)).Methods("PUT")

	Router.PathPrefix("/").Handler(http.HandlerFunc(
		routes.PatchHandler(db_system, redis_auth, redis_session, cud_publisher),
	)).Methods("PATCH")

	Router.PathPrefix("/").Handler(http.HandlerFunc(
		routes.DeleteHandler(db_system, media_storage, redis_session, cud_publisher),
	)).Methods("DELETE")

	// Start Web Server
	port := Getenvi("APPPORT", "8080")
	fmt.Printf("🚀 Server Burung berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(port, Router); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
