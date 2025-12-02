package models

import "time"

type Work struct {
	ID             int       `json:"id"`
	StudentName    string    `json:"student_name"`
	AssignmentName string    `json:"assignment_name"`
	FileContent    string    `json:"file_content"`
	UploadedAt     time.Time `json:"uploaded_at"`
}
