package handler

import (
	"net/http"
	"search-gin/internal/sse"
)

func HandleSSE(w http.ResponseWriter, r *http.Request) {
	sse.HandleSSE(w, r)
}
