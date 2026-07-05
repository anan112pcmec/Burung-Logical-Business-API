#!/bin/bash

# ====================================================
# 🚀 BACKEND STARTUP SCRIPT (Windows Git Bash Compatible)
# ====================================================

# Disable strict error exit for better control
set +e

# Colors for better readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MAX_WAIT_TIME=60
COMPOSE_FILE="docker-compose.yml"
HEALTH_CHECK_INTERVAL=2

# ====================================================
# FUNCTION: Print colored messages
# ====================================================
print_error() {
    echo -e "${RED}❌ ERROR: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# ====================================================
# FUNCTION: Check if command exists
# ====================================================
command_exists() {
    command -v "$1" >/dev/null 2>&1
    return $?
}

# ====================================================
# FUNCTION: Wait for container to be healthy
# ====================================================
wait_for_containers() {
    local max_wait=$1
    local elapsed=0
    
    print_info "Menunggu containers menjadi healthy..."
    
    # Check if jq is available
    if ! command_exists jq; then
        print_warning "jq tidak terinstall, skip health check"
        sleep 5
        return 0
    fi
    
    while [ $elapsed -lt $max_wait ]; do
        local unhealthy=$(docker compose ps --format json 2>/dev/null | jq -r 'select(.Health != "healthy" and .State == "running") | .Name' 2>/dev/null)
        
        if [ -z "$unhealthy" ]; then
            print_success "Semua containers sudah healthy!"
            return 0
        fi
        
        echo -ne "\r⏳ Menunggu... ${elapsed}s/${max_wait}s"
        sleep $HEALTH_CHECK_INTERVAL
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
    done
    
    echo ""
    print_warning "Timeout menunggu containers. Melanjutkan..."
    return 0
}

# ====================================================
# MAIN SCRIPT
# ====================================================

echo -e "${BLUE}"
echo "==================================="
echo "   🚀 BACKEND STARTUP SCRIPT"
echo "==================================="
echo -e "${NC}"

# 1. Check if Docker is installed
print_info "Memeriksa instalasi Docker..."
if ! command_exists docker; then
    print_error "Docker tidak terinstall!"
    print_info "Install Docker Desktop dari: https://www.docker.com/products/docker-desktop"
    exit 1
fi
print_success "Docker terinstall"

# 2. Check if Docker is running
print_info "Memeriksa Docker daemon..."
if ! docker info > /dev/null 2>&1; then
    print_error "Docker tidak berjalan! Nyalakan Docker Desktop terlebih dahulu."
    exit 1
fi
print_success "Docker daemon berjalan"

# 3. Check if Docker Compose is available
print_info "Memeriksa Docker Compose..."
if ! docker compose version >/dev/null 2>&1; then
    print_error "Docker Compose tidak tersedia!"
    print_info "Pastikan Docker Desktop versi terbaru sudah terinstall"
    exit 1
fi
print_success "Docker Compose tersedia"

# 4. Check if docker-compose.yml exists
print_info "Memeriksa file $COMPOSE_FILE..."
if [ ! -f "$COMPOSE_FILE" ]; then
    print_error "File $COMPOSE_FILE tidak ditemukan!"
    print_info "Pastikan Anda berada di direktori yang benar"
    exit 1
fi
print_success "File $COMPOSE_FILE ditemukan"

# 5. Check if Go is installed
print_info "Memeriksa instalasi Go..."
if ! command_exists go; then
    print_error "Go tidak terinstall!"
    print_info "Install Go dari: https://go.dev/dl/"
    exit 1
fi
GO_VERSION=$(go version 2>/dev/null)
print_success "Go terinstall: $GO_VERSION"

# 6. Check if main.go exists
print_info "Memeriksa file main.go..."
if [ ! -f "main.go" ]; then
    print_error "File main.go tidak ditemukan!"
    print_info "Pastikan Anda berada di direktori project yang benar"
    exit 1
fi
print_success "File main.go ditemukan"

# 7. Check if containers are already running
print_info "Memeriksa status containers..."
RUNNING_CONTAINERS=$(docker compose ps --services --filter "status=running" 2>/dev/null)
if [ -n "$RUNNING_CONTAINERS" ]; then
    print_warning "Containers sudah berjalan!"
    echo "$RUNNING_CONTAINERS"
    echo ""
    read -p "Restart containers? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Menghentikan containers lama..."
        docker compose down
        if [ $? -eq 0 ]; then
            print_success "Containers berhasil dihentikan"
        else
            print_error "Gagal menghentikan containers"
            exit 1
        fi
    else
        print_info "Menggunakan containers yang sudah berjalan"
    fi
fi

# 8. Start Docker Compose
echo ""
print_info "Menjalankan docker compose..."
docker compose up -d
if [ $? -ne 0 ]; then
    print_error "Gagal menjalankan docker compose!"
    print_info "Coba jalankan: docker compose logs"
    exit 1
fi
print_success "Docker compose berhasil dijalankan"

# 9. Wait for containers to be ready
echo ""
wait_for_containers $MAX_WAIT_TIME

# 10. Show container status
echo ""
print_info "Status containers:"
docker compose ps

# 11. Check for any failed containers
echo ""
print_info "Memeriksa containers yang gagal..."
FAILED_CONTAINERS=$(docker compose ps --format json 2>/dev/null | jq -r 'select(.State != "running") | .Name' 2>/dev/null)
if [ -n "$FAILED_CONTAINERS" ]; then
    print_warning "Beberapa containers gagal start:"
    echo "$FAILED_CONTAINERS"
    print_info "Periksa logs dengan: docker compose logs [container_name]"
else
    print_success "Semua containers berjalan dengan baik"
fi

print_info "Mengonfigurasi peran dan database PostgreSQL..."

# 1. Buat role/user 'burung_admin' dengan password 'burung_secure_pass'
print_info "Membuat user 'burung_admin' jika belum ada..."
docker exec -i postgres_burung_sot psql -U postgres -d postgres -c "
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'burung_admin') THEN
        CREATE ROLE burung_admin WITH LOGIN PASSWORD 'burung_secure_pass' SUPERUSER;
    END IF;
