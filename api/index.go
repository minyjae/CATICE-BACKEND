package handler

import (
	"net/http"

	"github/minyjae/catice/internal/app"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	app.Handler(w, r)
}
