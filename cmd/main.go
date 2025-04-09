package main

import (
	"context"
	"fmt"
	"go-backend/auth"
	"go-backend/endpoints"
	"go-backend/factories"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		next.ServeHTTP(w, r)
	})
}

func main() {
	router := mux.NewRouter()
	router.Use(commonMiddleware)

	// Create a new service director
	serviceDirector := factories.NewServiceDirector()

	// Create Firebase instance
	opt := option.WithCredentialsFile("./escapia-login-firebase-adminsdk-fbsvc-aa851b3e38.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Failed to create Firebase app: %v", err)
	}

	// Create Firebase auth client
	authClient, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("Failed to create Firebase auth client: %v", err)
	}

	//Setup auth and middleware
	authService := &auth.AuthService{
		//can connect to database here,
		FireAuth: authClient,
	}
	authController := endpoints.NewAuthController(authService)

	// Firebase login Routes
	router.HandleFunc("/login", authController.LoginHandler).Methods("POST")
	router.HandleFunc("/register", authController.RegisterHandler).Methods("POST")

	// Start a simple server to verify the server is running
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Server check verified")
	}).Methods("GET")

	//Process Prompt
	router.HandleFunc("/promptOpenAI", func(w http.ResponseWriter, r *http.Request) {
		serviceDirector.ProcessPrompt(w, r)
	}).Methods("POST")

	port := "8000"
	log.Println("Server listening on port", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("Error starting server:", err)
	}

}