END
\$\$;" 2>/dev/null

if [ $? -eq 0 ]; then
    print_success "User 'burung_admin' berhasil dikonfigurasi"
    PG_ADMIN_USER="postgres"
else
    print_warning "User default 'postgres' tidak bisa login, asumsikan 'burung_admin' sudah aktif sebagai superuser"
    PG_ADMIN_USER="burung_admin"
fi

# 2. Periksa apakah database 'burung_sot_db' sudah ada, jika belum buat database-nya
print_info "Memeriksa database 'burung_sot_db'..."
DB_EXISTS=$(docker exec -i postgres_burung_sot psql -U "$PG_ADMIN_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='burung_sot_db'")

if [ "$DB_EXISTS" = "1" ]; then
    print_info "Database 'burung_sot_db' sudah ada, melewati pembuatan database."
else
    print_info "Database 'burung_sot_db' tidak ditemukan. Membuat database baru..."
    docker exec -i postgres_burung_sot psql -U "$PG_ADMIN_USER" -d postgres -c "CREATE DATABASE burung_sot_db OWNER burung_admin;"

    if [ $? -eq 0 ]; then
        print_success "Database 'burung_sot_db' berhasil dibuat dan hak milik diberikan ke 'burung_admin'"
    else
        print_error "Gagal membuat database 'burung_sot_db'"
        exit 1
    fi
fi

echo ""
read -p "🗑️  Reset total isi database 'burung_sot_db' (semua table, enum, function akan dihapus)? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Mereset schema 'public' di database 'burung_sot_db'..."
    docker exec -i postgres_burung_sot psql -U "$PG_ADMIN_USER" -d burung_sot_db -c "
        DROP SCHEMA public CASCADE;
        CREATE SCHEMA public AUTHORIZATION burung_admin;
        GRANT ALL ON SCHEMA public TO burung_admin;
    "
    if [ $? -eq 0 ]; then
        print_success "Database 'burung_sot_db' berhasil direset (schema 'public' bersih kembali)"
    else
        print_error "Gagal mereset database 'burung_sot_db'"
        exit 1
    fi
else
    print_info "Melewati reset database, melanjutkan proses berikutnya"
fi

print_info "Memeriksa user di dalam RabbitMQ..."

# 0. Tunggu sampai RabbitMQ app benar-benar ready (bukan cuma container running)
RABBITMQ_READY=0
for i in $(seq 1 15); do
    if docker exec -i rabbitmq_burung_message_broker rabbitmqctl await_startup --timeout 5 >/dev/null 2>&1; then
        RABBITMQ_READY=1
        break
    fi
    print_info "Menunggu RabbitMQ app siap... (percobaan $i/15)"
    sleep 2
done

if [ "$RABBITMQ_READY" -ne 1 ]; then
    print_warning "RabbitMQ app tidak kunjung siap, melewati konfigurasi user RabbitMQ"
else
    # 1. Cek apakah user 'burung_user' sudah ada di dalam daftar user RabbitMQ
    USER_EXISTS=$(docker exec -i rabbitmq_burung_message_broker rabbitmqctl list_users 2>/dev/null | awk '{print $1}' | grep -Fx "burung_user")

    if [ -n "$USER_EXISTS" ]; then
        print_info "User 'burung_user' sudah ada di RabbitMQ, melewati pembuatan user."
    else
        print_info "User 'burung_user' tidak ditemukan. Membuat user baru..."

        # 2. Buat user 'burung_user' dengan password 'burung_pass'
        ADD_USER_OUTPUT=$(docker exec -i rabbitmq_burung_message_broker rabbitmqctl add_user "burung_user" "burung_pass" 2>&1)

        if [ $? -eq 0 ] || echo "$ADD_USER_OUTPUT" | grep -qi "already exists"; then
            print_success "User 'burung_user' berhasil dibuat (atau sudah ada sebelumnya)"

            # 3. Set tag sebagai administrator agar user ini bisa masuk ke Management Web UI
            print_info "Mengatur role administrator untuk 'burung_user'..."
            docker exec -i rabbitmq_burung_message_broker rabbitmqctl set_user_tags "burung_user" administrator

            # 4. Berikan izin akses penuh (read, write, configure) di vhost default ("/")
            print_info "Mengatur izin akses (permissions) untuk 'burung_user'..."
            docker exec -i rabbitmq_burung_message_broker rabbitmqctl set_permissions -p "/" "burung_user" ".*" ".*" ".*"

            print_success "Konfigurasi penuh 'burung_user' selesai dilakukan"
        else
            print_warning "Gagal membuat user 'burung_user' di RabbitMQ, melewati step ini"
            echo "$ADD_USER_OUTPUT"
        fi
    fi
