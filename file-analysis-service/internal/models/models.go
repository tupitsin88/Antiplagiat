package models

import "time"

type PlagiatReport struct {
	ID             int       `json:"id"`
	WorkID         int       `json:"work_id"`
	PlagiatScore   float32   `json:"plagiat_score"`   // 0.0 - 100.0
	PlagiatSources string    `json:"plagiat_sources"` // JSON string or text
	WordCloudURL   string    `json:"word_cloud_url"`  // Ссылка на облако слов
	CheckedAt      time.Time `json:"checked_at"`
}

type CheckResponse struct {
	ReportID     int     `json:"report_id"`
	PlagiatScore float32 `json:"plagiat_score"`
	Status       string  `json:"status"` // "checked"
}

type WorkData struct {
	ID          int
	StudentName string
}
