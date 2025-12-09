package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/anan112pcmec/Burung-backend-1/app/config"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/kurir"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/seller"
	"github.com/anan112pcmec/Burung-backend-1/app/routes/userroute"
)

func PutHandler(db *config.InternalDBReadWriteSystem, ms *minio.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("PutHandler dijalankan...")

		// Jika path diawali "/user/"
		if len(r.URL.Path) >= 6 && r.URL.Path[:6] == "/user/" {
			userroute.PutUserHandler(db, w, r, ms)
			return
		}

		// Jika path diawali "/seller/"
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/seller/" {
			seller.PutSellerHandler(db, w, r, ms)
			return
		}

		// Jika path diawali "/kurir/"
		if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/kurir/" {
			kurir.PutKurirHandler(db, w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "url mu tidak jelas",
		})
	}
}
