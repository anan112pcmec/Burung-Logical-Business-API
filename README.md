# Burung Logical Business API App

Backend service untuk operasi bisnis inti — menangani write, read, penyimpanan file, dan event publishing dalam satu alur yang bersih.

## Stack

![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)
![MinIO](https://img.shields.io/badge/MinIO-C72E49?logo=minio&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?logo=rabbitmq&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-FF4438?logo=redis&logoColor=white)

## Architecture

![Architecture Diagram](https://github.com/user-attachments/assets/0f2d3565-e6a5-48b2-a37c-15f7e62a84c8)

| Layer | Komponen | Peran |
|---|---|---|
| Transport | HTTP | POST · PUT · PATCH · DELETE |
| Backend | burung-logical-business-app | Business logic, auth, routing |
| Auth Cache | Redis | Session token & user data |
| Write | PostgreSQL Master | CUD operations |
| Read | PostgreSQL Sync Replica | Query & metrics |
| Storage | MinIO | Dokumen, foto, video, objek |
| Messaging | RabbitMQ | Publish event setelah operasi selesai |

## Flow

```
POST / PUT / DELETE / PATCH
        │
        ▼
burung-logical-business-app
        │
        ├── Auth? ──────────────► Redis (cache session & user data)
        │
        ├── CUD ────────────────► PostgreSQL Master
        │        ◄── sync ──────► PostgreSQL Sync Replica (read / metrics)
        │
        ├── File / Media ───────► MinIO (dokumen, foto, video, dll)
        │
        └── Publish event ──────► RabbitMQ
                                        │
                                        ▼
                                     Selesai ✓
```

## Getting Started

```bash
# Clone repo
git clone https://github.com/<your-org>/burung-logical-business-app.git
cd burung-logical-business-app

# Copy env
cp .env.example .env

# Run
go run ./cmd/main.go
```

## Configuration

### RabbitMQ

Tambahkan di `rabbitmq.conf`:

```properties
management.load_definitions = /etc/rabbitmq/definitions_vhost_rmq.json
```

File `definitions_vhost_rmq.json` mendefinisikan vhost, user, exchange, dan queue secara deklaratif saat RabbitMQ pertama kali startup.

---

### Environment Variables

Salin `.env.example` ke `.env` lalu sesuaikan nilainya:

```bash
cp .env.example .env
```

```dotenv
# App
APPNAME=Backend
APPENV=dev                  # dev | staging | production
APPPORT=:8080

# PostgreSQL Master (write)
DB_MASTER_HOST=localhost
DB_MASTER_USER=postgres
DB_MASTER_PORT=5432
DB_MASTER_PASS=your_password
DB_MASTER_NAME=your_db_name

# PostgreSQL Sync Replica (read)
DB_REPLICA_SYSTEM_HOST=localhost
DB_REPLICA_SYSTEM_USER=postgres
DB_REPLICA_SYSTEM_PORT=5432
DB_REPLICA_SYSTEM_PASS=your_password
DB_REPLICA_SYSTEM_NAME=your_db_name

# Redis
RDSHOST=localhost
RDSPORT=6379
RDSAUTH=1                   # Redis DB index untuk auth
RDSSESSION=2                # Redis DB index untuk session

# Meilisearch
MEILIHOST=localhost
MEILIPORT=7700
MEILIKEY=your_meili_master_key

# MinIO
MINIO_ENDPOINT=127.0.0.1:9000
MINIO_USE_SSL=false
MINIO_ACCESS_KEY=your_access_key
MINIO_SECRET_KEY=your_secret_key
MINIO_SIGNED_URL_EXPIRE_SEC=300
MINIO_PHOTOS_BUCKET=burung-foto
MINIO_VIDEOS_BUCKET=burung-video
MINIO_DOKUMENS_BUCKET=burung-dokumen

# CORS & Rate Limiting
ACCESS_CTRL=http://localhost:5173
REQ_LIMIT=10
BURST_LIMIT=100

# RabbitMQ
RMQ_HOST=localhost
RMQ_USER=your_rmq_user
RMQ_PASS=your_rmq_pass
RMQ_PORT=5672
EXCHANGE=md.cud

# SMTP (Gmail)
CONFIG_SMTP_HOST=smtp.gmail.com
CONFIG_SMTP_PORT=587
CONFIG_SENDER_NAME=App Name <your_email@gmail.com>
CONFIG_AUTH_EMAIL=your_email@gmail.com
CONFIG_AUTH_PASSWORD=your_gmail_app_password   # Google App Password, bukan password utama

# Third-party API Keys
OPEN_ROUTE_KEY=your_openroute_api_key
RAJA_ONG_COST_KEY=your_rajaongkir_cost_key
RAJA_ONG_DELIVERY_KEY=your_rajaongkir_delivery_key
```

> **Jangan commit file `.env` ke repository.** Pastikan `.env` sudah ada di `.gitignore`.

```gitignore
.env
```
