package main

import (
	"fmt"
	"go-backend/factories"
	"log"
	"net/http"

	"github.com/gorilla/mux"
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
