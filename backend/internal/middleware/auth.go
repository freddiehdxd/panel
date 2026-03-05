package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"panel-backend/internal/models"
)

type contextKey string

const UserContextKey contextKey = "user"

// JWTClaims represents the JWT payload
type JWTClaims struct {
	jwt.RegisteredClaims
}

// Auth middleware verifies JWT from cookie or Authorization header
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""

			// 1. Try HttpOnly cookie first
			if cookie, err := r.Cookie("panel_token"); err == nil {
				tokenStr = cookie.Value
			}

			// 2. Fall back to Authorization header
			if tokenStr == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					tokenStr = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if tokenStr == "" {
				writeJSON(w, http.StatusUnauthorized, models.ApiResponse{
					Success: false,
					Error:   "Missing token",
				})
				return
			}

			// Verify JWT
			token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, models.ApiResponse{
					Success: false,
					Error:   "Invalid or expired token",
				})
				return
			}

			claims, ok := token.Claims.(*JWTClaims)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, models.ApiResponse{
					Success: false,
					Error:   "Invalid token claims",
				})
				return
			}

			// Store username in context
			ctx := context.WithValue(r.Context(), UserContextKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUsername extracts username from request context
func GetUsername(r *http.Request) string {
	if username, ok := r.Context().Value(UserContextKey).(string); ok {
		return username
	}
	return "anonymous"
}

// ValidateToken checks if a JWT token string is valid (used by WebSocket auth)
func ValidateToken(tokenStr, jwtSecret string) bool {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	return err == nil && token.Valid
}
