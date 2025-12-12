package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/tupitsin88/antiplagiat/file-storing-service/internal/models"
	"github.com/tupitsin88/antiplagiat/file-storing-service/internal/storage"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Ограничиваем размер загрузки 10MB
	r.ParseMultipartForm(10 << 20)

	studentName := r.FormValue("student_name")
	assignmentName := r.FormValue("assignment_name")
	if studentName == "" || assignmentName == "" {
		http.Error(w, "Fields student_name and assignment_name are required", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Если файл формата не .txt, то возвращаем ошибку
	filename := fileHeader.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".txt") {
		http.Error(w, "Only .txt files are allowed", http.StatusBadRequest)
		return
	}

	// Сохраняем метаданные в PostgreSQL
	var workID int
	query := "INSERT INTO works (student_name, assignment_name) VALUES ($1, $2) RETURNING id"
	err = storage.DB.QueryRow(query, studentName, assignmentName).Scan(&workID)
	if err != nil {
		log.Printf("DB Insert Error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Загружаем файл в MinIO
	ctx := context.Background()
	bucketName := os.Getenv("S3_BUCKET")
	objectName := fmt.Sprintf("%d.txt", workID)
	contentType := "text/plain"
	_, err = storage.MinioClient.PutObject(ctx, bucketName, objectName, file, fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		log.Printf("MinIO Upload Error: %v", err)
		http.Error(w, "Failed to upload file to storage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     workID,
		"status": "uploaded",
		"file":   objectName,
	})
}

// GetFileHandler скачивает файл из S3-хранилища
func GetFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	objectName := fmt.Sprintf("%s.txt", idStr)
	bucketName := os.Getenv("S3_BUCKET")
	object, err := storage.MinioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		http.Error(w, "File object not found", http.StatusNotFound)
		return
	}
	defer object.Close()
	stat, err := object.Stat()
	if err != nil {
		http.Error(w, "File not found in storage", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", objectName))
	w.Header().Set("Content-Type", stat.ContentType)
	if _, err := io.Copy(w, object); err != nil {
		log.Println("Stream Error:", err)
	}
}

// GetHandler получение метаданных о работе
func GetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var work models.Work
	err = storage.DB.QueryRow(
		"SELECT id, student_name, assignment_name, uploaded_at FROM works WHERE id = $1", id,
	).Scan(&work.ID, &work.StudentName, &work.AssignmentName, &work.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Work not found", http.StatusNotFound)
		} else {
			log.Printf("DB Error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(work)
}

func GetAllWorksHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := storage.DB.Query("SELECT id, student_name, assignment_name, uploaded_at FROM works")
	if err != nil {
		log.Printf("DB Query Error: %v", err)
		http.Error(w, "Failed to fetch works", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var works []models.Work
	for rows.Next() {
		var work models.Work
		if err := rows.Scan(&work.ID, &work.StudentName, &work.AssignmentName, &work.UploadedAt); err != nil {
			log.Printf("Scan Error: %v", err)
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		works = append(works, work)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(works)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
