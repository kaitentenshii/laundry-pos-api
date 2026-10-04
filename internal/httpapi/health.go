package httpapi

import "net/http"

type healthResponse struct {
	Status string `json:"status"`
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
