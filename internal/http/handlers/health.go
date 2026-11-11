package handlers

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type HealthHandler struct {
	client *mongo.Client
}

func NewHealthHandler(client *mongo.Client) *HealthHandler {
	return &HealthHandler{client: client}
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
	Mongo  string `json:"mongo" example:"ok"`
}

// Health godoc
// @Summary  Health check
// @Description Checks that the API is running and MongoDB answers a ping.
// @Tags     health
// @Produce  json
// @Success  200 {object} HealthResponse
// @Failure  503 {object} HealthResponse
// @Router   /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.client.Ping(ctx, nil); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "error", Mongo: "unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Mongo: "ok"})
}
