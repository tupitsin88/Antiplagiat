package models

import "time"

type Work struct {
	ID             int       `json:"id"`
	StudentName    string    `json:"student_name"`
	AssignmentName string    `json:"assignment_name"`
	UploadedAt     time.Time `json:"uploaded_at"`
}
