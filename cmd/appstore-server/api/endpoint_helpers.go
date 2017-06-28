package api

import (
	"net/http"

	"github.com/pressly/chi/render"
)

type ErrorJson struct {
	Error string `json:"error"`
}

func returnJSON(w http.ResponseWriter, r *http.Request, res interface{}, err error, status int) {
	render.Status(r, status)
	if err != nil {
		render.JSON(w, r, ErrorJson{err.Error()})
	} else {
		render.JSON(w, r, res)
	}
}
