package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tupitsin88/antiplagiat/file-analysis-service/internal/models"
	"github.com/tupitsin88/antiplagiat/file-analysis-service/internal/storage"
)

func CheckPlagiat(workID int) (float32, error) {
	resp, err := http.Get(fmt.Sprintf("http://file-storing-service:8081/get/%d", workID))
	if err != nil {
		return 0, fmt.Errorf("failed to get work metadata: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("File Service returned %d: %s", resp.StatusCode, string(body))
	}

	var work models.Work
	if err := json.NewDecoder(resp.Body).Decode(&work); err != nil {
		return 0, fmt.Errorf("failed to decode work: %v", err)
	}

	fileResp, err := http.Get(fmt.Sprintf("http://file-storing-service:8081/download/%d", workID))
	if err != nil {
		return 0, fmt.Errorf("failed to download file: %v", err)
	}
	defer fileResp.Body.Close()

	if fileResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(fileResp.Body)
		return 0, fmt.Errorf("File Service download returned %d: %s", fileResp.StatusCode, string(body))
	}

	fileContent, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read file content: %v", err)
	}

	score := calculatePlagiarism(string(fileContent))
	return score, nil
}

func calculatePlagiarism(fileText string) float32 {
	length := len([]rune(fileText))
	if length <= 5000 {
		return 0
	}

	excess := float32(length - 5000)
	score := (excess / 5000.0) * 100

	if score > 100 {
		score = 100
	}
	return score
}

func CheckHandler(w http.ResponseWriter, r *http.Request) {
	var report models.PlagiatReport
	vars := mux.Vars(r)
	idStr := vars["work_id"]
	var err error
	report.WorkID, err = strconv.Atoi(idStr)
	if err != nil {
		log.Println("Некорректный work_id")
		http.Error(w, "Invalid work_id", http.StatusBadRequest)
		return
	}

	var existingReport models.PlagiatReport
	err = storage.DB.QueryRow(
		"SELECT id, work_id, plagiat_score, plagiat_sources, checked_at FROM plagiat_reports WHERE work_id = $1",
		report.WorkID,
	).Scan(&existingReport.ID, &existingReport.WorkID, &existingReport.PlagiatScore, &existingReport.PlagiatSources, &existingReport.CheckedAt)

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            existingReport.ID,
			"work_id":       existingReport.WorkID,
			"plagiat_score": existingReport.PlagiatScore,
			"status":        "checked",
		})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("DB query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	report.PlagiatScore, err = CheckPlagiat(report.WorkID)
	if err != nil {
		log.Printf("CheckPlagiat error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var insertedID int
	err = storage.DB.QueryRow(
		"INSERT INTO plagiat_reports (work_id, plagiat_score, plagiat_sources) VALUES ($1, $2, $3) RETURNING id",
		report.WorkID, report.PlagiatScore, "empty").Scan(&insertedID)

	if err != nil {
		log.Printf("DB insert error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            insertedID,
		"work_id":       report.WorkID,
		"plagiat_score": report.PlagiatScore,
		"status":        "checked",
	})
}

func GetReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var plag models.PlagiatReport
	err = storage.DB.QueryRow(
		"SELECT id, work_id, plagiat_score, plagiat_sources, checked_at FROM plagiat_reports WHERE id = $1", id,
	).Scan(&plag.ID, &plag.WorkID, &plag.PlagiatScore, &plag.PlagiatSources, &plag.CheckedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Report not found", http.StatusNotFound)
		} else {
			log.Printf("GetReportHandler DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plag)
}

func HealthHandler(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
