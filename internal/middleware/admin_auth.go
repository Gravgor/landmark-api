package middleware

import (
	"fmt"
	"landmark-api/internal/services"
	"net/http"
)

func AdminMiddleware(authService services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractTokenFromHeader(r)
			if tokenString == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			user, subscription, err := authService.VerifyTokenAdmin(tokenString)
			if err != nil {
				fmt.Print(err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := services.WithUserAndSubscriptionContext(r.Context(), user, subscription)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
