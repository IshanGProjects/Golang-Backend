package main

import (
	"fmt"
	"go-backend/factories"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	// Adjust this path as per your GOPATH or module path
)

func main() {
	router := mux.NewRouter()
	router.Use(commonMiddleware)

	// Instantiate ServiceDirector from the factories package
	serviceDirector := factories.NewServiceDirector()

	// Start the server, ensuring that the port is properly typed
	port := os.Getenv("EXPRESS_PORT")
	if port == "" {
		port = "8000" // Default port
	}

	address := ":" + port
	log.Printf("Starting the server on %s", address)

	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("ListenAndServe error: ", err)
	}

	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Server check verified")
	}).Methods("GET")

	//router.HandleFunc("/generate-embedding", embeddingPromptHandler).Methods("POST")

	router.HandleFunc("/promptOpenAI", func(w http.ResponseWriter, r *http.Request) {
		serviceDirector.ProcessPrompt(w, r)
	}).Methods("POST")

	//router.HandleFunc("/qdrant-test", checkQdrantStatus).Methods("GET")

	log.Println("App server now listening on port", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Stub functions for handlers to be implemented
func embeddingPromptHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Request received for embedding")
}

func checkQdrantStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Qdrant status checked")
}
