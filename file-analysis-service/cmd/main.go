package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/tupitsin88/antiplagiat/file-analysis-service/internal/handlers"
	"github.com/tupitsin88/antiplagiat/file-analysis-service/internal/storage"
)

func main() {
	storage.InitDB()
	defer storage.DB.Close()

	router := mux.NewRouter()
	router.HandleFunc("/check/{work_id}", handlers.CheckHandler).Methods("POST")
	router.HandleFunc("/get-report/{id}", handlers.GetReportHandler).Methods("GET")
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	server := &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatal("Shutdown error:", err)
		}
		log.Println("Server gracefully stopped")
	}()

	log.Println("File Analysis Service started on :8082")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Server error:", err)
	}
}
