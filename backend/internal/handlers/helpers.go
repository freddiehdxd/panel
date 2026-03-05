package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"panel-backend/internal/models"
)

// JSON sends a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Success sends a successful API response
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, models.ApiResponse{Success: true, Data: data})
}

// SuccessCreated sends a 201 successful API response
func SuccessCreated(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, models.ApiResponse{Success: true, Data: data})
}

// Error sends an error API response
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, models.ApiResponse{Success: false, Error: msg})
}

// ReadJSON reads and decodes JSON body into target
func ReadJSON(r *http.Request, target interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.Unmarshal(body, target)
}
