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
	storage.InitMinio()
	defer storage.DB.Close()

	router := mux.NewRouter()
	router.HandleFunc("/upload", handlers.UploadHandler).Methods("POST")
	router.HandleFunc("/get/{id}", handlers.GetHandler).Methods("GET")
	router.HandleFunc("/get", handlers.GetAllWorksHandler).Methods("GET")
	router.HandleFunc("/download/{id}", handlers.GetFileHandler).Methods("GET")
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down File Storing Service...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatal("Shutdown error:", err)
		}
		log.Println("Server gracefully stopped")
		os.Exit(0)
	}()

	log.Println("File Storing Service started on :8081")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}
