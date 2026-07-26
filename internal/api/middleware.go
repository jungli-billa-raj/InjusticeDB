package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
)

// AuthMiddleware inspects the JWT token and attaches user claims to context
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			Error(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			Error(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			Error(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			Error(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		userIDStr, _ := claims["user_id"].(string)
		role, _ := claims["role"].(string)

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			Error(w, http.StatusUnauthorized, "Invalid user ID in token")
			return
		}

		// Attach parsed claims to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UserRoleKey, role)

		// Pass down to actual handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
