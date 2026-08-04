package handler

import (
	"net/http"

	"github/minyjae/catice/pkg/app"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	app.Handler(w, r)
}
