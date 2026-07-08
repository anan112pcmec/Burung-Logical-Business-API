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
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/previsioning"
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

	rdsauth, _ := strconv.Atoi(Getenvi("RDSAUTH", "0"))
	rdssession, _ := strconv.Atoi(Getenvi("RDSESSION", "1"))
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

	previsioning.PrevisioningEnvironment(db_system, cud_publisher, media_storage)

	// ==========================================
	// PHASE: ROUTER & HTTP SERVER HTTP
	// ==========================================

	Router := mux.NewRouter()
	// Router.Use(enableCORS)
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
	fmt.Printf("ÃƒÂ°Ã…Â¸Ã…Â¡Ã¢â€šÂ¬ Server Burung berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(port, Router); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
