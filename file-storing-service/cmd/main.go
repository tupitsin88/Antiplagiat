package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/tupitsin88/antiplagiat/file-storing-service/internal/handlers"
	"github.com/tupitsin88/antiplagiat/file-storing-service/internal/storage"
)

func main() {
	storage.InitDB()
	defer storage.DB.Close()

	router := mux.NewRouter()
	router.HandleFunc("/upload", handlers.UploadHandler).Methods("POST")
	router.HandleFunc("/get/{id}", handlers.GetHandler).Methods("GET")
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Server gracefully stopped")
		os.Exit(0)
	}()

	log.Println("File Storing Service started on :8081")
	if err := http.ListenAndServe(":8081", router); err != nil {
		log.Fatal("Server error:", err)
	}
}
