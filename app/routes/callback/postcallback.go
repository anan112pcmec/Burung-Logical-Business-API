package callback

import (
	"net/http"

	callback_payment_out "github.com/anan112pcmec/Burung-backend-1/app/callback/payment_out"
	"github.com/anan112pcmec/Burung-backend-1/app/environment"
	"github.com/anan112pcmec/Burung-backend-1/app/helper"
	mb_cud_publisher "github.com/anan112pcmec/Burung-backend-1/app/message_broker/publisher/cud_exchange"
)

func CallbackPostHandler(w http.ResponseWriter, r *http.Request, db *environment.InternalDBReadWriteSystem, cud_publisher *mb_cud_publisher.Publisher) {
	var status int16

	ctx := r.Context()
	switch r.URL.Path {
	case "/update_status_disbursment":
		var data callback_payment_out.PayloadUpdateStatusPaymentOut
		if err := helper.DecodeJSONBody(r, &data); err != nil {
			http.Error(w, "Gagal parsing JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		status = callback_payment_out.UpdateStatusPaymentOut(ctx, data, db, cud_publisher)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(status))
}
