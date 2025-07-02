package server

import (
	"net/http"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

func HandleErr(w http.ResponseWriter, r *http.Request) {
	util.RespondWithError(w, 500, "Something went wrong")
}
