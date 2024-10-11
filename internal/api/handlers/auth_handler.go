package handlers

import (
	"encoding/json"
	"landmark-api/internal/services"
	"net/http"
)

// AuthHandler handles authentication-related requests
// @Description Handles user registration, login, and token verification
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates a new AuthHandler
// @Description Creates a new AuthHandler with the given AuthService
// @Param authService services.AuthService
// @Return *AuthHandler
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// registrationRequest represents the structure of a registration request
type registrationRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registrationResponse struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	Error string `json:"error,omitempty"`
}

// loginRequest represents the structure of a login request
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse represents the structure of an authentication response
type authResponse struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

type validateResponse struct {
	Validate string `json:"validate"`
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with the provided email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param registration body registrationRequest true "Registration details"
// @Success 200 {object} authResponse
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.authService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := registrationResponse{}
	resp.User.ID = user.ID.String()
	resp.User.Email = user.Email

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Login godoc
// @Summary Authenticate a user
// @Description Authenticate a user with the provided email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param login body loginRequest true "Login details"
// @Success 200 {object} authResponse
// @Failure 400 {string} string "Invalid request body"
// @Failure 401 {string} string "Unauthorized"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	resp := authResponse{Token: token}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	resp := validateResponse{Validate: "Token valid"}
	json.NewEncoder(w).Encode(resp)
}

// AuthMiddleware verifies the JWT token
// @Description Middleware to verify JWT token and add user and subscription to context
// @Param next http.HandlerFunc
// @Return http.HandlerFunc
func (h *AuthHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		user, subscription, err := h.authService.VerifyToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := services.WithUserAndSubscriptionContext(r.Context(), user, subscription)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
