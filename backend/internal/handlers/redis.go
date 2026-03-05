package handlers

import (
	"net/http"

	"panel-backend/internal/models"
	"panel-backend/internal/services"
)

// RedisHandler handles Redis status and management routes
type RedisHandler struct {
	exec *services.Executor
}

// NewRedisHandler creates a new Redis handler
func NewRedisHandler(exec *services.Executor) *RedisHandler {
	return &RedisHandler{exec: exec}
}

// Status handles GET /api/redis
func (h *RedisHandler) Status(w http.ResponseWriter, r *http.Request) {
	result, err := h.exec.RunBin("systemctl", "is-active", "--quiet", "redis-server")
	if err != nil {
		// Not installed or error
		Success(w, models.RedisInfo{
			Installed:  false,
			Running:    false,
			Connection: nil,
		})
		return
	}

	running := result.Code == 0

	if running {
		Success(w, models.RedisInfo{
			Installed: true,
			Running:   true,
			Connection: models.RedisConnection{
				Host:   "127.0.0.1",
				Port:   6379,
				URL:    "redis://127.0.0.1:6379",
				EnvVar: "REDIS_URL=redis://127.0.0.1:6379",
			},
		})
	} else {
		Success(w, models.RedisInfo{
			Installed:  false,
			Running:    false,
			Connection: nil,
		})
	}
}

// Install handles POST /api/redis/install
func (h *RedisHandler) Install(w http.ResponseWriter, r *http.Request) {
	result, err := h.exec.RunScript("install_redis.sh")
	if err != nil {
		Error(w, http.StatusInternalServerError, "Install failed")
		return
	}

	if result.Code != 0 {
		msg := result.Stderr
		if msg == "" {
			msg = "Install failed"
		}
		Error(w, http.StatusInternalServerError, msg)
		return
	}

	Success(w, map[string]string{"message": "Redis installed and started"})
}
