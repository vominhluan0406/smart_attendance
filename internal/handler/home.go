package handler

import (
	"log"
	"net/http"

	"github.com/smart-attendance/smart-attendance/internal/renderer"
)

type HomeHandler struct {
	render *renderer.Renderer
}

func NewHomeHandler(render *renderer.Renderer) *HomeHandler {
	return &HomeHandler{render: render}
}

func (h *HomeHandler) Index(w http.ResponseWriter, r *http.Request) {
	if err := h.render.Render(w, "home.html", nil); err != nil {
		log.Printf("[handler][home] render error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *HomeHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
