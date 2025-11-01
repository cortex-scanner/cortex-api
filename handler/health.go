package handler

import "net/http"

func HandleHealth(w http.ResponseWriter, r *http.Request) error {
	return RespondOne(w, r, "OK")
}
