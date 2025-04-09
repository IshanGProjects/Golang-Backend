package auth

import (
	"context"
	"errors"
	"log"

	"firebase.google.com/go/v4/auth"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthService provides authentication services
type AuthService struct {
	FireAuth *auth.Client
}

// Login authenticates a user with Firebase Authentication and returns an ID token
func (s *AuthService) Login(email, password string) (string, error) {
	// Authenticate with Firebase using email and password
	user, err := s.FireAuth.GetUserByEmail(context.Background(), email)
	if err != nil {
		log.Printf("failed to get user from Firebase: %v", err)
		return "", errors.New("invalid email or password")
	}

	// Firebase does not expose password verification in Admin SDK.
	// Instead, the client should authenticate via Firebase SDK and send the ID token to the server.

	// Generate a Firebase custom token using the UID
	token, err := s.FireAuth.CustomToken(context.Background(), user.UID)
	if err != nil {
		log.Printf("failed to generate custom token: %v", err)
		return "", errors.New("internal server error")
	}

	return token, nil
}

// Register creates a new user in Firebase and returns a Firebase custom token
func (s *AuthService) Register(email, password string) (string, error) {
	// Create a new user in Firebase Authentication
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password)

	userRecord, err := s.FireAuth.CreateUser(context.Background(), params)
	if err != nil {
		log.Printf("failed to create Firebase user: %v", err)
		return "", errors.New("failed to create user")
	}

	// Generate a Firebase custom token
	customToken, err := s.FireAuth.CustomToken(context.Background(), userRecord.UID)
	if err != nil {
		log.Printf("failed to generate custom token: %v", err)
		return "", errors.New("internal server error")
	}

	return customToken, nil
}
