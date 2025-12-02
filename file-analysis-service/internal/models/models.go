package models

import "time"

type Work struct {
	ID             int       `json:"id"`
	StudentName    string    `json:"student_name"`
	AssignmentName string    `json:"assignment_name"`
	UploadedAt     time.Time `json:"uploaded_at"`
}

type PlagiatReport struct {
	ID             int       `json:"id"`
	WorkID         int       `json:"work_id"`
	PlagiatScore   float32   `json:"plagiat_score"`   // 0.0 - 100.0
	PlagiatSources string    `json:"plagiat_sources"` // JSON string
	CheckedAt      time.Time `json:"checked_at"`
}

type CheckResponse struct {
	ReportID     int     `json:"report_id"`
	PlagiatScore float32 `json:"plagiat_score"`
	Status       string  `json:"status"` // "checked"
}
