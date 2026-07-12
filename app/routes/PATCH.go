package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/userroute"
	pengguna_barang_services "github.com/anan112pcmec/Burung-backend-1/app/service/pengguna_service/barang_services"
)

func PatchHandler(db *environment.InternalDBReadWriteSystem, rds_auth *redis.Client, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher, b *pengguna_barang_services.BatchViewUpdate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("PatchHandler dijalankan...")

		if len(r.URL.Path) >= 6 && r.URL.Path[:6] == "/user/" {
			userroute.PatchUserHandler(db, w, r, rds_auth, rds_session, mb_cud_publisher, b)
			return
		}

		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/seller/" {
			seller.PatchSellerHandler(db, w, r, rds_auth, rds_session, mb_cud_publisher)
			return
		}

		if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/kurir/" {
			kurir.PatchKurirHandler(db, w, r, rds_auth, rds_session, mb_cud_publisher)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "url mu tidak jelas",
		})
	}
}
