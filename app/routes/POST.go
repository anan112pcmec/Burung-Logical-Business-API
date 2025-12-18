package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
	admin_routes "github.com/anan112pcmec/Burung-backend-1/app/routes/admin"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/auth"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/callback"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/userroute"
)

func PostHandler(db *config.InternalDBReadWriteSystem, rds_auth, rds_session *redis.Client, mb_cud_publisher *mb_cud_publisher.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("PostHandler dijalankan...")

		if len(r.URL.Path) >= 6 && r.URL.Path[:6] == "/auth/" {
			fmt.Println("Auth HANDLER JALAN")
			auth.HandleAuth(db, w, r, rds_auth, rds_session)
			return
		}

		if len(r.URL.Path) >= 6 && r.URL.Path[:6] == "/user/" {
			userroute.PostUserHandler(db, w, r, rds_session, mb_cud_publisher)
			return
		}

		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/seller/" {
			seller.PostSellerHandler(db, w, r, rds_session, mb_cud_publisher)
			return
		}

		// Jika path diawali "/kurir/"
		if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/kurir/" {
			kurir.PostKurirHandler(db, w, r, rds_session, mb_cud_publisher)
			return
		}

		if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/admin/" {
			admin_routes.AdminPostHandler(w, r)
			return
		}

		if len(r.URL.Path) >= 10 && r.URL.Path[:10] == "/callback/" {
			callback.CallbackPostHandler(w, r, db)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "url mu tidak jelas",
		})
	}
}
