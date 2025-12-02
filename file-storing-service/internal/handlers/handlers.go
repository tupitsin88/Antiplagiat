package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tupitsin88/antiplagiat/file-storing-service/internal/models"
	"github.com/tupitsin88/antiplagiat/file-storing-service/internal/storage"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	var data models.Work
	data.StudentName = r.FormValue("student_name")
	data.AssignmentName = r.FormValue("assignment_name")

	if data.StudentName == "" || data.AssignmentName == "" {
		http.Error(w, "Empty fields", http.StatusBadRequest)
		return
	}

	r.ParseMultipartForm(10 << 20) // 10MB максимум
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if fileHeader.Size > 10<<20 {
		log.Println("File too big")
		http.Error(w, "File too large (max 10MB)", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	var insertedID int
	err = storage.DB.QueryRow("INSERT INTO works (student_name, assignment_name, file_content) VALUES ($1, $2, $3) RETURNING id",
		data.StudentName, data.AssignmentName, fileData).Scan(&insertedID)

	if err != nil {
		log.Printf("DB error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("File uploaded: work_id=%d, student=%s, size=%d bytes", insertedID, data.StudentName, len(fileData))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"work_id": insertedID,
		"status":  "uploaded",
	})

}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
			log.Printf("getHandler DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(work)
}

func GetFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var fileContent []byte
	err = storage.DB.QueryRow(
		"SELECT file_content FROM works WHERE id = $1", id,
	).Scan(&fileContent)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Work not found", http.StatusNotFound)
		} else {
			log.Printf("GetFileHandler DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(fileContent)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
