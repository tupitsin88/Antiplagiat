package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tupitsin88/antiplagiat/file-analysis-service/internal/models"
	"github.com/tupitsin88/antiplagiat/file-analysis-service/internal/storage"
)

func getFileServiceURL() string {
	val := os.Getenv("FILE_SERVICE_URL")
	if val == "" {
		return "http://file-storing-service:8081"
	}
	return val
}

func generateWordCloudURL(text string) string {
	limit := 3000
	runes := []rune(text)
	if len(runes) > limit {
		text = string(runes[:limit])
	}

	baseURL := "https://quickchart.io/wordcloud"
	params := url.Values{}
	params.Add("text", text)
	params.Add("format", "png")
	params.Add("fontFamily", "Arial")
	params.Add("width", "1000")
	params.Add("height", "1000")
	params.Add("fontScale", "20")
	params.Add("scale", "linear")
	params.Add("background", "white")
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// downloadWithRetry выполняет HTTP запрос с повторными попытками
func downloadWithRetry(url string) ([]byte, error) {
	var err error
	for i := 0; i < 3; i++ {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, reqErr := client.Get(url)
		if reqErr == nil {
			if resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				return io.ReadAll(resp.Body)
			}
			resp.Body.Close()
			err = fmt.Errorf("status code %d", resp.StatusCode)
		} else {
			err = reqErr
		}
		time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
	}
	return nil, err
}

// downloadWorkContent скачивает текст работы по ID из File Storing Service с ретраями
func downloadWorkContent(id int) (string, error) {
	downloadURL := fmt.Sprintf("%s/download/%d", getFileServiceURL(), id)
	content, err := downloadWithRetry(downloadURL)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// Получает метаданные работы, чтобы узнать assignment_name
func getWorkMetadata(id int) (string, string, error) {
	url := fmt.Sprintf("%s/get/%d", getFileServiceURL(), id)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var meta struct {
		StudentName    string `json:"student_name"`
		AssignmentName string `json:"assignment_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", "", err
	}
	return meta.StudentName, meta.AssignmentName, nil
}

// Алгоритм сравнения: пересечение слов / слова текущего текста
func calculateSimilarity(text1, text2 string) float32 {
	t1 := strings.ToLower(text1)
	t2 := strings.ToLower(text2)
	words1 := strings.Fields(t1)
	words2 := strings.Fields(t2)

	if len(words1) == 0 {
		return 0.0
	}
	if len(words2) == 0 {
		return 0.0
	}
	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}
	intersection := 0
	for _, w := range words2 {
		if set1[w] {
			intersection++
		}
	}

	score := (float32(intersection) / float32(len(words1))) * 100.0
	if score > 100.0 {
		return 100.0
	}
	return score
}

// Основная логика проверки с Graceful Degradation
func checkPlagiatLogic(workID int) (float32, string, string, error) {
	currentText, err := downloadWorkContent(workID)
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to download source work: %v", err)
	}

	_, assignmentName, err := getWorkMetadata(workID)
	if err != nil {
		// Если метаданные недоступны, просто возвращаем результат без сравнения
		log.Printf("Metadata unavailable for work %d: %v", workID, err)
		return 0, currentText, "metadata_error", nil
	}
	rows, err := storage.DB.Query(
		"SELECT id, student_name FROM works WHERE assignment_name = $1 AND id != $2",
		assignmentName, workID,
	)
	if err != nil {
		// Если БД недоступна, возвращаем результат без сравнения
		log.Printf("DB error getting neighbors: %v", err)
		return 0, currentText, "db_error", nil
	}
	defer rows.Close()

	var maxScore float32 = 0.0
	var sources []string
	errorsCount := 0
	for rows.Next() {
		var other models.WorkData
		if err := rows.Scan(&other.ID, &other.StudentName); err != nil {
			continue
		}

		// Если не скачалась чужая работа — не падаем, а пропускаем
		otherText, err := downloadWorkContent(other.ID)
		if err != nil {
			log.Printf("Warning: Failed to download work %d for comparison: %v. Skipping.", other.ID, err)
			errorsCount++
			continue
		}
		score := calculateSimilarity(currentText, otherText)
		if score > 50.0 {
			sources = append(sources, fmt.Sprintf("%s (id:%d, %.1f%%)", other.StudentName, other.ID, score))
		}
		if score > maxScore {
			maxScore = score
		}
	}
	plagiatSources := "internet"
	if len(sources) > 0 {
		plagiatSources = strings.Join(sources, ", ")
	} else if maxScore < 10.0 {
		plagiatSources = "none"
	}
	if errorsCount > 0 {
		plagiatSources += fmt.Sprintf(" (skipped %d due to errors)", errorsCount)
	}
	return maxScore, currentText, plagiatSources, nil
}

func CheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workID, err := strconv.Atoi(vars["work_id"])
	if err != nil {
		http.Error(w, "Invalid work ID", http.StatusBadRequest)
		return
	}

	var existingID int
	err = storage.DB.QueryRow("SELECT id FROM plagiat_reports WHERE work_id = $1", workID).Scan(&existingID)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "already_checked",
			"message":     "Work already checked",
			"analysis_id": existingID,
			"work_id":     workID,
		})
		return
	}

	score, text, sources, err := checkPlagiatLogic(workID)
	if err != nil {
		log.Printf("Analysis failed: %v", err)
		http.Error(w, "Analysis failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cloudURL := generateWordCloudURL(text)
	var newID int
	err = storage.DB.QueryRow(
		`INSERT INTO plagiat_reports (work_id, plagiat_score, plagiat_sources, word_cloud_url) 
         VALUES ($1, $2, $3, $4) RETURNING id`,
		workID, score, sources, cloudURL).Scan(&newID)

	if err != nil {
		log.Printf("DB Save Error: %v", err)
		http.Error(w, "DB Save Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"id":      newID,
		"work_id": workID,
	})
}

func GetReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	var report models.PlagiatReport
	err := storage.DB.QueryRow(
		`SELECT id, work_id, plagiat_score, plagiat_sources, COALESCE(word_cloud_url, ''), checked_at 
        FROM plagiat_reports WHERE id = $1`, id).Scan(
		&report.ID, &report.WorkID, &report.PlagiatScore, &report.PlagiatSources, &report.WordCloudURL, &report.CheckedAt)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
