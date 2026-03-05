package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"panel-backend/internal/config"
	"panel-backend/internal/services"
)

// DatabasesHandler handles managed database routes
type DatabasesHandler struct {
	db  *services.DB
	cfg *config.Config
}

// NewDatabasesHandler creates a new databases handler
func NewDatabasesHandler(db *services.DB, cfg *config.Config) *DatabasesHandler {
	return &DatabasesHandler{db: db, cfg: cfg}
}

// List handles GET /api/databases
func (h *DatabasesHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx,
		"SELECT id, name, db_user, created_at FROM managed_databases ORDER BY created_at DESC")
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to fetch databases")
		return
	}
	defer rows.Close()

	type dbEntry struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		DBUser    string    `json:"db_user"`
		CreatedAt time.Time `json:"created_at"`
	}

	databases := make([]dbEntry, 0)
	for rows.Next() {
		var d dbEntry
		if err := rows.Scan(&d.ID, &d.Name, &d.DBUser, &d.CreatedAt); err != nil {
			Error(w, http.StatusInternalServerError, "Failed to scan database")
			return
		}
		databases = append(databases, d)
	}

	Success(w, databases)
}

// Create handles POST /api/databases
func (h *DatabasesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		User string `json:"user"`
	}
	if err := ReadJSON(r, &body); err != nil {
		Error(w, http.StatusBadRequest, "name and user required")
		return
	}

	if body.Name == "" || body.User == "" {
		Error(w, http.StatusBadRequest, "name and user required")
		return
	}

	if !services.ValidatePgIdentifier(body.Name) || !services.ValidatePgIdentifier(body.User) {
		Error(w, http.StatusBadRequest, "Invalid name or user. Use lowercase letters, numbers and underscores.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Check for duplicates
	var exists bool
	err := h.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM managed_databases WHERE name = $1 OR db_user = $2)",
		body.Name, body.User).Scan(&exists)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Database error")
		return
	}
	if exists {
		Error(w, http.StatusConflict, "Database or user already exists")
		return
	}

	// Generate password
	passwordBytes := make([]byte, 16)
	if _, err := rand.Read(passwordBytes); err != nil {
		Error(w, http.StatusInternalServerError, "Failed to generate password")
		return
	}
	password := hex.EncodeToString(passwordBytes)

	// Create PostgreSQL user with temp password, then alter to real password
	// This avoids SQL injection since CREATE USER doesn't support parameterized passwords
	_, err = h.db.Exec(ctx,
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD 'temp_password'`, body.User))
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create user: %v", err))
		return
	}

	// Set real password via ALTER ROLE with parameterized query
	_, err = h.db.Exec(ctx,
		fmt.Sprintf(`ALTER ROLE "%s" WITH PASSWORD $1`, body.User), password)
	if err != nil {
		// Rollback: drop user
		h.db.Exec(ctx, fmt.Sprintf(`DROP USER IF EXISTS "%s"`, body.User))
		Error(w, http.StatusInternalServerError, fmt.Sprintf("Failed to set password: %v", err))
		return
	}

	// Create database (must be outside a transaction in PostgreSQL)
	// We need to use a separate connection for this
	_, err = h.db.Exec(ctx,
		fmt.Sprintf(`CREATE DATABASE "%s" OWNER "%s"`, body.Name, body.User))
	if err != nil {
		// Rollback: drop user
		h.db.Exec(ctx, fmt.Sprintf(`DROP USER IF EXISTS "%s"`, body.User))
		Error(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create database: %v", err))
		return
	}

	// Store in managed_databases
	_, err = h.db.Exec(ctx,
		"INSERT INTO managed_databases (name, db_user, password) VALUES ($1, $2, $3)",
		body.Name, body.User, password)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to save database record")
		return
	}

	connStr := fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", body.User, password, h.cfg.DBHost, body.Name)

	SuccessCreated(w, map[string]string{
		"name":              body.Name,
		"db_user":           body.User,
		"connection_string": connStr,
	})
}

// Delete handles DELETE /api/databases/:name
func (h *DatabasesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if !services.ValidatePgIdentifier(name) {
		Error(w, http.StatusBadRequest, "Invalid database name")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Look up the database
	var dbUser string
	err := h.db.QueryRow(ctx,
		"SELECT db_user FROM managed_databases WHERE name = $1", name).Scan(&dbUser)
	if err != nil {
		Error(w, http.StatusNotFound, "Database not found")
		return
	}

	// Terminate active connections
	h.db.Exec(ctx,
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", name)

	// Drop database
	h.db.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, name))

	// Drop user (best effort)
	h.db.Exec(ctx, fmt.Sprintf(`DROP USER IF EXISTS "%s"`, dbUser))

	// Remove from managed_databases
	h.db.Exec(ctx, "DELETE FROM managed_databases WHERE name = $1", name)

	Success(w, map[string]string{"message": "Database " + name + " deleted"})
}
