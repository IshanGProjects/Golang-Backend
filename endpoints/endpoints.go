package endpoints

import (
	"encoding/json"
	"go-backend/auth"
	"net/http"
)

// AuthController is the controller for handling authentication requests
type AuthController struct {
	authService *auth.AuthService
}

// NewAuthController creates a new instance of the AuthController struct
func NewAuthController(authService *auth.AuthService) *AuthController {
	return &AuthController{authService}
}

// Login handles the POST /login route and login a new user with the provided credentials
func (c *AuthController) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if loginData.Email == "" || loginData.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email and password are required"})
		return
	}

	customToken, err := c.authService.Login(loginData.Email, loginData.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": customToken})
}

func (c *AuthController) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var registrationData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Parse the JSON body
	if err := json.NewDecoder(r.Body).Decode(&registrationData); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Basic validation
	if registrationData.Email == "" || registrationData.Password == "" {
		http.Error(w, `{"error": "Email and password are required"}`, http.StatusBadRequest)
		return
	}

	// You need access to your authService here — assume it's available in the outer scope
	customToken, err := c.authService.Register(registrationData.Email, registrationData.Password)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Return token as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": customToken})
}
