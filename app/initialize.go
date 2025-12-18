package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	routes "github.com/anan112pcmec/Burung-backend-1/app/Routes"
	maintain_cache "github.com/anan112pcmec/Burung-backend-1/app/cache/maintain"
	"github.com/anan112pcmec/Burung-backend-1/app/config"
	media_storage_database_migrate "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/migrate"
	media_storage_database_seeders "github.com/anan112pcmec/Burung-backend-1/app/database/media_storage_database/seeders"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/enums"
	"github.com/anan112pcmec/Burung-backend-1/app/database/sot_database/migrate"
	mb_cud_exchange_provisioning "github.com/anan112pcmec/Burung-backend-1/app/message_broker/provisioning/cud_exchange"
)

func Getenvi(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Run() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	rdsauth, _ := strconv.Atoi(Getenvi("RDSENTITY", "0"))
	rdssession, _ := strconv.Atoi(Getenvi("RDSAUTH", "0"))
	rdsengagement, _ := strconv.Atoi(Getenvi("RDSSESSION", "0"))
	minioSSl, _ := strconv.ParseBool(Getenvi("MINIO_USE_SSL", "NIL"))

	env := config.Environment{
		DB_MASTER_HOST:         Getenvi("DB_MASTER_HOST", "NIL"),
		DB_MASTER_USER:         Getenvi("DB_MASTER_USER", "NIL"),
		DB_MASTER_PASS:         Getenvi("DB_MASTER_PASS", "NIL"),
		DB_MASTER_NAME:         Getenvi("DB_MASTER_NAME", "NIL"),
		DB_MASTER_PORT:         Getenvi("DB_MASTER_PORT", "NIL"),
		DB_REPLICA_SYSTEM_HOST: Getenvi("DB_REPLICA_SYSTEM_HOST", "NIL"),
		DB_REPLICA_SYSTEM_USER: Getenvi("DB_REPLICA_SYSTEM_USER", "NIL"),
		DB_REPLICA_SYSTEM_PASS: Getenvi("DB_REPLICA_SYSTEM_PASS", "NIL"),
		DB_REPLICA_SYSTEM_NAME: Getenvi("DB_REPLICA_SYSTEM_NAME", "NIL"),
		DB_REPLICA_SYSTEM_PORT: Getenvi("DB_REPLICA_SYSTEM_PORT", "NIL"),

		DB_REPLICA_CLIENT_HOST: Getenvi("DB_REPLICA_CLIENT_HOST", "NIL"),
		DB_REPLICA_CLIENT_USER: Getenvi("DB_REPLICA_CLIENT_USER", "NIL"),
		DB_REPLICA_CLIENT_PASS: Getenvi("DB_REPLICA_CLIENT_PASS", "NIL"),
		DB_REPLICA_CLIENT_NAME: Getenvi("DB_REPLICA_CLIENT_NAME", "NIL"),
		DB_REPLICA_CLIENT_PORT: Getenvi("DB_REPLICA_CLIENT_PORT", "NIL"),

		RDSHOST:         Getenvi("RDSHOST", "NIL"),
		RDSPORT:         Getenvi("RDSPORT", "NIL"),
		RDSAUTHDB:       rdsauth,
		RDSSESSIONDB:    rdssession,
		RDSENGAGEMENTDB: rdsengagement,
		MEILIHOST:       Getenvi("MEILIHOST", "NIL"),
		MEILIPORT:       Getenvi("MEILIPORT", "NIL"),
		MEILIKEY:        Getenvi("MEILIKEY", "NIL"),

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

	db_system, db_replica_client, redis_auth, redis_session, _, searchengine, cud_publisher, media_storage :=
		env.RunConnectionEnvironment()

	// Router utama
	Router := mux.NewRouter()
	Router.Use(enableCORS)
	// Router.Use(rateLimitMiddleware)
	// Router.Use(blockBadRequestsMiddleware)

	// Jalankan enums dan migrasi
	// Migration SOT
	initSotDatabase := func() {
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
		//
	}
	initSotDatabase()

	initSotThresholdDatabase := func() {
		migrate.UpThresholdTable(db_system.Write)
	}
	initSotThresholdDatabase()

	// Message Broker

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

	//

	// Caching data
	maintain_cache.DataAlamatEkspedisiUp(db_system.Write)
	maintain_cache.DataOperasionalPengirimanUp()
	//

	// Media Storage Initializing
	media_storage_database_seeders.BucketFotoName = Getenvi("MINIO_PHOTOS_BUCKET", "NIL")
	media_storage_database_seeders.BucketVideoName = Getenvi("MINIO_VIDEOS_BUCKET", "NIL")
	media_storage_database_seeders.BucketDokumenName = Getenvi("MINIO_DOKUMENS_BUCKET", "NIL")
	media_storage_database_migrate.MigrateBucketMediaStorage(media_storage)
	//

	// Setup routes
	Router.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	Router.PathPrefix("/").Handler(http.HandlerFunc(
		routes.GetHandler(db_replica_client, redis_auth, redis_session, searchengine),
	)).Methods("GET")

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

	// go cleanupClients()

	// Jalankan web server
	port := Getenvi("APPPORT", "8080")
	fmt.Printf("🚀 Server Burung berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(port, Router); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}

	defer cud_publisher.Ch.Close()
}
