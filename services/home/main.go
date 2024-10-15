package main

import (
	"context"
	"net/http"

	"github.com/terenceodonoghue/go/services/home/internal/controller"
	"github.com/terenceodonoghue/go/services/home/internal/middleware"
)

func main() {
	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", controller.GetStatus(ctx))

	http.ListenAndServe(":8080", middleware.CORS(mux))
}
