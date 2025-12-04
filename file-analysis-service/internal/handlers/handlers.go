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
	resp, err := http.Get(fmt.Sprintf("http://localhost:8081/get/%d", workID))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var work models.Work
	json.NewDecoder(resp.Body).Decode(&work)

	fileResp, err := http.Get(fmt.Sprintf("http://localhost:8081/download/%d", workID))
	if err != nil {
		return 0, err
	}
	defer fileResp.Body.Close()

	fileContent, _ := io.ReadAll(fileResp.Body)
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

	idStr := r.FormValue("work_id")
	var err error
	report.WorkID, err = strconv.Atoi(idStr)
	if err != nil {
		log.Println("Некорретный work_id")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report.PlagiatScore, err = CheckPlagiat(report.WorkID)
	var insertedID int
	err = storage.DB.QueryRow("INSERT INTO (work_id, plagiat_score, plagiat_sources ) VALUES ($1, $2, $3) RETURNING id ",
		report.WorkID, report.PlagiatScore, report.PlagiatSources).Scan(&insertedID)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"work_id": report.WorkID,
		"status":  "checked",
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

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