fi

echo ""
read -p "🗑️  Hapus semua queue di RabbitMQ? Exchange, vhost, dan routing TIDAK akan disentuh (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ "$RABBITMQ_READY" -ne 1 ]; then
        print_warning "RabbitMQ app tidak ready, melewati penghapusan queue"
    else
        print_info "Menghapus semua queue di vhost '/'..."
        docker exec -i rabbitmq_burung_message_broker rabbitmqctl eval '
            lists:foreach(
                fun(Q) -> {ok, _} = rabbit_amqqueue:delete(Q, false, false, <<"admin">>) end,
                rabbit_amqqueue:list(<<"/">>)
            ).
        '
        if [ $? -eq 0 ]; then
            print_success "Semua queue berhasil dihapus (exchange & routing tetap utuh)"
        else
            print_error "Gagal menghapus queue di RabbitMQ"
            exit 1
        fi
    fi
else
    print_info "Melewati penghapusan queue RabbitMQ, melanjutkan proses berikutnya"
fi

print_info "Memeriksa kredensial root di dalam container MinIO..."

# 1. Ambil nilai env MINIO_ROOT_USER dan MINIO_ROOT_PASSWORD langsung dari dalam kontainer
CURRENT_MINIO_USER=$(docker exec -i minio_burung_storage printenv MINIO_ROOT_USER 2>/dev/null | tr -d '\r')
CURRENT_MINIO_PASS=$(docker exec -i minio_burung_storage printenv MINIO_ROOT_PASSWORD 2>/dev/null | tr -d '\r')

# 2. Validasi apakah nilainya sudah sesuai dengan 'burung_app' dan 'burung_app123'
if [ "$CURRENT_MINIO_USER" = "burung_app" ] && [ "$CURRENT_MINIO_PASS" = "burung_app123" ]; then
    print_success "Kredensial MinIO sudah sesuai (User: burung_app)"
else
    print_warning "Kredensial MinIO tidak sesuai atau belum terkonfigurasi di dalam kontainer!"
    print_info "Kredensial saat ini -> User: '${CURRENT_MINIO_USER}', Pass: '${CURRENT_MINIO_PASS}'"
    
    # Kredensial root MinIO tidak bisa diubah secara runtime dari dalam psql/cli tanpa restart kontainer.
    # Satu-satunya cara aman "membuat/mengubahnya" adalah memastikan env ini terpasang di docker-compose.
    print_error "Silakan sesuaikan environment MINIO_ROOT_USER dan MINIO_ROOT_PASSWORD pada file docker-compose.yml Anda, lalu jalankan ulang."
    exit 1
fi

echo ""
read -p "🗑️  Hapus semua object di semua bucket MinIO? Bucket tetap ada, isinya saja dikosongkan (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Menghapus semua object di semua bucket MinIO..."

    MINIO_NETWORK=$(docker inspect minio_burung_storage --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}' | head -n1)

    docker run --rm --network "$MINIO_NETWORK" minio/mc sh -c "
        mc alias set local http://minio_burung_storage:9000 '$CURRENT_MINIO_USER' '$CURRENT_MINIO_PASS' >/dev/null &&
        for b in \$(mc ls local | awk '{print \$NF}' | tr -d '/'); do
            echo '  -> Mengosongkan bucket:' \$b
            mc rm --recursive --force local/\$b >/dev/null
        done
    "

    if [ $? -eq 0 ]; then
        print_success "Semua object di seluruh bucket MinIO berhasil dihapus"
    else
        print_error "Gagal menghapus object di MinIO"
        exit 1
    fi
else
    print_info "Melewati reset MinIO, melanjutkan proses berikutnya"
fi

# 13. Run backend Go application
echo ""
print_success "Semua checks passed! Starting backend..."
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Disable auto-restart for cleaner exit
print_info "Menjalankan backend Go application..."
echo ""

go run main.go
EXIT_CODE=$?

# 14. Exit message
echo ""
if [ $EXIT_CODE -eq 0 ]; then
    print_success "Backend berhenti dengan normal"
else
    print_error "Backend berhenti dengan exit code: $EXIT_CODE"
fi

print_info "Docker containers masih berjalan"
print_info "Gunakan 'docker compose down' untuk menghentikan containers"
print_info "Gunakan 'docker compose logs' untuk melihat logs"

exit $EXIT_CODE