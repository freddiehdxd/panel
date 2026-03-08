package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"panel-backend/internal/services"
)

const (
	defaultLogLines = 200
	maxLogLines     = 1000
)

// LogsHandler handles log viewing routes
type LogsHandler struct {
	pm2  *services.PM2
	exec *services.Executor
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(pm2 *services.PM2, exec *services.Executor) *LogsHandler {
	return &LogsHandler{pm2: pm2, exec: exec}
}

// AppLogs handles GET /api/logs/app/:name
func (h *LogsHandler) AppLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if !services.ValidateAppName(name) {
		Error(w, http.StatusBadRequest, "Invalid app name")
		return
	}

	lines := parseLines(r)

	log, err := h.pm2.Logs(name, lines)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	Success(w, map[string]string{"log": log})
}

// AppLogFile handles GET /api/logs/app/:name/file?type=out|error&lines=N
// Reads PM2 log files directly from disk (no subprocess) for lightweight polling.
func (h *LogsHandler) AppLogFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if !services.ValidateAppName(name) {
		Error(w, http.StatusBadRequest, "Invalid app name")
		return
	}

	logType := r.URL.Query().Get("type")
	if logType != "error" {
		logType = "out"
	}

	lines := parseLines(r)

	// PM2 stores logs in ~/.pm2/logs/{name}-{out|error}.log
	pm2Home := os.Getenv("HOME")
	if pm2Home == "" {
		pm2Home = "/root"
	}
	logFile := filepath.Join(pm2Home, ".pm2", "logs", fmt.Sprintf("%s-%s.log", name, logType))

	// Check file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		Success(w, map[string]string{
			"log":  "",
			"type": logType,
			"file": logFile,
		})
		return
	}

	result, err := h.exec.RunBin("tail", "-n", fmt.Sprintf("%d", lines), logFile)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to read log file")
		return
	}

	Success(w, map[string]interface{}{
		"log":  strings.TrimSpace(result.Stdout),
		"type": logType,
		"file": logFile,
	})
}

// NginxLogs handles GET /api/logs/nginx
func (h *LogsHandler) NginxLogs(w http.ResponseWriter, r *http.Request) {
	logType := r.URL.Query().Get("type")
	if logType == "" {
		logType = "access"
	}

	var logFile string
	switch logType {
	case "access":
		logFile = "/var/log/nginx/access.log"
	case "error":
		logFile = "/var/log/nginx/error.log"
	default:
		logFile = "/var/log/nginx/access.log"
		logType = "access"
	}

	lines := parseLines(r)

	result, err := h.exec.RunBin("tail", "-n", fmt.Sprintf("%d", lines), logFile)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to read logs")
		return
	}

	Success(w, map[string]string{
		"log":  result.Stdout,
		"type": logType,
	})
}

// parseLines parses the ?lines= query parameter
func parseLines(r *http.Request) int {
	linesStr := r.URL.Query().Get("lines")
	if linesStr == "" {
		return defaultLogLines
	}

	n, err := strconv.Atoi(linesStr)
	if err != nil || n <= 0 {
		return defaultLogLines
	}
	if n > maxLogLines {
		return maxLogLines
	}
	return n
}
