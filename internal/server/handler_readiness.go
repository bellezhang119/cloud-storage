package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

func HandlerReadiness(w http.ResponseWriter, r *http.Request) {
	util.RespondWithJSON(w, 200, "The API is ready")
}
